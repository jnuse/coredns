package rewrite

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestIPRewriterResponseRule(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		domain         string
		recordType     uint16
		recordDomain   string
		originalIP     string
		expectedIP     string
		shouldRewrite  bool
	}{
		{
			name:          "IPv4 A record rewrite - domain match",
			ip:            "10.0.0.1",
			domain:        "example.com",
			recordType:    dns.TypeA,
			recordDomain:  "example.com.",
			originalIP:    "192.168.1.1",
			expectedIP:    "10.0.0.1",
			shouldRewrite: true,
		},
		{
			name:          "IPv4 A record - domain no match",
			ip:            "10.0.0.1",
			domain:        "example.com",
			recordType:    dns.TypeA,
			recordDomain:  "other.com.",
			originalIP:    "192.168.1.1",
			expectedIP:    "192.168.1.1",
			shouldRewrite: false,
		},
		{
			name:          "IPv4 A record - original IP doesn't matter",
			ip:            "10.0.0.1",
			domain:        "example.com",
			recordType:    dns.TypeA,
			recordDomain:  "example.com.",
			originalIP:    "8.8.8.8",
			expectedIP:    "10.0.0.1",
			shouldRewrite: true,
		},
		{
			name:          "IPv6 AAAA record rewrite - domain match",
			ip:            "2001:db8:1::1",
			domain:        "example.com",
			recordType:    dns.TypeAAAA,
			recordDomain:  "example.com.",
			originalIP:    "2001:db8::1",
			expectedIP:    "2001:db8:1::1",
			shouldRewrite: true,
		},
		{
			name:          "IPv6 AAAA record - domain no match",
			ip:            "2001:db8:1::1",
			domain:        "example.com",
			recordType:    dns.TypeAAAA,
			recordDomain:  "other.com.",
			originalIP:    "2001:db8::1",
			expectedIP:    "2001:db8::1",
			shouldRewrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := newIPRewriterResponseRule()
			err := rule.addMapping(tt.ip, tt.domain)
			if err != nil {
				t.Fatalf("addMapping failed: %v", err)
			}

			msg := new(dns.Msg)
			var rr dns.RR

			if tt.recordType == dns.TypeA {
				a := &dns.A{
					Hdr: dns.RR_Header{
						Name:   tt.recordDomain,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    300,
					},
					A: parseIPv4(tt.originalIP),
				}
				rr = a
			} else if tt.recordType == dns.TypeAAAA {
				aaaa := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   tt.recordDomain,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    300,
					},
					AAAA: parseIPv6(tt.originalIP),
				}
				rr = aaaa
			}

			rule.RewriteResponse(msg, rr)

			var resultIP string
			if tt.recordType == dns.TypeA {
				resultIP = rr.(*dns.A).A.String()
			} else if tt.recordType == dns.TypeAAAA {
				resultIP = rr.(*dns.AAAA).AAAA.String()
			}

			if resultIP != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, resultIP)
			}
		})
	}
}

func TestIPRewriterInvalidMappings(t *testing.T) {
	tests := []struct {
		name   string
		ip     string
		domain string
	}{
		{
			name:   "invalid IP",
			ip:     "invalid",
			domain: "example.com",
		},
		{
			name:   "empty domain",
			ip:     "192.168.1.1",
			domain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := newIPRewriterResponseRule()
			err := rule.addMapping(tt.ip, tt.domain)
			if err == nil && tt.ip == "invalid" {
				t.Errorf("expected error for invalid IP, got nil")
			}
		})
	}
}

