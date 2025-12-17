# doh

## Name

*doh* - enables DNS-over-HTTPS (DoH) for CoreDNS server.

## Description

The *doh* plugin configures CoreDNS to serve DNS queries over HTTP or HTTPS, following the DNS-over-HTTPS (DoH) specification (RFC 8484).

By default, DoH runs over **plain HTTP** (without TLS encryption). This is suitable for:
- Local development and testing
- Internal networks behind a reverse proxy (like nginx or Caddy) that handles TLS termination
- Service mesh sidecar scenarios where TLS is handled by the mesh

For production use over the internet, you should enable TLS using the `tls` directive.

## Syntax

```corefile
doh [PATH] {
    tls [selfsigned|CERT KEY]
}
```

- **PATH**: The HTTP path for DoH queries. Default is `/dns-query` (RFC 8484 standard).
- **tls**: Optional. Enables HTTPS (TLS encryption).
  - `selfsigned`: Automatically generates a self-signed certificate (for testing only).
  - `CERT KEY`: Paths to TLS certificate and private key files.

If `tls` is not specified, the server will run in **plain HTTP mode** (no encryption).

## Examples

### HTTP DoH (No TLS)

Suitable for local development or behind a TLS-terminating proxy:

```corefile
.:8053 {
    doh /dns-query
    forward . 8.8.8.8
    log
}
```

Test with:
```bash
curl -H "accept: application/dns-message" \
     --data-binary @query.bin \
     http://localhost:8053/dns-query
```

### HTTPS DoH with Self-Signed Certificate

For testing HTTPS locally:

```corefile
.:8443 {
    doh /dns-query {
        tls selfsigned
    }
    forward . 8.8.8.8
    log
}
```

### HTTPS DoH with Real Certificate

For production use:

```corefile
.:443 {
    doh /dns-query {
        tls /path/to/cert.pem /path/to/key.pem
    }
    forward . 8.8.8.8
    log
}
```

### Custom DoH Path

Use a custom path instead of the standard `/dns-query`:

```corefile
.:8053 {
    doh /my-dns-endpoint
    forward . 1.1.1.1
}
```

### Mixed Protocols

Run both HTTP and HTTPS DoH on different ports:

```corefile
# HTTP DoH for internal use
.:8053 {
    doh /dns-query
    forward . 8.8.8.8
    log
}

# HTTPS DoH for external clients
.:8443 {
    doh /dns-query {
        tls /etc/coredns/cert.pem /etc/coredns/key.pem
    }
    forward . 8.8.8.8
    log
}
```

## Behind a Reverse Proxy

When running behind a reverse proxy (nginx, Caddy, etc.) that handles TLS termination, use plain HTTP mode:

**CoreDNS config:**
```corefile
.:8053 {
    doh /dns-query
    forward . 8.8.8.8
}
```

**Nginx config:**
```nginx
server {
    listen 443 ssl http2;
    server_name dns.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location /dns-query {
        proxy_pass http://localhost:8053/dns-query;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }
}
```

## Client Testing

### Using curl

Generate a DNS query:
```bash
# Query for example.com A record
echo -n "00 00 01 00 00 01 00 00 00 00 00 00 07 65 78 61 6d 70 6c 65 03 63 6f 6d 00 00 01 00 01" | \
  xxd -r -p > query.bin
```

Send via HTTP POST:
```bash
curl -H "Content-Type: application/dns-message" \
     --data-binary @query.bin \
     http://localhost:8053/dns-query
```

### Using Python

```python
import requests
import dns.message

# Create DNS query
query = dns.message.make_query('example.com', 'A')
query_bytes = query.to_wire()

# Send DoH request
response = requests.post(
    'http://localhost:8053/dns-query',
    headers={'Content-Type': 'application/dns-message'},
    data=query_bytes
)

# Parse response
answer = dns.message.from_wire(response.content)
print(answer)
```

## Security Considerations

⚠️ **Warning**: Plain HTTP DoH does not encrypt DNS queries. Use it only in trusted environments:
- Local development
- Internal networks
- Behind a TLS-terminating reverse proxy

For public-facing servers, **always use `tls`** with valid certificates.

## See Also

- [RFC 8484 - DNS Queries over HTTPS (DoH)](https://datatracker.ietf.org/doc/html/rfc8484)
- [CoreDNS TLS Plugin](https://coredns.io/plugins/tls/)
- [Mozilla DoH Documentation](https://wiki.mozilla.org/Trusted_Recursive_Resolver)
