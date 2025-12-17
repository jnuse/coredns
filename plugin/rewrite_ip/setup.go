package rewrite_ip

import (
	"fmt"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register("rewrite_ip", setup)
}

func setup(c *caddy.Controller) error {
	rules, err := parseRewriteIP(c)
	if err != nil {
		return plugin.Error("rewrite_ip", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return RewriteIP{Next: next, Rules: rules}
	})

	return nil
}

func parseRewriteIP(c *caddy.Controller) ([]Rule, error) {
	var rules []Rule

	for c.Next() {
		rule := Rule{
			Patterns: []string{},
		}

		// 解析块内容
		for c.NextBlock() {
			switch c.Val() {
			case "match":
				// match pattern1 pattern2 ...
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				rule.Patterns = append(rule.Patterns, args...)

			case "map_to":
				// map_to domain.com
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				rule.MapTo = c.Val()

			case "hosts":
				// hosts /path/to/file
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				hostsPath := c.Val()

				// 加载 hosts 文件
				hostFile, err := NewHostFile(hostsPath)
				if err != nil {
					return nil, fmt.Errorf("failed to load hosts file %s: %v", hostsPath, err)
				}
				rule.HostFile = hostFile

			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}

		// 验证规则完整性
		if len(rule.Patterns) == 0 {
			return nil, c.Err("rewrite_ip: 'match' is required")
		}
		if rule.HostFile == nil {
			return nil, c.Err("rewrite_ip: 'hosts' is required")
		}

		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return nil, c.Err("rewrite_ip: no rules defined")
	}

	return rules, nil
}
