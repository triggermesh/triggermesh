package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Target is struct for target payload
type Target struct {
	URL       string     `json:"url,omitempty"`
	ID        int64      `json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Active    bool       `json:"active,omitempty"`
	// email_target
	Email   string `json:"email,omitempty"`
	Subject string `json:"subject,omitempty"`
	// http_target
	TargetURL   string `json:"target_url,omitempty"`
	Method      string `json:"method,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// TargetAPI an interface containing all of the target related zendesk methods
type TargetAPI interface {
	GetTargets(ctx context.Context) ([]Target, Page, error)
	CreateTarget(ctx context.Context, ticketField Target) (Target, error)
	GetTarget(ctx context.Context, ticketID int64) (Target, error)
	UpdateTarget(ctx context.Context, ticketID int64, field Target) (Target, error)
	DeleteTarget(ctx context.Context, ticketID int64) error
}

// GetTargets fetches target list
// ref: https://developer.zendesk.com/rest_api/docs/core/targets#list-targets
func (z *Client) GetTargets(ctx context.Context) ([]Target, Page, error) {
	var data struct {
		Targets []Target `json:"targets"`
		Page
	}

	body, err := z.get(ctx, "/targets.json")
	if err != nil {
		return []Target{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return []Target{}, Page{}, err
	}

	return data.Targets, data.Page, nil
}

// CreateTarget creates new target
// ref: https://developer.zendesk.com/rest_api/docs/core/targets#create-target
func (z *Client) CreateTarget(ctx context.Context, target Target) (Target, error) {
	var data, result struct {
		Target Target `json:"target"`
	}

	data.Target = target

	body, err := z.post(ctx, "/targets.json", data)
	if err != nil {
		return Target{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Target{}, err
	}

	return result.Target, nil
}

// GetTarget gets a specified target
// ref: https://developer.zendesk.com/rest_api/docs/support/targets#show-target
func (z *Client) GetTarget(ctx context.Context, targetID int64) (Target, error) {
	var result struct {
		Target Target `json:"target"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/targets/%d.json", targetID))

	if err != nil {
		return Target{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Target{}, err
	}

	return result.Target, err
}

// UpdateTarget updates a field with the specified target
// ref: https://developer.zendesk.com/rest_api/docs/support/targets#update-target
func (z *Client) UpdateTarget(ctx context.Context, targetID int64, field Target) (Target, error) {
	var result, data struct {
		Target Target `json:"target"`
	}

	data.Target = field

	body, err := z.put(ctx, fmt.Sprintf("/targets/%d.json", targetID), data)

	if err != nil {
		return Target{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Target{}, err
	}

	return result.Target, err
}

// DeleteTarget deletes the specified target
// ref: https://developer.zendesk.com/rest_api/docs/support/targets#delete-target
func (z *Client) DeleteTarget(ctx context.Context, targetID int64) error {
	err := z.delete(ctx, fmt.Sprintf("/targets/%d.json", targetID))

	if err != nil {
		return err
	}

	return nil
}
