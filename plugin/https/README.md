# https

[![Build Status](https://cloud.drone.io/api/badges/v-byte-cpu/coredns-https/status.svg)](https://cloud.drone.io/v-byte-cpu/coredns-https)
[![GoReportCard Status](https://goreportcard.com/badge/github.com/v-byte-cpu/coredns-https)](https://goreportcard.com/report/github.com/v-byte-cpu/coredns-https)
[![GitHub release](https://img.shields.io/github/v/release/v-byte-cpu/coredns-https)](https://github.com/v-byte-cpu/coredns-https/releases/latest)

**https** is a [CoreDNS](https://github.com/coredns/coredns) plugin that proxies DNS messages to upstream resolvers using DNS-over-HTTPS protocol. See [RFC 8484](https://tools.ietf.org/html/rfc8484).

## Installation

External CoreDNS plugins can be enabled in one of two ways:
  1. [Build with compile-time configuration file](https://coredns.io/2017/07/25/compile-time-enabling-or-disabling-plugins/#build-with-compile-time-configuration-file)
  2. [Build with external golang source code](https://coredns.io/2017/07/25/compile-time-enabling-or-disabling-plugins/#build-with-external-golang-source-code)

Method #1 can be quickly described using a sequence of the following commands:

```
git clone --depth 1 https://github.com/coredns/coredns.git
cd coredns
go get github.com/v-byte-cpu/coredns-https
echo "https:github.com/v-byte-cpu/coredns-https" >> plugin.cfg
go generate
go mod tidy -compat=1.17
go build
```

## Syntax

In its most basic form:

~~~
https FROM TO...
~~~

* **FROM** is the base domain to match for the request to be proxied.
* **TO...** are the destination endpoints to proxy to. The number of upstreams is
  limited to 15.

Multiple upstreams are randomized (see `policy`) on first use. When a proxy returns an error
the next upstream in the list is tried.

Extra knobs are available with an expanded syntax:

~~~
https FROM TO... {
    except IGNORED_NAMES...
    tls CERT KEY CA
    tls_servername NAME
    policy random|round_robin|sequential
    max_fails NUM
    timeout DURATION
}
~~~

* **FROM** and **TO...** as above.
* **IGNORED_NAMES** in `except` is a space-separated list of domains to exclude from proxying.
  Requests that match none of these names will be passed through.
* `tls` **CERT** **KEY** **CA** define the TLS properties for TLS connection. From 0 to 3 arguments can be
  provided with the meaning as described below

  * `tls` - no client authentication is used, and the system CAs are used to verify the server certificate (by default)
  * `tls` **CA** - no client authentication is used, and the file CA is used to verify the server certificate
  * `tls` **CERT** **KEY** - client authentication is used with the specified cert/key pair.
    The server certificate is verified with the system CAs
  * `tls` **CERT** **KEY**  **CA** - client authentication is used with the specified cert/key pair.
    The server certificate is verified using the specified CA file

* `policy` specifies the policy to use for selecting upstream servers. The default is `random`.
* `max_fails` is the number of subsequent failed health checks needed before considering an upstream to be down.
  If 0, the upstream will never be marked as down. Default is 2.
* `timeout` **DURATION** sets the timeout for each upstream request (e.g., `2s`, `500ms`). Default is `2s`.


## Metrics

If monitoring is enabled (via the *prometheus* plugin) then the following metrics are exported:

* `coredns_https_requests_total{to}` - total query count per upstream.
* `coredns_https_request_duration_seconds{to}` - histogram of request duration per upstream.
* `coredns_https_responses_total{to, rcode}` - counter of response codes per upstream.
* `coredns_https_healthcheck_broken_total{}` - counter of when all upstreams are unhealthy.

## Examples

Proxy all requests within `example.org.` to a DoH nameserver:

~~~ corefile
example.org {
    https . cloudflare-dns.com/dns-query
}
~~~

Forward everything except requests to `example.org`

~~~ corefile
. {
    https . dns.quad9.net/dns-query {
        except example.org
    }
}
~~~

Load balance all requests between multiple upstreams

~~~ corefile
. {
    https . dns.quad9.net/dns-query cloudflare-dns.com:443/dns-query dns.google/dns-query
}
~~~

Configure health checking and timeout

~~~ corefile
. {
    https . dns.quad9.net/dns-query cloudflare-dns.com/dns-query {
        max_fails 3
        timeout 5s
        policy round_robin
    }
}
~~~

Internal DoH server with custom TLS:

~~~ corefile
. {
    https . internal-doh.example.com/dns-query {
        tls /path/to/cert.pem /path/to/key.pem /path/to/ca.pem
        tls_servername internal-doh.example.com
    }
}
~~~

## Usage Guide

### Health Checking

The plugin automatically tracks failures for each upstream server. When an upstream fails `max_fails` consecutive times, it is marked as unhealthy and excluded from the rotation. The default is `max_fails 2`.

**Example:** Mark upstream down after 3 failures
~~~
https . dns.quad9.net/dns-query {
    max_fails 3
}
~~~

**Disable health checking** (always use all upstreams):
~~~
https . dns.quad9.net/dns-query {
    max_fails 0
}
~~~

### Timeout Configuration

Control how long to wait for upstream responses. Default is `2s`.

**Example:** Use 5-second timeout for slow networks
~~~
https . dns.quad9.net/dns-query {
    timeout 5s
}
~~~

**Example:** Fast timeout for local DoH server
~~~
https . localhost:8053/dns-query {
    timeout 500ms
}
~~~

### Load Balancing Policies

Choose how requests are distributed across upstreams:

- **random** (default): Randomly select an upstream for each request
- **round_robin**: Distribute requests evenly in circular order
- **sequential**: Always try upstreams in the configured order

**Example:** Round-robin for even distribution
~~~
https . dns1.example.com/dns-query dns2.example.com/dns-query {
    policy round_robin
}
~~~

**Example:** Sequential with fallback
~~~
https . primary-doh.example.com/dns-query backup-doh.example.com/dns-query {
    policy sequential
}
~~~

### Production Best Practices

**High Availability Setup:**
~~~
. {
    errors
    cache 300
    https . dns.quad9.net/dns-query cloudflare-dns.com/dns-query dns.google/dns-query {
        max_fails 2
        timeout 3s
        policy random
    }
    prometheus :9153
}
~~~

**Low Latency Setup:**
~~~
. {
    https . cloudflare-dns.com/dns-query {
        max_fails 1
        timeout 1s
    }
}
~~~

**Monitoring:**
Monitor these Prometheus metrics for health:
- `coredns_https_healthcheck_broken_total` - Alerts when all upstreams fail
- `coredns_https_request_duration_seconds` - Track upstream latency
- `coredns_https_responses_total` - Monitor error rates per upstream

### Common Issues

**All upstreams unhealthy:** When all upstreams are marked down, the plugin will still attempt to use them (random selection) to avoid complete service failure. Check the `healthcheck_broken_total` metric.

**High latency:** Increase the `timeout` value or check upstream DoH server performance.

**Uneven load distribution:** Use `policy round_robin` instead of `random` for more predictable distribution.
~~~

Internal DoH server:

~~~ corefile
. {
    https . 10.0.0.10:853/dns-query {
      tls ca.crt
      tls_servername internal.domain
    }
}
~~~