package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type (
	// View is struct for group membership payload
	// https://developer.zendesk.com/api-reference/ticketing/business-rules/views/
	View struct {
		ID          int64     `json:"id,omitempty"`
		Active      bool      `json:"active"`
		Description string    `json:"description"`
		Position    int64     `json:"position"`
		Title       string    `json:"title"`
		CreatedAt   time.Time `json:"created_at,omitempty"`
		UpdatedAt   time.Time `json:"updated_at,omitempty"`

		// Conditions Conditions
		// Execution Execution
		// Restriction Restriction
	}

	// ViewAPI encapsulates methods on view
	ViewAPI interface {
		GetView(context.Context, int64) (View, error)
		GetViews(context.Context) ([]View, Page, error)
		GetTicketsFromView(context.Context, int64) ([]Ticket, error)
	}
)

// GetViews gets all views
// ref: https://developer.zendesk.com/api-reference/ticketing/business-rules/views/#list-views
func (z *Client) GetViews(ctx context.Context) ([]View, Page, error) {
	var result struct {
		Views []View `json:"views"`
		Page
	}

	body, err := z.get(ctx, "/views.json")

	if err != nil {
		return []View{}, Page{}, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return []View{}, Page{}, err
	}

	return result.Views, result.Page, nil
}

// GetView gets a given view
// ref: https://developer.zendesk.com/api-reference/ticketing/business-rules/views/#show-view
func (z *Client) GetView(ctx context.Context, viewID int64) (View, error) {
	var result struct {
		View View `json:"view"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/views/%d.json", viewID))

	if err != nil {
		return View{}, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return View{}, err
	}

	return result.View, nil
}

// GetTicketsFromView gets the tickets of the specified view
// ref: https://developer.zendesk.com/api-reference/ticketing/business-rules/views/#list-tickets-from-a-view
func (z *Client) GetTicketsFromView(ctx context.Context, viewID int64) ([]Ticket, error) {
	var result struct {
		Tickets []Ticket `json:"tickets"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/views/%d/tickets.json", viewID))

	if err != nil {
		return []Ticket{}, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return []Ticket{}, err
	}

	return result.Tickets, nil
}
