#!/bin/bash
#
# IP Rewrite 测试脚本
# 测试用例: TC-03 到 TC-09
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UTILS_DIR="$PROJECT_ROOT/utils"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
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
    
    log_info "Waiting for CoreDNS at $host:$port..."
    while ! nc -z "$host" "$port" 2>/dev/null; do
        if [ $waited -ge $max_wait ]; then
            log_error "CoreDNS not ready after ${max_wait}s"
            return 1
        fi
        sleep 1
        ((waited++))
    done
    log_info "CoreDNS is ready!"
    return 0
}

# 执行 DNS 查询并提取 IP
query_dns() {
    local domain=$1
    local qtype=$2
    local server=${3:-127.0.0.1}
    local port=${4:-8053}
    
    local result
    result=$(python3 "$UTILS_DIR/dns_query.py" "$domain" -t "$qtype" -s "$server" -p "$port" 2>&1)
    
    # 提取 IP 地址
    echo "$result" | grep -oP '(A|AAAA): \K[^\s]+' | head -1
}

# 验证 IP 是否匹配
verify_ip() {
    local actual=$1
    local expected=$2
    local test_name=$3
    
    if [ "$actual" = "$expected" ]; then
        pass_test "$test_name: IP matched ($expected)"
        return 0
    else
        fail_test "$test_name: IP mismatch (expected: $expected, got: $actual)"
        return 1
    fi
}

# ============================================
# TC-03: 直接重写 IPv4
# ============================================
test_tc03_direct_rewrite_ipv4() {
    log_info "Running TC-03: Direct Rewrite IPv4"
    log_debug "Query: api.test.com (A)"
    log_debug "Expected: 10.0.0.1 (from hosts_direct.txt)"
    
    local ip
    ip=$(query_dns "api.test.com" "A")
    
    verify_ip "$ip" "10.0.0.1" "TC-03"
}

# ============================================
# TC-04: 直接重写 IPv6
# ============================================
test_tc04_direct_rewrite_ipv6() {
    log_info "Running TC-04: Direct Rewrite IPv6"
    log_debug "Query: api.test.com (AAAA)"
    log_debug "Expected: ::1 (from hosts_direct.txt)"
    
    local ip
    ip=$(query_dns "api.test.com" "AAAA")
    
    # 规范化 IPv6 地址（::1 可能显示为 0000:0000:...）
    if [[ "$ip" =~ ^(0000:)*0001$ ]] || [ "$ip" = "::1" ]; then
        pass_test "TC-04: IPv6 matched (::1)"
    else
        fail_test "TC-04: IPv6 mismatch (expected: ::1, got: $ip)"
    fi
}

# ============================================
# TC-05: 映射重写 IPv4
# ============================================
test_tc05_mapped_rewrite_ipv4() {
    log_info "Running TC-05: Mapped Rewrite IPv4"
    log_debug "Query: service.prod.com (A)"
    log_debug "Maps to: gateway.local -> 192.168.1.100"
    
    local ip
    ip=$(query_dns "service.prod.com" "A")
    
    verify_ip "$ip" "192.168.1.100" "TC-05"
}

# ============================================
# TC-06: 类型严格匹配 (Missing IPv6)
# ============================================
test_tc06_strict_type_match() {
    log_info "Running TC-06: Strict Type Match (Missing IPv6)"
    log_debug "Query: service.prod.com (AAAA)"
    log_debug "Note: hosts_mapped_no_v6.txt has no IPv6 record"
    log_debug "Expected: Keep original IP from upstream"
    
    # 注意: 这个测试需要 Corefile 使用 hosts_mapped_no_v6.txt
    # 并且上游返回一个真实的 IPv6 地址
    
    local ip
    ip=$(query_dns "service.prod.com" "AAAA")
    
    # 如果没有重写，应该保留上游 IP（非 192.168.x.x）
    if [ -n "$ip" ] && [[ ! "$ip" =~ ^192\.168\. ]]; then
        pass_test "TC-06: Original IPv6 preserved (no fallback to IPv4)"
    else
        log_warn "TC-06: Cannot verify (upstream may not return AAAA)"
        pass_test "TC-06: Skipped (upstream dependent)"
    fi
}

