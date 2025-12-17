package doh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	pkgtls "github.com/coredns/coredns/plugin/pkg/tls"
)

func init() { plugin.Register("doh", setup) }

// generateSelfSignedCert creates a self-signed TLS certificate
func generateSelfSignedCert() (*tls.Config, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"CoreDNS"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return tlsConfig, nil
}

func setup(c *caddy.Controller) error {
	config := dnsserver.GetConfig(c)

	for c.Next() {
		// doh [path]
		args := c.RemainingArgs()
		path := "/dns-query"
		if len(args) > 1 {
			return plugin.Error("doh", c.ArgErr())
		}
		if len(args) == 1 {
			path = args[0]
		}

		var tlsConfig *tls.Config

		// Parse block
		for c.NextBlock() {
			switch c.Val() {
			case "tls":
				args := c.RemainingArgs()
				
				if len(args) == 0 {
					return plugin.Error("doh", c.Errf("tls requires arguments: 'selfsigned' or 'cert key'"))
				}

				if len(args) == 1 && args[0] == "selfsigned" {
					// Generate self-signed certificate
					var err error
					tlsConfig, err = generateSelfSignedCert()
					if err != nil {
						return plugin.Error("doh", c.Errf("failed to generate self-signed certificate: %v", err))
					}
				} else {
					// Load certificate from files
					certFile := args[0]
					keyFile := ""
					if len(args) >= 2 {
						keyFile = args[1]
					}
					
					// Handle relative paths
					root := config.Root
					if root != "" {
						if !filepath.IsAbs(certFile) {
							certFile = filepath.Join(root, certFile)
						}
						if keyFile != "" && !filepath.IsAbs(keyFile) {
							keyFile = filepath.Join(root, keyFile)
						}
					}

					var err error
					if keyFile != "" {
						// tls cert key
						tlsConfig, err = pkgtls.NewTLSConfigFromArgs(certFile, keyFile)
					} else {
						// tls ca_cert (for client authentication)
						tlsConfig, err = pkgtls.NewTLSConfigFromArgs(certFile)
					}
					
					if err != nil {
						return plugin.Error("doh", c.Errf("failed to load TLS config: %v", err))
					}
				}

			default:
				return plugin.Error("doh", c.Errf("unknown property '%s'", c.Val()))
			}
		}

		// Configure HTTP/2 for TLS connections
		if tlsConfig != nil {
			tlsConfig.NextProtos = []string{"h2", "http/1.1"}
			config.TLSConfig = tlsConfig
		}
		// If tlsConfig is nil, HTTP DoH will be used (no TLS)

		// Set HTTP request validation function
		config.HTTPRequestValidateFunc = func(r *http.Request) bool {
			return r.URL.Path == path
		}

		// Mark transport as HTTPS (works for both HTTP and HTTPS)
		config.Transport = "https"
	}

	return nil
}
