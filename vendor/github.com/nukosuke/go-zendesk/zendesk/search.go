package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
)

// SearchOptions are the options that can be provided to the search API
//
// ref: https://developer.zendesk.com/rest_api/docs/support/search#available-parameters
type SearchOptions struct {
	PageOptions
	Query     string `url:"query"`
	SortBy    string `url:"sort_by,omitempty"`
	SortOrder string `url:"sort_order,omitempty"`
}

// CountOptions are the options that can be provided to the search results count API
//
// ref: https://developer.zendesk.com/rest_api/docs/support/search#show-results-count
type CountOptions struct {
	Query string `url:"query"`
}

type SearchAPI interface {
	Search(ctx context.Context, opts *SearchOptions) (SearchResults, Page, error)
	SearchCount(ctx context.Context, opts *CountOptions) (int, error)
}

type SearchResults struct {
	results []interface{}
}

func (r *SearchResults) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.results)
}

func (r *SearchResults) UnmarshalJSON(b []byte) error {
	var (
		results []interface{}
		tmp     []json.RawMessage
	)

	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	for _, v := range tmp {
		value, err := r.getObject(v)
		if err != nil {
			return err
		}

		results = append(results, value)
	}

	r.results = results

	return nil
}

func (r *SearchResults) getObject(blob json.RawMessage) (interface{}, error) {
	m := make(map[string]interface{})

	err := json.Unmarshal(blob, &m)
	if err != nil {
		return nil, err
	}

	t, ok := m["result_type"].(string)
	if !ok {
		return nil, fmt.Errorf("could not assert result type to string. json was: %v", blob)
	}

	var value interface{}

	switch t {
	case "group":
		var g Group
		err = json.Unmarshal(blob, &g)
		value = g
	case "ticket":
		var t Ticket
		err = json.Unmarshal(blob, &t)
		value = t
	case "user":
		var u User
		err = json.Unmarshal(blob, &u)
		value = u
	case "organization":
		var o Organization
		err = json.Unmarshal(blob, &o)
		value = o
	case "topic":
		var t Topic
		err = json.Unmarshal(blob, &t)
		value = t
	default:
		err = fmt.Errorf("value of result was an unsupported type %s", t)
	}

	if err != nil {
		return nil, err
	}

	return value, nil
}

// String return string formatted for Search results
func (r *SearchResults) String() string {
	return fmt.Sprintf("%v", r.results)
}

// List return internal array in Search Results
func (r *SearchResults) List() []interface{} {
	return r.results
}

// Search allows users to query zendesk's unified search api.
//
// ref: https://developer.zendesk.com/rest_api/docs/support/search
func (z *Client) Search(ctx context.Context, opts *SearchOptions) (SearchResults, Page, error) {
	var data struct {
		Results SearchResults `json:"results"`
		Page
	}

	if opts == nil {
		return SearchResults{}, Page{}, &OptionsError{opts}
	}

	u, err := addOptions("/search.json", opts)
	if err != nil {
		return SearchResults{}, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return SearchResults{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return SearchResults{}, Page{}, err
	}

	return data.Results, data.Page, nil
}

// SearchCount allows users to get count of results of a query of zendesk's unified search api.
//
// ref: https://developer.zendesk.com/rest_api/docs/support/search#show-results-count
func (z *Client) SearchCount(ctx context.Context, opts *CountOptions) (int, error) {
	var data struct {
		Count int `json:"count"`
	}

	if opts == nil {
		return 0, &OptionsError{opts}
	}

	u, err := addOptions("/search/count.json", opts)
	if err != nil {
		return 0, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}

	return data.Count, nil
}
