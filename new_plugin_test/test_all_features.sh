#!/bin/bash
#
# 大一统功能测试脚本
# 测试 HTTP DoH + HTTPS DoH + rewrite_ip 的完整集成
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COREDNS_BIN="${SCRIPT_DIR}/../coredns"
CONFIG_FILE="${SCRIPT_DIR}/configs/Corefile.all"
UTILS_DIR="${SCRIPT_DIR}/utils"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${CYAN}[✓]${NC} $1"
}

log_section() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# 检查依赖
check_dependencies() {
    if [ ! -f "$COREDNS_BIN" ]; then
        log_error "CoreDNS binary not found at $COREDNS_BIN"
        log_error "Please build CoreDNS first: cd .. && go build"
        exit 1
    fi
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "Config file not found: $CONFIG_FILE"
        exit 1
    fi
    
    if ! command -v python3 &> /dev/null; then
        log_error "python3 not found"
        exit 1
    fi
    
    if ! command -v dig &> /dev/null; then
        log_error "dig not found"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl not found"
        exit 1
    fi
}

# 启动 CoreDNS
start_coredns() {
    log_info "Starting CoreDNS with unified config..."
    "$COREDNS_BIN" -conf "$CONFIG_FILE" > /tmp/coredns_all.log 2>&1 &
    COREDNS_PID=$!
    echo $COREDNS_PID > /tmp/coredns_all.pid
    
    # 等待启动
    sleep 3
    
    if ! kill -0 $COREDNS_PID 2>/dev/null; then
        log_error "CoreDNS failed to start"
        cat /tmp/coredns_all.log
        exit 1
    fi
    
    log_success "CoreDNS started (PID: $COREDNS_PID)"
    log_info "Listening on:"
    log_info "  - Standard DNS: 127.0.0.1:8053"
    log_info "  - HTTP DoH:     127.0.0.1:8080"
    log_info "  - HTTPS DoH:    127.0.0.1:8443"
}

# 停止 CoreDNS
stop_coredns() {
    if [ -f /tmp/coredns_all.pid ]; then
        COREDNS_PID=$(cat /tmp/coredns_all.pid)
        if kill -0 $COREDNS_PID 2>/dev/null; then
            log_info "Stopping CoreDNS (PID: $COREDNS_PID)..."
            kill $COREDNS_PID
            sleep 1
        fi
        rm -f /tmp/coredns_all.pid
    fi
}

# 测试标准 DNS (UDP)
test_standard_dns() {
    log_section "Test 1: Standard DNS (UDP:8053)"
    
    log_info "Query: example.com (should be rewritten to 10.0.0.1)"
    RESULT=$(dig @127.0.0.1 -p 8053 example.com +short | head -1)
    
    if [ "$RESULT" = "10.0.0.1" ]; then
        log_success "IP rewrite successful: example.com -> 10.0.0.1"
    else
        log_error "Expected 10.0.0.1, got: $RESULT"
        return 1
    fi
    
    log_info "Query: github.com (should NOT be rewritten)"
    RESULT=$(dig @127.0.0.1 -p 8053 github.com +short | head -1)
    
    if [ -n "$RESULT" ] && [ "$RESULT" != "10.0.0.1" ]; then
        log_success "Non-matching domain preserved: github.com -> $RESULT"
    else
        log_error "Unexpected result for github.com: $RESULT"
        return 1
    fi
}

