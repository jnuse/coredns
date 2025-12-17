#!/bin/bash
#
# HTTP DoH 测试脚本
# 测试用例: TC-01 和 TC-02
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UTILS_DIR="$PROJECT_ROOT/utils"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 测试结果记录
pass_test() {
    ((PASSED_TESTS++))
    ((TOTAL_TESTS++))
    log_info "✓ $1"
}

fail_test() {
    ((FAILED_TESTS++))
    ((TOTAL_TESTS++))
    log_error "✗ $1"
}

# 等待服务就绪
wait_for_service() {
    local host=$1
    local port=$2
    local max_wait=10
    local waited=0
    
    log_info "Waiting for service at $host:$port..."
    while ! nc -z "$host" "$port" 2>/dev/null; do
        if [ $waited -ge $max_wait ]; then
            log_error "Service not ready after ${max_wait}s"
            return 1
        fi
        sleep 1
        ((waited++))
    done
    log_info "Service is ready!"
    return 0
}

# ============================================
# TC-01: HTTP DoH 基础查询
# ============================================
test_tc01_http_doh_basic() {
    log_info "Running TC-01: HTTP DoH Basic Query"
    
    local doh_url="http://127.0.0.1:8053/dns-query"
    local test_domain="example.com"
    
    # 使用 Python 工具发送 DoH 请求
    local output
    output=$(python3 "$UTILS_DIR/dns_query.py" "$test_domain" --doh "$doh_url" 2>&1)
    local exit_code=$?
    
    echo "$output"
    
    if [ $exit_code -eq 0 ] && echo "$output" | grep -q "Response Code: 0"; then
        pass_test "TC-01: HTTP DoH query successful"
    else
        fail_test "TC-01: HTTP DoH query failed"
        echo "$output"
    fi
}

# ============================================
# TC-01 (Alternative): 使用 curl 测试
# ============================================
test_tc01_http_doh_curl() {
    log_info "Running TC-01 (curl): HTTP DoH with curl"
    
    # 生成 DNS 查询二进制
    local query_bin="/tmp/dns_query.bin"
    python3 -c "
import sys
sys.path.insert(0, '$UTILS_DIR')
from dns_query import DNSQuery
query = DNSQuery.build_query('example.com', 'A')
sys.stdout.buffer.write(query)
" > "$query_bin"
    
    # 使用 curl 发送请求
    local response
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -H "Content-Type: application/dns-message" \
        -H "Accept: application/dns-message" \
        --data-binary "@$query_bin" \
        "http://127.0.0.1:8053/dns-query" 2>&1)
    
    local http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
    
    if [ "$http_code" = "200" ]; then
        pass_test "TC-01 (curl): HTTP 200 OK received"
    else
        fail_test "TC-01 (curl): Expected HTTP 200, got $http_code"
    fi
    
    rm -f "$query_bin"
}

# ============================================
# TC-02: 混合协议共存测试
# ============================================
test_tc02_mixed_protocol() {
    log_info "Running TC-02: Mixed Protocol (HTTP + HTTPS)"
    
    local http_url="http://127.0.0.1:8053/dns-query"
    local https_url="https://127.0.0.1:8443/dns-query"
    local test_domain="example.com"
    
    # 测试 HTTP 端口
    log_info "Testing HTTP port 8053..."
    local http_result
    http_result=$(python3 "$UTILS_DIR/dns_query.py" "$test_domain" --doh "$http_url" 2>&1)
    
    if echo "$http_result" | grep -q "Response Code: 0"; then
        pass_test "TC-02: HTTP port (8053) works"
    else
        fail_test "TC-02: HTTP port (8053) failed"
    fi
    
    # 测试 HTTPS 端口 (注意: 需要有效的证书配置)
    log_info "Testing HTTPS port 8443..."
    log_warn "TC-02: HTTPS test skipped (requires TLS certificate setup)"
    # 如果配置了证书，可以取消下面的注释
    # https_result=$(python3 "$UTILS_DIR/dns_query.py" "$test_domain" --doh "$https_url" 2>&1)
    # if echo "$https_result" | grep -q "Response Code: 0"; then
    #     pass_test "TC-02: HTTPS port (8443) works"
    # else
    #     fail_test "TC-02: HTTPS port (8443) failed"
    # fi
}

# ============================================
# 主测试流程
# ============================================
main() {
    log_info "=========================================="
    log_info "HTTP DoH Test Suite"
    log_info "=========================================="
    
    # 检查依赖
    if ! command -v python3 &> /dev/null; then
        log_error "python3 not found. Please install Python 3."
        exit 1
    fi
    
    if ! command -v nc &> /dev/null; then
        log_warn "netcat (nc) not found. Service check may not work."
    fi
    
    # 等待 CoreDNS 服务启动
    if ! wait_for_service "127.0.0.1" "8053"; then
        log_error "CoreDNS HTTP DoH service not running on port 8053"
        log_error "Please start CoreDNS with: coredns -conf configs/Corefile.http_doh"
        exit 1
    fi
    
    # 运行测试
    test_tc01_http_doh_basic
    test_tc01_http_doh_curl
    test_tc02_mixed_protocol
    
    # 输出测试结果
    log_info "=========================================="
    log_info "Test Summary"
    log_info "=========================================="
    log_info "Total Tests: $TOTAL_TESTS"
    log_info "Passed: $PASSED_TESTS"
    log_info "Failed: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        log_info "All tests passed! ✓"
        exit 0
    else
        log_error "Some tests failed. ✗"
        exit 1
    fi
}

main "$@"
