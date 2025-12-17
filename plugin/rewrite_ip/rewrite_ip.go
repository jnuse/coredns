package rewrite_ip

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

// RewriteIP 实现 IP 重写插件
type RewriteIP struct {
	Next  plugin.Handler
	Rules []Rule
}

// Rule 定义一条重写规则
type Rule struct {
	Patterns []string  // 通配符模式列表
	MapTo    string    // 映射目标域名（可选）
	HostFile *HostFile // hosts 文件数据
}

// ServeDNS 实现 plugin.Handler 接口
func (r RewriteIP) ServeDNS(ctx context.Context, w dns.ResponseWriter, req *dns.Msg) (int, error) {
	// 使用自定义 ResponseWriter 拦截响应
	rw := &ResponseRewriter{
		ResponseWriter: w,
		rules:          r.Rules,
	}

	// 调用下一个插件（如 forward）
	return plugin.NextOrFailure(r.Name(), r.Next, ctx, rw, req)
}

// Name 返回插件名称
func (r RewriteIP) Name() string { return "rewrite_ip" }

// ResponseRewriter 包装 ResponseWriter 以拦截响应
type ResponseRewriter struct {
	dns.ResponseWriter
	rules []Rule
}

// WriteMsg 拦截并修改 DNS 响应
func (rw *ResponseRewriter) WriteMsg(res *dns.Msg) error {
	if res == nil || len(res.Answer) == 0 {
		return rw.ResponseWriter.WriteMsg(res)
	}

	// 遍历所有答案记录
	for i, rr := range res.Answer {
		switch record := rr.(type) {
		case *dns.A:
			// 处理 A 记录（IPv4）
			recordName := record.Header().Name
			if newIP := rw.findReplacementIPv4(recordName); newIP != nil {
				record.A = newIP
				res.Answer[i] = record
			}

		case *dns.AAAA:
			// 处理 AAAA 记录（IPv6）
			recordName := record.Header().Name
			if newIP := rw.findReplacementIPv6(recordName); newIP != nil {
				record.AAAA = newIP
				res.Answer[i] = record
			}
		}
	}

	return rw.ResponseWriter.WriteMsg(res)
}

// findReplacementIPv4 查找 IPv4 替换地址
func (rw *ResponseRewriter) findReplacementIPv4(recordName string) []byte {
	for _, rule := range rw.rules {
		// 检查是否匹配规则
		if !matchAny(recordName, rule.Patterns) {
			continue
		}

		// 确定查询域名
		lookupDomain := recordName
		if rule.MapTo != "" {
			lookupDomain = rule.MapTo
		}

		// 查询 hosts 文件
		if ips := rule.HostFile.LookupIPv4(lookupDomain); len(ips) > 0 {
			// 返回第一个 IPv4 地址（转换为 4 字节）
			ip4 := ips[0].To4()
			return ip4
		}
	}
	return nil
}

// findReplacementIPv6 查找 IPv6 替换地址
func (rw *ResponseRewriter) findReplacementIPv6(recordName string) []byte {
	for _, rule := range rw.rules {
		// 检查是否匹配规则
		if !matchAny(recordName, rule.Patterns) {
			continue
		}

		// 确定查询域名
		lookupDomain := recordName
		if rule.MapTo != "" {
			lookupDomain = rule.MapTo
		}

		// 查询 hosts 文件
		if ips := rule.HostFile.LookupIPv6(lookupDomain); len(ips) > 0 {
			// 返回第一个 IPv6 地址（16 字节）
			ip6 := ips[0].To16()
			return ip6
		}
	}
	return nil
}

// matchAny 检查域名是否匹配任一模式
func matchAny(domain string, patterns []string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	
	for _, pattern := range patterns {
		if matchPattern(domain, pattern) {
			return true
		}
	}
	return false
}

// matchPattern 实现通配符匹配
func matchPattern(domain, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSuffix(pattern, "."))
	
	// 精确匹配
	if domain == pattern {
		return true
	}

	// 通配符匹配 (*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // 去掉 *，保留 .example.com
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}

	// 使用 filepath.Match 进行更复杂的通配符匹配
	matched, _ := filepath.Match(pattern, domain)
	return matched
}
