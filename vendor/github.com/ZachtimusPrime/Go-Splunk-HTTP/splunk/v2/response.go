package splunk

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// EventCollectorResponse is the payload returned by the HTTP Event Collector
// in response to requests.
// https://docs.splunk.com/Documentation/Splunk/latest/RESTREF/RESTinput#services.2Fcollector
type EventCollectorResponse struct {
	Text               string     `json:"text"`
	Code               StatusCode `json:"code"`
	InvalidEventNumber *int       `json:"invalid-event-number"`
	AckID              *int       `json:"ackId"`
}

var _ error = (*EventCollectorResponse)(nil)

// Error implements the error interface.
func (r *EventCollectorResponse) Error() string {
	if r == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(r.Text + " (Code: " + strconv.Itoa(int(r.Code)))
	if r.InvalidEventNumber != nil {
		sb.WriteString(", InvalidEventNumber: " + strconv.Itoa(*r.InvalidEventNumber))
	}
	if r.AckID != nil {
		sb.WriteString(", AckID: " + strconv.Itoa(*r.AckID))
	}
	sb.WriteRune(')')

	return sb.String()
}

// StatusCode defines the meaning of responses returned by HTTP Event Collector
// endpoints.
type StatusCode int8

const (
	Success StatusCode = iota
	TokenDisabled
	TokenRequired
	InvalidAuthz
	InvalidToken
	NoData
	InvalidDataFormat
	IncorrectIndex
	InternalServerError
	ServerBusy
	DataChannelMissing
	InvalidDataChannel
	EventFieldRequired
	EventFieldBlank
	ACKDisabled
	ErrorHandlingIndexedFields
	QueryStringAuthzNotEnabled
)

// HTTPCode returns the HTTP code corresponding to the given StatusCode. It
// returns -1 and an error in case the HTTP status code can not be determined.
func (c StatusCode) HTTPCode() (code int, err error) {
	switch c {
	case Success:
		code = http.StatusOK
	case TokenDisabled:
		code = http.StatusForbidden
	case TokenRequired:
		code = http.StatusUnauthorized
	case InvalidAuthz:
		code = http.StatusUnauthorized
	case InvalidToken:
		code = http.StatusForbidden
	case NoData:
		code = http.StatusBadRequest
	case InvalidDataFormat:
		code = http.StatusBadRequest
	case IncorrectIndex:
		code = http.StatusBadRequest
	case InternalServerError:
		code = http.StatusInternalServerError
	case ServerBusy:
		code = http.StatusServiceUnavailable
	case DataChannelMissing:
		code = http.StatusBadRequest
	case InvalidDataChannel:
		code = http.StatusBadRequest
	case EventFieldRequired:
		code = http.StatusBadRequest
	case EventFieldBlank:
		code = http.StatusBadRequest
	case ACKDisabled:
		code = http.StatusBadRequest
	case ErrorHandlingIndexedFields:
		code = http.StatusBadRequest
	case QueryStringAuthzNotEnabled:
		code = http.StatusBadRequest
	default:
		code = -1
		err = fmt.Errorf("unknown status code %d", c)
	}

	return
}