# 测试 HTTP DoH
test_http_doh() {
    log_section "Test 2: HTTP DoH (Port 8080)"
    
    log_info "Query: example.com via HTTP DoH"
    RESULT=$(python3 "$UTILS_DIR/dns_query.py" example.com \
        --doh http://127.0.0.1:8080/dns-query 2>&1 | grep -A5 "Answers:" | grep "10.0.0.1" || true)
    
    if [ -n "$RESULT" ]; then
        log_success "HTTP DoH + IP rewrite working: example.com -> 10.0.0.1"
    else
        log_error "HTTP DoH test failed"
        python3 "$UTILS_DIR/dns_query.py" example.com --doh http://127.0.0.1:8080/dns-query
        return 1
    fi
}

# 测试 HTTPS DoH
test_https_doh() {
    log_section "Test 3: HTTPS DoH (Port 8443)"
    
    log_info "Query: example.com via HTTPS DoH (self-signed cert)"
    RESULT=$(python3 "$UTILS_DIR/dns_query.py" example.com \
        --doh https://127.0.0.1:8443/dns-query 2>&1 | grep -A5 "Answers:" | grep "10.0.0.1" || true)
    
    if [ -n "$RESULT" ]; then
        log_success "HTTPS DoH + IP rewrite working: example.com -> 10.0.0.1"
    else
        log_error "HTTPS DoH test failed"
        python3 "$UTILS_DIR/dns_query.py" example.com --doh https://127.0.0.1:8443/dns-query
        return 1
    fi
}

# 测试映射重写
test_mapped_rewrite() {
    log_section "Test 4: Mapped IP Rewrite"
    
    # 注意: 需要一个真实存在的 *.prod.com 域名来测试
    # 这里我们使用 example.com 作为替代，需要更新配置
    log_info "This test requires a real *.prod.com domain"
    log_info "Skipping mapped rewrite test (requires production domain)"
}

# 并发测试
test_concurrent() {
    log_section "Test 5: Concurrent Requests"
    
    log_info "Sending 10 concurrent requests to HTTP DoH..."
    
    for i in {1..10}; do
        python3 "$UTILS_DIR/dns_query.py" example.com \
            --doh http://127.0.0.1:8080/dns-query > /tmp/doh_test_$i.txt 2>&1 &
    done
    
    wait
    
    SUCCESS_COUNT=$(grep -l "10.0.0.1" /tmp/doh_test_*.txt 2>/dev/null | wc -l)
    
    if [ "$SUCCESS_COUNT" -ge 8 ]; then
        log_success "Concurrent test passed: $SUCCESS_COUNT/10 requests successful"
    else
        log_error "Concurrent test failed: only $SUCCESS_COUNT/10 successful"
        return 1
    fi
    
    rm -f /tmp/doh_test_*.txt
}

# 性能测试
test_performance() {
    log_section "Test 6: Performance Benchmark"
    
    log_info "Running 100 queries via standard DNS..."
    START=$(date +%s%N)
    
    for i in {1..100}; do
        dig @127.0.0.1 -p 8053 example.com +short > /dev/null 2>&1
    done
    
    END=$(date +%s%N)
    DURATION=$(( ($END - $START) / 1000000 ))
    QPS=$(( 100000 / $DURATION ))
    
    log_success "100 queries completed in ${DURATION}ms (~${QPS} QPS)"
}

# 主函数
main() {
    echo -e "${CYAN}"
    echo "╔════════════════════════════════════════╗"
    echo "║   CoreDNS 大一统功能测试套件         ║"
    echo "║   DoH + rewrite_ip 集成测试          ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${NC}\n"
    
    # 清理环境
    trap stop_coredns EXIT
    pkill -9 coredns 2>/dev/null || true
    
    # 检查依赖
    check_dependencies
    
    # 启动 CoreDNS
    start_coredns
    
    # 运行测试
    PASSED=0
    FAILED=0
    
    if test_standard_dns; then ((PASSED++)); else ((FAILED++)); fi
    if test_http_doh; then ((PASSED++)); else ((FAILED++)); fi
    if test_https_doh; then ((PASSED++)); else ((FAILED++)); fi
    # test_mapped_rewrite  # 跳过
    if test_concurrent; then ((PASSED++)); else ((FAILED++)); fi
    if test_performance; then ((PASSED++)); else ((FAILED++)); fi
    
    # 总结
    log_section "Test Summary"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    echo -e "${RED}Failed: $FAILED${NC}"
    
    if [ $FAILED -eq 0 ]; then
        echo -e "\n${CYAN}🎉 All tests passed!${NC}\n"
        exit 0
    else
        echo -e "\n${RED}❌ Some tests failed${NC}\n"
        exit 1
    fi
}

main "$@"
