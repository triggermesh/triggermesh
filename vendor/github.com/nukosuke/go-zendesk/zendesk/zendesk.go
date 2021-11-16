package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-querystring/query"
)

const (
	baseURLFormat = "https://%s.zendesk.com/api/v2"
)

var defaultHeaders = map[string]string{
	"User-Agent":   "nukosuke/go-zendesk",
	"Content-Type": "application/json",
}

var subdomainRegexp = regexp.MustCompile("^[a-z0-9][a-z0-9-]+[a-z0-9]$")

type (
	// Client of Zendesk API
	Client struct {
		baseURL    *url.URL
		httpClient *http.Client
		credential Credential
		headers    map[string]string
	}

	// BaseAPI encapsulates base methods for zendesk client
	BaseAPI interface {
		Get(ctx context.Context, path string) ([]byte, error)
		Post(ctx context.Context, path string, data interface{}) ([]byte, error)
	}
)

// NewClient creates new Zendesk API client
func NewClient(httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &Client{httpClient: httpClient}
	client.headers = defaultHeaders
	return client, nil
}

// SetHeader saves HTTP header in client. It will be included all API request
func (z *Client) SetHeader(key string, value string) {
	z.headers[key] = value
}

// SetSubdomain saves subdomain in client. It will be used
// when call API
func (z *Client) SetSubdomain(subdomain string) error {
	if !subdomainRegexp.MatchString(subdomain) {
		return fmt.Errorf("%s is invalid subdomain", subdomain)
	}

	baseURLString := fmt.Sprintf(baseURLFormat, subdomain)
	baseURL, err := url.Parse(baseURLString)
	if err != nil {
		return err
	}

	z.baseURL = baseURL
	return nil
}

// SetEndpointURL replace full URL of endpoint without subdomain validation.
// This is mainly used for testing to point to mock API server.
func (z *Client) SetEndpointURL(newURL string) error {
	baseURL, err := url.Parse(newURL)
	if err != nil {
		return err
	}

	z.baseURL = baseURL
	return nil
}

// SetCredential saves credential in client. It will be set
// to request header when call API
func (z *Client) SetCredential(cred Credential) {
	z.credential = cred
}

// get get JSON data from API and returns its body as []bytes
func (z *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, z.baseURL.String()+path, nil)
	if err != nil {
		return nil, err
	}

	req = z.prepareRequest(ctx, req)

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, Error{
			body: body,
			resp: resp,
		}
	}
	return body, nil
}

// post send data to API and returns response body as []bytes
func (z *Client) post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, z.baseURL.String()+path, strings.NewReader(string(bytes)))
	if err != nil {
		return nil, err
	}

	req = z.prepareRequest(ctx, req)

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, Error{
			body: body,
			resp: resp,
		}
	}

	return body, nil
}

// put sends data to API and returns response body as []bytes
func (z *Client) put(ctx context.Context, path string, data interface{}) ([]byte, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, z.baseURL.String()+path, strings.NewReader(string(bytes)))
	if err != nil {
		return nil, err
	}

	req = z.prepareRequest(ctx, req)

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, Error{
			body: body,
			resp: resp,
		}
	}

	return body, nil
}

// delete sends data to API and returns an error if unsuccessful
func (z *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequest(http.MethodDelete, z.baseURL.String()+path, nil)
	if err != nil {
		return err
	}

	req = z.prepareRequest(ctx, req)

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return Error{
			body: body,
			resp: resp,
		}
	}

	return nil
}

// prepare request sets common request variables such as authn and user agent
func (z *Client) prepareRequest(ctx context.Context, req *http.Request) *http.Request {
	out := req.WithContext(ctx)
	z.includeHeaders(out)
	if z.credential != nil {
		out.SetBasicAuth(z.credential.Email(), z.credential.Secret())
	}

	return out
}

// includeHeaders set HTTP headers from client.headers to *http.Request
func (z *Client) includeHeaders(req *http.Request) {
	for key, value := range z.headers {
		req.Header.Set(key, value)
	}
}

// addOptions build query string
func addOptions(s string, opts interface{}) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// Get allows users to send requests not yet implemented
func (z *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return z.get(ctx, path)
}

// Post allows users to send requests not yet implemented
func (z *Client) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return z.post(ctx, path, data)
}
