package splunk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// Event represents the log event object that is sent to Splunk when Client.Log is called.
type Event struct {
	Time       EventTime   `json:"time"`                 // when the event happened
	Host       string      `json:"host"`                 // hostname
	Source     string      `json:"source,omitempty"`     // optional description of the source of the event; typically the app's name
	SourceType string      `json:"sourcetype,omitempty"` // optional name of a Splunk parsing configuration; this is usually inferred by Splunk
	Index      string      `json:"index,omitempty"`      // optional name of the Splunk index to store the event in; not required if the token has a default index set in Splunk
	Event      interface{} `json:"event"`                // throw any useful key/val pairs here
}

// EventTime marshals timestamps using the Splunk HTTP Event Collector's default format.
type EventTime struct {
	time.Time
}

func (t EventTime) MarshalJSON() ([]byte, error) {
	// The milliseconds are truncated, not rounded to nearest; eg. 12:00:00.5008274 will be logged as 12:00:00.500.
	return []byte(fmt.Sprintf("%d.%d", t.Unix(), t.Nanosecond()/1e6)), nil
}

// Client manages communication with Splunk's HTTP Event Collector.
// New client objects should be created using the NewClient function.
//
// The URL field must be defined and pointed at a Splunk servers Event Collector port (i.e. https://{your-splunk-URL}:8088/services/collector).
// The Token field must be defined with your access token to the Event Collector.
type Client struct {
	HTTPClient *http.Client // HTTP client used to communicate with the API
	URL        string
	Hostname   string
	Token      string
	Source     string //Default source
	SourceType string //Default source type
	Index      string //Default index
}

// NewClient creates a new client to Splunk.
// This should be the primary way a Splunk client object is constructed.
//
// If an httpClient object is specified it will be used instead of the
// default http.DefaultClient.
func NewClient(httpClient *http.Client, URL string, Token string, Source string, SourceType string, Index string) *Client {
	// Create a new client
	if httpClient == nil {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}}
		httpClient = &http.Client{Timeout: time.Second * 20, Transport: tr}
	}
	hostname, _ := os.Hostname()
	c := &Client{
		HTTPClient: httpClient,
		URL:        URL,
		Hostname:   hostname,
		Token:      Token,
		Source:     Source,
		SourceType: SourceType,
		Index:      Index,
	}
	return c
}

// NewEvent creates a new log event to send to Splunk.
// This should be the primary way a Splunk log object is constructed, and is automatically called by the Log function attached to the client.
// This method takes the current timestamp for the event, meaning that the event is generated at runtime.
func (c *Client) NewEvent(event interface{}, source string, sourcetype string, index string) *Event {
	e := &Event{
		Time:       EventTime{time.Now()},
		Host:       c.Hostname,
		Source:     source,
		SourceType: sourcetype,
		Index:      index,
		Event:      event,
	}
	return e
}

// NewEventWithTime creates a new log event with a specified timetamp to send to Splunk.
// This is similar to NewEvent but if you want to log in a different time rather than time.Now this becomes handy. If that's
// the case, use this function to create the Event object and the the LogEvent function.
func (c *Client) NewEventWithTime(t time.Time, event interface{}, source string, sourcetype string, index string) *Event {
	e := &Event{
		Time:       EventTime{time.Now()},
		Host:       c.Hostname,
		Source:     source,
		SourceType: sourcetype,
		Index:      index,
		Event:      event,
	}
	return e
}

// Client.Log is used to construct a new log event and POST it to the Splunk server.
//
// All that must be provided for a log event are the desired map[string]string key/val pairs. These can be anything
// that provide context or information for the situation you are trying to log (i.e. err messages, status codes, etc).
// The function auto-generates the event timestamp and hostname for you.
func (c *Client) Log(event interface{}) error {
	// create Splunk log
	log := c.NewEvent(event, c.Source, c.SourceType, c.Index)
	return c.LogEvent(log)
}

// Client.LogWithTime is used to construct a new log event with a specified timestamp and POST it to the Splunk server.
//
// This is similar to Client.Log, just with the t parameter.
func (c *Client) LogWithTime(t time.Time, event interface{}) error {
	// create Splunk log
	log := c.NewEventWithTime(t, event, c.Source, c.SourceType, c.Index)
	return c.LogEvent(log)
}

// Client.LogEvent is used to POST a single event to the Splunk server.
func (c *Client) LogEvent(e *Event) error {
	// Convert requestBody struct to byte slice to prep for http.NewRequest
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return c.doRequest(bytes.NewBuffer(b))
}

// Client.LogEvents is used to POST multiple events with a single request to the Splunk server.
func (c *Client) LogEvents(events []*Event) error {
	buf := new(bytes.Buffer)
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf.Write(b)
		// Each json object should be separated by a blank line
		buf.WriteString("\r\n\r\n")
	}
	// Convert requestBody struct to byte slice to prep for http.NewRequest
	return c.doRequest(buf)
}

//Writer is a convience method for creating an io.Writer from a Writer with default values
func (c *Client) Writer() io.Writer {
	return &Writer{
		Client: c,
	}
}

// Client.doRequest is used internally to POST the bytes of events to the Splunk server.
func (c *Client) doRequest(b *bytes.Buffer) error {
	// make new request
	url := c.URL
	req, err := http.NewRequest("POST", url, b)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Splunk "+c.Token)

	// receive response
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	// need to make sure we close the body to avoid hanging the connection
	defer res.Body.Close()

	// If statusCode is not OK, return the error
	switch res.StatusCode {
	case 200:
		// need to read the reply otherwise the connection hangs
		io.Copy(ioutil.Discard, res.Body)
		return nil
	default:
		respBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		// try deserializing response body to a typed HEC response
		hecResp := &EventCollectorResponse{}
		if err := json.Unmarshal(respBody, hecResp); err == nil {
			return hecResp
		}

		// otherwise, return the response body as an error string
		return errors.New(string(respBody))
	}
}
