package tls

import (
	ctls "crypto/tls"
	"path/filepath"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/tls"
)

func init() { plugin.Register("tls", setup) }

func setup(c *caddy.Controller) error {
	err := parseTLS(c)
	if err != nil {
		return plugin.Error("tls", err)
	}
	return nil
}

func parseTLS(c *caddy.Controller) error {
	config := dnsserver.GetConfig(c)

	if config.TLSConfig != nil {
		return plugin.Error("tls", c.Errf("TLS already configured for this server instance"))
	}

	for c.Next() {
		args := c.RemainingArgs()
		
		// Check if this is a block-only configuration (e.g., for allow_http_doh)
		if len(args) == 0 && c.NextBlock() {
			switch c.Val() {
			case "allow_http_doh":
				// Allow DNS-over-HTTPS to run without TLS (plain HTTP)
				// This is for local/internal use only
				if len(c.RemainingArgs()) != 0 {
					return c.ArgErr()
				}
				config.AllowHTTP = true
				// Don't set TLSConfig, allowing HTTP DoH
				return nil
			default:
				return c.Errf("tls requires certificate and key arguments, or use 'allow_http_doh' for HTTP DoH")
			}
		}
		
		// Normal TLS configuration with certificate files
		if len(args) < 2 || len(args) > 3 {
			return plugin.Error("tls", c.ArgErr())
		}
		clientAuth := ctls.NoClientCert
		for c.NextBlock() {
			switch c.Val() {
			case "client_auth":
				authTypeArgs := c.RemainingArgs()
				if len(authTypeArgs) != 1 {
					return c.ArgErr()
				}
				switch authTypeArgs[0] {
				case "nocert":
					clientAuth = ctls.NoClientCert
				case "request":
					clientAuth = ctls.RequestClientCert
				case "require":
					clientAuth = ctls.RequireAnyClientCert
				case "verify_if_given":
					clientAuth = ctls.VerifyClientCertIfGiven
				case "require_and_verify":
					clientAuth = ctls.RequireAndVerifyClientCert
				default:
					return c.Errf("unknown authentication type '%s'", authTypeArgs[0])
				}
			case "allow_http_doh":
				return c.Errf("allow_http_doh must be used without certificate arguments")
			default:
				return c.Errf("unknown option '%s'", c.Val())
			}
		}
		for i := range args {
			if !filepath.IsAbs(args[i]) && config.Root != "" {
				args[i] = filepath.Join(config.Root, args[i])
			}
		}
		tls, err := tls.NewTLSConfigFromArgs(args...)
		if err != nil {
			return err
		}
		tls.ClientAuth = clientAuth
		// NewTLSConfigFromArgs only sets RootCAs, so we need to let ClientCAs refer to it.
		tls.ClientCAs = tls.RootCAs

		config.TLSConfig = tls
	}
	return nil
}
