// Package doh implements DNS-over-HTTPS (DoH) for CoreDNS.
// It allows CoreDNS to serve DNS queries over HTTP or HTTPS.
package doh

import (
	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// DoH is a plugin that enables DNS-over-HTTPS support.
// The actual HTTP/HTTPS serving is handled by core/dnsserver/server_https.go
// This plugin is only responsible for configuration parsing.
type DoH struct {
	Next plugin.Handler
}

// ServeDNS implements the plugin.Handler interface.
// For DoH, the actual DNS message handling is done by the ServerHTTPS.ServeHTTP method.
// This is a pass-through plugin that does nothing in the chain.
func (d DoH) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(d.Name(), d.Next, ctx, w, r)
}

// Name implements the Handler interface.
func (d DoH) Name() string { return "doh" }
