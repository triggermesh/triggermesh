package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SLAPolicyFilter zendesk slaPolicy condition
//
// ref: https://developer.zendesk.com/rest_api/docs/core/slas/policies#conditions-reference
type SLAPolicyFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// SLA Policy metric values
//
// ref: https://developer.zendesk.com/rest_api/docs/support/sla_policies#metrics
const (
	AgentWorkTimeMetric      = "agent_work_time"
	FirstReplyTimeMetric     = "first_reply_time"
	NextReplyTimeMetric      = "next_reply_time"
	PausableUpdateTimeMetric = "pausable_update_time"
	PeriodicUpdateTimeMetric = "periodic_update_time"
	RequesterWaitTimeMetric  = "requester_wait_time"
)

type SLAPolicyMetric struct {
	Priority      string `json:"priority"`
	Metric        string `json:"metric"`
	Target        int    `json:"target"`
	BusinessHours bool   `json:"business_hours"`
}

// SLAPolicy is zendesk slaPolicy JSON payload format
//
// ref: https://developer.zendesk.com/rest_api/docs/core/slas/policies#json-format
type SLAPolicy struct {
	ID          int64  `json:"id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Position    int64  `json:"position,omitempty"`
	Active      bool   `json:"active,omitempty"`
	Filter      struct {
		All []SLAPolicyFilter `json:"all"`
		Any []SLAPolicyFilter `json:"any"`
	} `json:"filter"`
	PolicyMetrics []SLAPolicyMetric `json:"policy_metrics,omitempty"`
	CreatedAt     *time.Time        `json:"created_at,omitempty"`
	UpdatedAt     *time.Time        `json:"updated_at,omitempty"`
}

// SLAPolicyListOptions is options for GetSLAPolicies
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#list-slas/policies
type SLAPolicyListOptions struct {
	PageOptions
	Active    bool   `url:"active,omitempty"`
	SortBy    string `url:"sort_by,omitempty"`
	SortOrder string `url:"sort_order,omitempty"`
}

// SLAPolicyAPI an interface containing all slaPolicy related methods
type SLAPolicyAPI interface {
	GetSLAPolicies(ctx context.Context, opts *SLAPolicyListOptions) ([]SLAPolicy, Page, error)
	CreateSLAPolicy(ctx context.Context, slaPolicy SLAPolicy) (SLAPolicy, error)
	GetSLAPolicy(ctx context.Context, id int64) (SLAPolicy, error)
	UpdateSLAPolicy(ctx context.Context, id int64, slaPolicy SLAPolicy) (SLAPolicy, error)
	DeleteSLAPolicy(ctx context.Context, id int64) error
}

// GetSLAPolicies fetch slaPolicy list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#getting-slas/policies
func (z *Client) GetSLAPolicies(ctx context.Context, opts *SLAPolicyListOptions) ([]SLAPolicy, Page, error) {
	var data struct {
		SLAPolicies []SLAPolicy `json:"sla_policies"`
		Page
	}

	if opts == nil {
		return []SLAPolicy{}, Page{}, &OptionsError{opts}
	}

	u, err := addOptions("/slas/policies.json", opts)
	if err != nil {
		return []SLAPolicy{}, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return []SLAPolicy{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return []SLAPolicy{}, Page{}, err
	}

	return data.SLAPolicies, data.Page, nil
}

// CreateSLAPolicy creates new slaPolicy
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#create-slaPolicy
func (z *Client) CreateSLAPolicy(ctx context.Context, slaPolicy SLAPolicy) (SLAPolicy, error) {
	var data, result struct {
		SLAPolicy SLAPolicy `json:"sla_policy"`
	}

	data.SLAPolicy = slaPolicy

	body, err := z.post(ctx, "/slas/policies.json", data)
	if err != nil {
		return SLAPolicy{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SLAPolicy{}, err
	}

	return result.SLAPolicy, nil
}

// GetSLAPolicy returns the specified slaPolicy
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#getting-slas/policies
func (z *Client) GetSLAPolicy(ctx context.Context, id int64) (SLAPolicy, error) {
	var result struct {
		SLAPolicy SLAPolicy `json:"sla_policy"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/slas/policies/%d.json", id))
	if err != nil {
		return SLAPolicy{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SLAPolicy{}, err
	}

	return result.SLAPolicy, nil
}

// UpdateSLAPolicy updates the specified slaPolicy and returns the updated one
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#update-slaPolicy
func (z *Client) UpdateSLAPolicy(ctx context.Context, id int64, slaPolicy SLAPolicy) (SLAPolicy, error) {
	var data, result struct {
		SLAPolicy SLAPolicy `json:"sla_policy"`
	}

	data.SLAPolicy = slaPolicy

	body, err := z.put(ctx, fmt.Sprintf("/slas/policies/%d.json", id), data)
	if err != nil {
		return SLAPolicy{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SLAPolicy{}, err
	}

	return result.SLAPolicy, nil
}

// DeleteSLAPolicy deletes the specified slaPolicy
//
// ref: https://developer.zendesk.com/rest_api/docs/support/slas/policies#delete-slaPolicy
func (z *Client) DeleteSLAPolicy(ctx context.Context, id int64) error {
	err := z.delete(ctx, fmt.Sprintf("/slas/policies/%d.json", id))
	if err != nil {
		return err
	}

	return nil
}
