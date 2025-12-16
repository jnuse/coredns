package https

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const (
	dnsMessageMimeType = "application/dns-message"

	// typical Ethernet MTU (1500 bytes) - min IP header size (20 bytes) - UDP header (8 bytes).
	// It seems like a reasonable limitation for DoH protocol.
	// However, if you know RFCs that specify this limit, update it.
	maxDNSMessageSize     = 1472
	defaultRequestTimeout = 2 * time.Second
)

var (
	dnsMessageMimeTypeHeader = []string{dnsMessageMimeType}

	errResponseTooLarge = errors.New("dns response size is too large")
	errResponseStatus   = errors.New("invalid http response status code")
)

// dnsClient is the client API for DNS service
type dnsClient interface {
	Query(ctx context.Context, dnsreq []byte) (result *dns.Msg, err error)
}

// newDoHDNSClient creates a new instance of dohDNSClient service.
// url must be a full URL to send DoH requests to like "https://example.com/dns-query"
func newDoHDNSClient(client httpRequestDoer, url string) *dohDNSClient {
	return &dohDNSClient{client, url}
}

type httpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// dohDNSClient is a DNS client that proxies requests to the upstream server using DoH protocol.
type dohDNSClient struct {
	client httpRequestDoer
	url    string
}

func (c *dohDNSClient) Query(ctx context.Context, dnsreq []byte) (r *dns.Msg, err error) {
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(dnsreq)); err != nil {
		return
	}
	req.Header["Accept"] = dnsMessageMimeTypeHeader
	req.Header["Content-Type"] = dnsMessageMimeTypeHeader

	var resp *http.Response
	if resp, err = c.client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	// RFC8484 Section 4.2.1:
	// A successful HTTP response with a 2xx status code is used for any valid DNS response,
	// regardless of the DNS response code.
	// HTTP responses with non-successful HTTP status codes do not contain
	// replies to the original DNS question in the HTTP request.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errResponseStatus
	}

	// limit the number of bytes read to avoid potential DoS attacks.
	// it would be better to add (*dns.Msg) Unpack(io.Reader) method to avoid byte slice allocation
	var body []byte
	if body, err = io.ReadAll(io.LimitReader(resp.Body, maxDNSMessageSize+1)); err != nil {
		return
	}
	if len(body) > maxDNSMessageSize {
		return nil, errResponseTooLarge
	}
	r = new(dns.Msg)
	err = r.Unpack(body)
	return
}

type metricDNSClient struct {
	fails  uint32 // atomic counter for health checking
	client dnsClient
	addr   string
}

func newMetricDNSClient(client dnsClient, addr string) *metricDNSClient {
	return &metricDNSClient{client: client, addr: addr}
}

func (c *metricDNSClient) Query(ctx context.Context, dnsreq []byte) (r *dns.Msg, err error) {
	start := time.Now()

	// decorator pattern
	if r, err = c.client.Query(ctx, dnsreq); err != nil {
		// Increment failure counter for health checking
		c.incrementFails()
		return
	}

	// Reset failure counter on successful query
	atomic.StoreUint32(&c.fails, 0)

	rc, ok := dns.RcodeToString[r.Rcode]
	if !ok {
		rc = strconv.Itoa(r.Rcode)
	}

	RequestCount.WithLabelValues(c.addr).Add(1)
	RcodeCount.WithLabelValues(rc, c.addr).Add(1)
	RequestDuration.WithLabelValues(c.addr).Observe(time.Since(start).Seconds())
	return
}

// incrementFails increments the number of fails safely.
func (c *metricDNSClient) incrementFails() {
	curVal := atomic.LoadUint32(&c.fails)
	if curVal > curVal+1 {
		// overflow occurred, do not update the counter again
		return
	}
	atomic.AddUint32(&c.fails, 1)
}

// Down returns true if this upstream is down, i.e. has *more* fails than maxfails.
func (c *metricDNSClient) Down(maxfails uint32) bool {
	if maxfails == 0 {
		return false
	}
	fails := atomic.LoadUint32(&c.fails)
	return fails > maxfails
}

// Fails returns the current fail count.
func (c *metricDNSClient) Fails() uint32 {
	return atomic.LoadUint32(&c.fails)
}

func newLoadBalanceDNSClient(clients []*metricDNSClient, opts ...lbDNSClientOption) *lbDNSClient {
	c := &lbDNSClient{
		p:        newRandomPolicy(),
		maxFails: 2, // default maxFails, same as forward plugin
		timeout:  defaultRequestTimeout,
		clients:  clients,
	}
	// option pattern
	for _, o := range opts {
		o(c)
	}
	return c
}

// lbDNSClient is a DNS client that load balances DNS requests between the list of DNS clients.
type lbDNSClient struct {
	p        policy
	timeout  time.Duration
	maxFails uint32
	clients  []*metricDNSClient
}

type lbDNSClientOption func(c *lbDNSClient)

func withLbPolicy(p policy) lbDNSClientOption {
	return func(c *lbDNSClient) {
		c.p = p
	}
}

func withLbRequestTimeout(timeout time.Duration) lbDNSClientOption {
	return func(c *lbDNSClient) {
		c.timeout = timeout
	}
}

func withLbMaxFails(maxFails uint32) lbDNSClientOption {
	return func(c *lbDNSClient) {
		c.maxFails = maxFails
	}
}

func (c *lbDNSClient) Query(ctx context.Context, dnsreq []byte) (r *dns.Msg, err error) {
	ids := c.p.List(len(c.clients))
	attempts := 0
	allDown := true

	// Try each upstream in policy order, skipping those that are down
	for _, idx := range ids {
		client := c.clients[idx]
		
		// Skip upstreams that are marked as down
		if client.Down(c.maxFails) {
			continue
		}
		
		allDown = false
		attempts++
		
		if r, err = c.query(ctx, dnsreq, idx); err == nil {
			return
		}
	}

	// If all upstreams are down, try them anyway (healthcheck might be broken)
	if allDown {
		HealthcheckBrokenCount.Add(1)
		// Try the first upstream regardless of health status
		if len(c.clients) > 0 {
			r, err = c.query(ctx, dnsreq, ids[0])
		}
	}
	return
}

func (c *lbDNSClient) query(ctx context.Context, dnsreq []byte, clientID int) (*dns.Msg, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.clients[clientID].Query(ctx, dnsreq)
}
