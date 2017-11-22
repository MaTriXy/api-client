package apiclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Client may be used to make requests to the designated API. When implementing your actual API client, include
// an instance of *Client inside your own client struct.
type Client struct {
	httpClient        *http.Client
	apiKeyValue       string
	apiKeyName        string
	baseURL           string
	requestsPerSecond int
	rateLimiter       chan int
}

// ClientOption is the type of constructor options for NewClient(...).
type ClientOption func(*Client) error

// the default rate limit
var defaultRequestsPerSecond = 10

// NewClient constructs a new Client which can make requests to the designated API.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{requestsPerSecond: defaultRequestsPerSecond}
	WithHTTPClient(&http.Client{})(c)
	for _, option := range options {
		err := option(c)
		if err != nil {
			return nil, err
		}
	}

	// Implement a bursty rate limiter.
	// Allow up to 1 second worth of requests to be made at once.
	c.rateLimiter = make(chan int, c.requestsPerSecond)
	// Prefill rateLimiter with 1 seconds worth of requests.
	for i := 0; i < c.requestsPerSecond; i++ {
		c.rateLimiter <- 1
	}
	go func() {
		// Wait a second for pre-filled quota to drain
		time.Sleep(time.Second)
		// Then, refill rateLimiter continuously
		for _ = range time.Tick(time.Second / time.Duration(c.requestsPerSecond)) {
			c.rateLimiter <- 1
		}
	}()

	return c, nil
}

// WithHTTPClient configures a Maps API client with a http.Client to make requests over.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) error {
		if _, ok := c.Transport.(*transport); !ok {
			t := c.Transport
			if t != nil {
				c.Transport = &transport{Base: t}
			} else {
				c.Transport = &transport{Base: http.DefaultTransport}
			}
		}
		client.httpClient = c
		return nil
	}
}

// WithAPIKey configures a Maps API client with an API Key
func WithAPIKey(apiKeyName, apiKeyValue string) ClientOption {
	return func(c *Client) error {
		c.apiKeyName = apiKeyName
		c.apiKeyValue = apiKeyValue
		return nil
	}
}

// WithRateLimit configures the rate limit for back end requests.
// Default is to limit to 10 requests per second.
func WithRateLimit(requestsPerSecond int) ClientOption {
	return func(c *Client) error {
		c.requestsPerSecond = requestsPerSecond
		return nil
	}
}

// APIConfig configures the URL for the API endpoint
type APIConfig struct {
	Host string
	Path string
}

type apiRequest interface {
	Params() url.Values
}

func (c *Client) get(ctx context.Context, config *APIConfig, apiReq apiRequest) (*http.Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.rateLimiter:
		// Execute request.
	}

	host := config.Host
	if c.baseURL != "" {
		host = c.baseURL
	}
	req, err := http.NewRequest("GET", host+config.Path, nil)
	if err != nil {
		return nil, err
	}
	q := c.generateAuthQuery(config.Path, apiReq.Params())
	req.URL.RawQuery = q
	return ctxhttp.Do(ctx, c.httpClient, req)
}

// GetBinary returns JSON data from the API endpoint
func (c *Client) GetJSON(ctx context.Context, config *APIConfig, apiReq apiRequest, resp interface{}) error {
	httpResp, err := c.get(ctx, config, apiReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	return json.NewDecoder(httpResp.Body).Decode(resp)
}

type BinaryResponse struct {
	StatusCode  int
	ContentType string
	Data        io.ReadCloser
}

// GetBinary returns binary data from the API endpoint
func (c *Client) GetBinary(ctx context.Context, config *APIConfig, apiReq apiRequest) (BinaryResponse, error) {
	httpResp, err := c.get(ctx, config, apiReq)
	if err != nil {
		return BinaryResponse{}, err
	}

	return BinaryResponse{httpResp.StatusCode, httpResp.Header.Get("Content-Type"), httpResp.Body}, nil
}

func (c *Client) generateAuthQuery(path string, q url.Values) string {
	if c.apiKeyValue != "" {
		q.Set(c.apiKeyName, c.apiKeyValue)
		return q.Encode()
	}
	return q.Encode()
}
