package zendesk

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Error an error type containing the http response from zendesk
type Error struct {
	body []byte
	resp *http.Response
}

// Error the error string for this error
func (e Error) Error() string {
	msg := string(e.body)
	if msg == "" {
		msg = http.StatusText(e.Status())
	}

	return fmt.Sprintf("%d: %s", e.resp.StatusCode, msg)
}

// Body is the Body of the HTTP response
func (e Error) Body() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBuffer(e.body))
}

// Headers the HTTP headers returned from zendesk
func (e Error) Headers() http.Header {
	return e.resp.Header
}

// Status the HTTP status code returned from zendesk
func (e Error) Status() int {
	return e.resp.StatusCode
}

// OptionsError is an error type for invalid option argument.
type OptionsError struct {
	opts interface{}
}

func (e *OptionsError) Error() string {
	return fmt.Sprintf("invalid options: %v", e.opts)
}
