package rewrite

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const (
	// IPMatch matches the ip answer rewrite
	IPMatch = "ip"
)

// ipRewriterResponseRule rewrites IP addresses in A and AAAA records based on domain name matching
type ipRewriterResponseRule struct {
	// 域名到 IPv4 地址的映射
	domainToIPv4 map[string]net.IP
	// 域名到 IPv6 地址的映射
	domainToIPv6 map[string]net.IP
	// 重写后的目标域名（用于查找 IP）
	rewrittenName string
	// 默认 IPv4 地址（当没有具体域名映射时使用）
	defaultIPv4 net.IP
	// 默认 IPv6 地址（当没有具体域名映射时使用）
	defaultIPv6 net.IP
}

// newIPRewriterResponseRule creates a new IP rewriter rule
func newIPRewriterResponseRule() *ipRewriterResponseRule {
	return &ipRewriterResponseRule{
		domainToIPv4: make(map[string]net.IP),
		domainToIPv6: make(map[string]net.IP),
	}
}

// addMapping adds a domain to IP mapping
func (r *ipRewriterResponseRule) addMapping(ip, domain string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// 规范化域名
	normalizedDomain := plugin.Name(domain).Normalize()

	// 检查是 IPv4 还是 IPv6
	if parsedIP.To4() != nil {
		r.domainToIPv4[normalizedDomain] = parsedIP
	} else {
		r.domainToIPv6[normalizedDomain] = parsedIP
	}

	return nil
}

// loadFromHostsFile loads domain to IP mappings from a standard hosts file
// Format: IP domain [domain...]
func (r *ipRewriterResponseRule) loadFromHostsFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open hosts file %s: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 去除行内注释
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		// 解析行: 格式为 "IP domain [domain...]" 或 "IP *" (默认IP)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := fields[0]
		// 检查是否是默认 IP (使用 * 或 default 关键字)
		if len(fields) == 2 && (fields[1] == "*" || fields[1] == "default") {
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return fmt.Errorf("line %d: invalid IP address: %s", lineNum, ip)
			}
			// 根据 IP 类型设置默认值
			if parsedIP.To4() != nil {
				r.defaultIPv4 = parsedIP
			} else {
				r.defaultIPv6 = parsedIP
			}
			continue
		}

		// 第一个字段后的所有字段都是域名
		for _, domain := range fields[1:] {
			if err := r.addMapping(ip, domain); err != nil {
				return fmt.Errorf("line %d: %v", lineNum, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading hosts file: %v", err)
	}

	return nil
}

// RewriteResponse implements the ResponseRule interface
func (r *ipRewriterResponseRule) RewriteResponse(res *dns.Msg, rr dns.RR) {
	// 使用重写后的域名进行 IP 查找（如果有），否则使用原始域名
	domain := r.rewrittenName
	if domain == "" {
		domain = rr.Header().Name
	}

	switch rr.Header().Rrtype {
	case dns.TypeA:
		// 检查是否有该域名的 IPv4 映射，没有则使用默认IP
		if newIP, ok := r.domainToIPv4[domain]; ok {
			aRecord := rr.(*dns.A)
			aRecord.A = newIP
		} else if r.defaultIPv4 != nil {
			aRecord := rr.(*dns.A)
			aRecord.A = r.defaultIPv4
		}
	case dns.TypeAAAA:
		// 检查是否有该域名的 IPv6 映射，没有则使用默认IP
		if newIP, ok := r.domainToIPv6[domain]; ok {
			aaaaRecord := rr.(*dns.AAAA)
			aaaaRecord.AAAA = newIP
		} else if r.defaultIPv6 != nil {
			aaaaRecord := rr.(*dns.AAAA)
			aaaaRecord.AAAA = r.defaultIPv6
		}
	}
}

// Len returns the total number of domain to IP mappings
func (r *ipRewriterResponseRule) Len() int {
	return len(r.domainToIPv4) + len(r.domainToIPv6)
}
