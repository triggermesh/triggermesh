package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TriggerCondition zendesk trigger condition
//
// ref: https://developer.zendesk.com/rest_api/docs/core/triggers#conditions-reference
type TriggerCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// TriggerAction is zendesk trigger action
//
// ref: https://developer.zendesk.com/rest_api/docs/core/triggers#actions
type TriggerAction struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// Trigger is zendesk trigger JSON payload format
//
// ref: https://developer.zendesk.com/rest_api/docs/core/triggers#json-format
type Trigger struct {
	ID         int64  `json:"id,omitempty"`
	Title      string `json:"title"`
	Active     bool   `json:"active,omitempty"`
	Position   int64  `json:"position,omitempty"`
	Conditions struct {
		All []TriggerCondition `json:"all"`
		Any []TriggerCondition `json:"any"`
	} `json:"conditions"`
	Actions     []TriggerAction `json:"actions"`
	Description string          `json:"description,omitempty"`
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
}

// TriggerListOptions is options for GetTriggers
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#list-triggers
type TriggerListOptions struct {
	PageOptions
	Active    bool   `url:"active,omitempty"`
	SortBy    string `url:"sort_by,omitempty"`
	SortOrder string `url:"sort_order,omitempty"`
}

// TriggerAPI an interface containing all trigger related methods
type TriggerAPI interface {
	GetTriggers(ctx context.Context, opts *TriggerListOptions) ([]Trigger, Page, error)
	CreateTrigger(ctx context.Context, trigger Trigger) (Trigger, error)
	GetTrigger(ctx context.Context, id int64) (Trigger, error)
	UpdateTrigger(ctx context.Context, id int64, trigger Trigger) (Trigger, error)
	DeleteTrigger(ctx context.Context, id int64) error
}

// GetTriggers fetch trigger list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#getting-triggers
func (z *Client) GetTriggers(ctx context.Context, opts *TriggerListOptions) ([]Trigger, Page, error) {
	var data struct {
		Triggers []Trigger `json:"triggers"`
		Page
	}

	if opts == nil {
		return []Trigger{}, Page{}, &OptionsError{opts}
	}

	u, err := addOptions("/triggers.json", opts)
	if err != nil {
		return []Trigger{}, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return []Trigger{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return []Trigger{}, Page{}, err
	}
	return data.Triggers, data.Page, nil
}

// CreateTrigger creates new trigger
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#create-trigger
func (z *Client) CreateTrigger(ctx context.Context, trigger Trigger) (Trigger, error) {
	var data, result struct {
		Trigger Trigger `json:"trigger"`
	}
	data.Trigger = trigger

	body, err := z.post(ctx, "/triggers.json", data)
	if err != nil {
		return Trigger{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Trigger{}, err
	}
	return result.Trigger, nil
}

// GetTrigger returns the specified trigger
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#getting-triggers
func (z *Client) GetTrigger(ctx context.Context, id int64) (Trigger, error) {
	var result struct {
		Trigger Trigger `json:"trigger"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/triggers/%d.json", id))
	if err != nil {
		return Trigger{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Trigger{}, err
	}
	return result.Trigger, nil
}

// UpdateTrigger updates the specified trigger and returns the updated one
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#update-trigger
func (z *Client) UpdateTrigger(ctx context.Context, id int64, trigger Trigger) (Trigger, error) {
	var data, result struct {
		Trigger Trigger `json:"trigger"`
	}

	data.Trigger = trigger
	body, err := z.put(ctx, fmt.Sprintf("/triggers/%d.json", id), data)
	if err != nil {
		return Trigger{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Trigger{}, err
	}

	return result.Trigger, nil
}

// DeleteTrigger deletes the specified trigger
//
// ref: https://developer.zendesk.com/rest_api/docs/support/triggers#delete-trigger
func (z *Client) DeleteTrigger(ctx context.Context, id int64) error {
	err := z.delete(ctx, fmt.Sprintf("/triggers/%d.json", id))
	if err != nil {
		return err
	}

	return nil
}
