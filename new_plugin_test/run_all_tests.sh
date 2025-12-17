#!/bin/bash
#
# 主测试运行脚本
# 运行所有测试套件
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SCRIPTS_DIR="$PROJECT_ROOT/scripts"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

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

log_section() {
    echo -e "${CYAN}=========================================="
    echo -e "$1"
    echo -e "==========================================${NC}"
}

# 显示使用说明
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run CoreDNS plugin test suites

OPTIONS:
    --http-doh          Run only HTTP DoH tests (TC-01, TC-02)
    --rewrite-ip        Run only IP Rewrite tests (TC-03 to TC-09)
    --all               Run all tests (default)
    -h, --help          Show this help message

EXAMPLES:
    # Run all tests
    $0 --all
    
    # Run only HTTP DoH tests
    $0 --http-doh
    
    # Run only IP Rewrite tests
    $0 --rewrite-ip

PREREQUISITES:
    - CoreDNS binary built and available
    - Python 3 installed
    - netcat (nc) installed for service checks

EOF
}

# 检查依赖
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=false
    
    if ! command -v python3 &> /dev/null; then
        log_error "python3 not found"
        missing_deps=true
    else
        log_info "✓ Python 3 found: $(python3 --version)"
    fi
    
    if ! command -v nc &> /dev/null; then
        log_warn "netcat (nc) not found - service checks may not work"
    else
        log_info "✓ netcat found"
    fi
    
    if ! command -v coredns &> /dev/null; then
        log_warn "coredns not in PATH - you may need to specify full path"
    else
        log_info "✓ CoreDNS found: $(coredns -version 2>&1 | head -1)"
    fi
    
    if $missing_deps; then
        log_error "Missing required dependencies"
        return 1
    fi
    
    log_info "All dependencies satisfied"
    return 0
}

# 运行 HTTP DoH 测试
run_http_doh_tests() {
    log_section "Running HTTP DoH Tests"
    
    if [ ! -f "$SCRIPTS_DIR/test_http_doh.sh" ]; then
        log_error "Test script not found: $SCRIPTS_DIR/test_http_doh.sh"
        return 1
    fi
    
    chmod +x "$SCRIPTS_DIR/test_http_doh.sh"
    
    if bash "$SCRIPTS_DIR/test_http_doh.sh"; then
        log_info "HTTP DoH tests: ${GREEN}PASSED${NC}"
        return 0
    else
        log_error "HTTP DoH tests: ${RED}FAILED${NC}"
        return 1
    fi
}

# 运行 IP Rewrite 测试
run_rewrite_ip_tests() {
    log_section "Running IP Rewrite Tests"
    
    if [ ! -f "$SCRIPTS_DIR/test_rewrite_ip.sh" ]; then
        log_error "Test script not found: $SCRIPTS_DIR/test_rewrite_ip.sh"
        return 1
    fi
    
    chmod +x "$SCRIPTS_DIR/test_rewrite_ip.sh"
    
    if bash "$SCRIPTS_DIR/test_rewrite_ip.sh"; then
        log_info "IP Rewrite tests: ${GREEN}PASSED${NC}"
        return 0
    else
        log_error "IP Rewrite tests: ${RED}FAILED${NC}"
        return 1
    fi
}

# 主函数
main() {
    local run_http_doh=false
    local run_rewrite_ip=false
    local run_all=true
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --http-doh)
                run_http_doh=true
                run_all=false
                shift
                ;;
            --rewrite-ip)
                run_rewrite_ip=true
                run_all=false
                shift
                ;;
            --all)
                run_all=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 如果没有指定特定测试，运行所有测试
    if $run_all; then
        run_http_doh=true
        run_rewrite_ip=true
    fi
    
    log_section "CoreDNS Plugin Test Suite"
    log_info "Project Root: $PROJECT_ROOT"
    echo ""
    
    # 检查依赖
    if ! check_dependencies; then
        exit 1
    fi
    echo ""
    
    local failed_suites=0
    local total_suites=0
    
    # 运行选定的测试套件
    if $run_http_doh; then
        ((total_suites++))
        if ! run_http_doh_tests; then
            ((failed_suites++))
        fi
        echo ""
    fi
    
    if $run_rewrite_ip; then
        ((total_suites++))
        if ! run_rewrite_ip_tests; then
            ((failed_suites++))
        fi
        echo ""
    fi
    
    # 最终结果
    log_section "Overall Test Results"
    log_info "Test Suites Run: $total_suites"
    
    if [ $failed_suites -eq 0 ]; then
        log_info "Result: ${GREEN}ALL PASSED ✓${NC}"
        exit 0
    else
        log_error "Failed Suites: ${RED}$failed_suites${NC}"
        log_error "Result: ${RED}SOME TESTS FAILED ✗${NC}"
        exit 1
    fi
}

main "$@"