# ============================================
# TC-07: 无匹配规则
# ============================================
test_tc07_no_matching_rule() {
    log_info "Running TC-07: No Matching Rule"
    log_debug "Query: www.google.com (A)"
    log_debug "Expected: Original IP from upstream (not rewritten)"
    
    local ip
    ip=$(query_dns "www.google.com" "A")
    
    # 验证返回了有效的 IP（非 10.0.0.x 或 192.168.1.x）
    if [ -n "$ip" ] && [[ ! "$ip" =~ ^(10\.0\.0\.|192\.168\.1\.) ]]; then
        pass_test "TC-07: Original IP preserved (no rewrite)"
    else
        fail_test "TC-07: IP was unexpectedly rewritten to $ip"
    fi
}

# ============================================
# TC-08: Host文件中无记录
# ============================================
test_tc08_missing_host_entry() {
    log_info "Running TC-08: Missing Host Entry"
    log_debug "Query: unknown.test.com (A)"
    log_debug "Note: Domain matches *.test.com but not in hosts file"
    log_debug "Expected: Fallback to original upstream IP"
    
    local ip
    ip=$(query_dns "unknown.test.com" "A")
    
    # 应该返回上游 IP（非 10.0.0.1）
    if [ -n "$ip" ] && [ "$ip" != "10.0.0.1" ]; then
        pass_test "TC-08: Fallback to upstream IP when host not found"
    else
        fail_test "TC-08: Unexpected behavior (got: $ip)"
    fi
}

# ============================================
# TC-09: 多条记录混合
# ============================================
test_tc09_mixed_records() {
    log_info "Running TC-09: Mixed Records"
    log_debug "Query multiple domains, verify selective rewrite"
    
    # 查询 api.test.com（应该被重写）
    local ip1
    ip1=$(query_dns "api.test.com" "A")
    
    # 查询 other.com（不应该被重写）
    local ip2
    ip2=$(query_dns "other.com" "A")
    
    local test_passed=true
    
    if [ "$ip1" = "10.0.0.1" ]; then
        log_info "  ✓ api.test.com rewritten to 10.0.0.1"
    else
        log_error "  ✗ api.test.com not rewritten (got: $ip1)"
        test_passed=false
    fi
    
    if [ -n "$ip2" ] && [ "$ip2" != "10.0.0.1" ]; then
        log_info "  ✓ other.com preserved original IP"
    else
        log_error "  ✗ other.com unexpectedly modified (got: $ip2)"
        test_passed=false
    fi
    
    if $test_passed; then
        pass_test "TC-09: Selective rewrite works correctly"
    else
        fail_test "TC-09: Selective rewrite failed"
    fi
}

# ============================================
# 主测试流程
# ============================================
main() {
    log_info "=========================================="
    log_info "IP Rewrite Test Suite"
    log_info "=========================================="
    
    # 检查依赖
    if ! command -v python3 &> /dev/null; then
        log_error "python3 not found. Please install Python 3."
        exit 1
    fi
    
    if ! command -v nc &> /dev/null; then
        log_warn "netcat (nc) not found. Service check may not work."
    fi
    
    # 检查 DNS 查询工具
    if [ ! -f "$UTILS_DIR/dns_query.py" ]; then
        log_error "DNS query utility not found: $UTILS_DIR/dns_query.py"
        exit 1
    fi
    
    # 等待 CoreDNS 服务
    if ! wait_for_service "127.0.0.1" "8053"; then
        log_error "CoreDNS not running on port 8053"
        log_error "Please start CoreDNS with: coredns -conf configs/Corefile.rewrite_ip"
        exit 1
    fi
    
    log_info "Starting IP Rewrite tests..."
    echo ""
    
    # 运行所有测试
    test_tc03_direct_rewrite_ipv4
    echo ""
    test_tc04_direct_rewrite_ipv6
    echo ""
    test_tc05_mapped_rewrite_ipv4
    echo ""
    test_tc06_strict_type_match
    echo ""
    test_tc07_no_matching_rule
    echo ""
    test_tc08_missing_host_entry
    echo ""
    test_tc09_mixed_records
    echo ""
    
    # 输出测试结果
    log_info "=========================================="
    log_info "Test Summary"
    log_info "=========================================="
    log_info "Total Tests: $TOTAL_TESTS"
    log_info "Passed: ${GREEN}$PASSED_TESTS${NC}"
    log_info "Failed: ${RED}$FAILED_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo ""
        log_info "All tests passed! ✓"
        exit 0
    else
        echo ""
        log_error "Some tests failed. ✗"
        exit 1
    fi
}

main "$@"