func TestIPRewriterMultipleMappings(t *testing.T) {
	rule := newIPRewriterResponseRule()

	// 添加多个域名到 IP 的映射
	mappings := map[string]string{
		"10.0.0.1": "server1.example.com",
		"10.0.0.2": "server2.example.com",
		"10.0.0.3": "server3.example.com",
	}

	for ip, domain := range mappings {
		if err := rule.addMapping(ip, domain); err != nil {
			t.Fatalf("failed to add mapping %s -> %s: %v", domain, ip, err)
		}
	}

	if rule.Len() != len(mappings) {
		t.Errorf("expected %d mappings, got %d", len(mappings), rule.Len())
	}

	// 测试每个映射
	for expectedIP, domain := range mappings {
		msg := new(dns.Msg)
		a := &dns.A{
			Hdr: dns.RR_Header{
				Name:   domain + ".",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: parseIPv4("192.168.1.1"), // 原始 IP 不重要
		}

		rule.RewriteResponse(msg, a)

		if a.A.String() != expectedIP {
			t.Errorf("mapping %s -> %s failed, got %s", domain, expectedIP, a.A.String())
		}
	}
}

func TestIPRewriterLoadFromFile(t *testing.T) {
	rule := newIPRewriterResponseRule()
	err := rule.loadFromHostsFile("testdata/ip-mappings.txt")
	if err != nil {
		t.Fatalf("failed to load hosts file: %v", err)
	}

	// 应该加载多个映射（包括带别名的）
	if rule.Len() == 0 {
		t.Error("expected mappings from file, got 0")
	}

	// 测试其中一个 IPv4 映射
	msg := new(dns.Msg)
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   "server1.example.com.",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		A: parseIPv4("1.2.3.4"), // 原始 IP 不重要，会被替换
	}

	rule.RewriteResponse(msg, a)

	if a.A.String() != "10.0.0.10" {
		t.Errorf("expected 10.0.0.10, got %s", a.A.String())
	}

	// 测试带别名的域名
	a2 := &dns.A{
		Hdr: dns.RR_Header{
			Name:   "server2-alias.example.com.",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		A: parseIPv4("5.6.7.8"),
	}

	rule.RewriteResponse(msg, a2)

	if a2.A.String() != "10.0.0.20" {
		t.Errorf("expected 10.0.0.20 for alias, got %s", a2.A.String())
	}

	// 测试其中一个 IPv6 映射
	aaaa := &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   "ipv6server1.example.com.",
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		AAAA: parseIPv6("2001:db8::999"), // 原始 IP 会被替换
	}

	rule.RewriteResponse(msg, aaaa)

	if aaaa.AAAA.String() != "2001:db8:1::1" {
		t.Errorf("expected 2001:db8:1::1, got %s", aaaa.AAAA.String())
	}
}

func TestIPRewriterFileNotFound(t *testing.T) {
	rule := newIPRewriterResponseRule()
	err := rule.loadFromHostsFile("nonexistent-file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestIPRewriterPreservesOtherFields(t *testing.T) {
	// 测试重写 IP 时保留其他字段（TTL、Class等）
	rule := newIPRewriterResponseRule()
	rule.addMapping("10.0.0.1", "example.com")

	msg := new(dns.Msg)
	originalTTL := uint32(600)
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   "example.com.",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    originalTTL,
		},
		A: parseIPv4("192.168.1.1"),
	}

	rule.RewriteResponse(msg, a)

	// 验证 IP 被重写
	if a.A.String() != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", a.A.String())
	}

	// 验证其他字段保持不变
	if a.Hdr.Ttl != originalTTL {
		t.Errorf("expected TTL %d, got %d", originalTTL, a.Hdr.Ttl)
	}
	if a.Hdr.Class != dns.ClassINET {
		t.Errorf("expected Class INET, got %d", a.Hdr.Class)
	}
	if a.Hdr.Name != "example.com." {
		t.Errorf("expected Name example.com., got %s", a.Hdr.Name)
	}
}

// Helper functions
func parseIPv4(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic("invalid IP: " + s)
	}
	return ip.To4()
}

func parseIPv6(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic("invalid IP: " + s)
	}
	// 确保返回 16 字节的 IPv6 地址
	if ip.To4() != nil {
		panic("not an IPv6 address: " + s)
	}
	return ip
}
