package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TicketFieldSystemFieldOption is struct for value of `system_field_options`
type TicketFieldSystemFieldOption struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int64  `json:"position"`
	RawName  string `json:"raw_name"`
	URL      string `json:"url"`
	Value    string `json:"value"`
}

// TicketField is struct for ticket_field payload
type TicketField struct {
	ID                  int64                          `json:"id,omitempty"`
	URL                 string                         `json:"url,omitempty"`
	Type                string                         `json:"type"`
	Title               string                         `json:"title"`
	RawTitle            string                         `json:"raw_title,omitempty"`
	Description         string                         `json:"description,omitempty"`
	RawDescription      string                         `json:"raw_description,omitempty"`
	Position            int64                          `json:"position,omitempty"`
	Active              bool                           `json:"active,omitempty"`
	Required            bool                           `json:"required,omitempty"`
	CollapsedForAgents  bool                           `json:"collapsed_for_agents,omitempty"`
	RegexpForValidation string                         `json:"regexp_for_validation,omitempty"`
	TitleInPortal       string                         `json:"title_in_portal,omitempty"`
	RawTitleInPortal    string                         `json:"raw_title_in_portal,omitempty"`
	VisibleInPortal     bool                           `json:"visible_in_portal,omitempty"`
	EditableInPortal    bool                           `json:"editable_in_portal,omitempty"`
	RequiredInPortal    bool                           `json:"required_in_portal,omitempty"`
	Tag                 string                         `json:"tag,omitempty"`
	CreatedAt           *time.Time                     `json:"created_at,omitempty"`
	UpdatedAt           *time.Time                     `json:"updated_at,omitempty"`
	SystemFieldOptions  []TicketFieldSystemFieldOption `json:"system_field_options,omitempty"`
	CustomFieldOptions  []CustomFieldOption            `json:"custom_field_options,omitempty"`
	SubTypeID           int64                          `json:"sub_type_id,omitempty"`
	Removable           bool                           `json:"removable,omitempty"`
	AgentDescription    string                         `json:"agent_description,omitempty"`
}

// TicketFieldAPI an interface containing all of the ticket field related zendesk methods
type TicketFieldAPI interface {
	GetTicketFields(ctx context.Context) ([]TicketField, Page, error)
	CreateTicketField(ctx context.Context, ticketField TicketField) (TicketField, error)
	GetTicketField(ctx context.Context, ticketID int64) (TicketField, error)
	UpdateTicketField(ctx context.Context, ticketID int64, field TicketField) (TicketField, error)
	DeleteTicketField(ctx context.Context, ticketID int64) error
}

// GetTicketFields fetches ticket field list
// ref: https://developer.zendesk.com/rest_api/docs/core/ticket_fields#list-ticket-fields
func (z *Client) GetTicketFields(ctx context.Context) ([]TicketField, Page, error) {
	var data struct {
		TicketFields []TicketField `json:"ticket_fields"`
		Page
	}

	body, err := z.get(ctx, "/ticket_fields.json")
	if err != nil {
		return []TicketField{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return []TicketField{}, Page{}, err
	}
	return data.TicketFields, data.Page, nil
}

// CreateTicketField creates new ticket field
// ref: https://developer.zendesk.com/rest_api/docs/core/ticket_fields#create-ticket-field
func (z *Client) CreateTicketField(ctx context.Context, ticketField TicketField) (TicketField, error) {
	var data, result struct {
		TicketField TicketField `json:"ticket_field"`
	}
	data.TicketField = ticketField

	body, err := z.post(ctx, "/ticket_fields.json", data)
	if err != nil {
		return TicketField{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return TicketField{}, err
	}
	return result.TicketField, nil
}

// GetTicketField gets a specified ticket field
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_fields#show-ticket-field
func (z *Client) GetTicketField(ctx context.Context, ticketID int64) (TicketField, error) {
	var result struct {
		TicketField TicketField `json:"ticket_field"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/ticket_fields/%d.json", ticketID))

	if err != nil {
		return TicketField{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return TicketField{}, err
	}

	return result.TicketField, err
}

// UpdateTicketField updates a field with the specified ticket field
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_fields#update-ticket-field
func (z *Client) UpdateTicketField(ctx context.Context, ticketID int64, field TicketField) (TicketField, error) {
	var result, data struct {
		TicketField TicketField `json:"ticket_field"`
	}

	data.TicketField = field

	body, err := z.put(ctx, fmt.Sprintf("/ticket_fields/%d.json", ticketID), data)

	if err != nil {
		return TicketField{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return TicketField{}, err
	}

	return result.TicketField, err
}

// DeleteTicketField deletes the specified ticket field
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_fields#delete-ticket-field
func (z *Client) DeleteTicketField(ctx context.Context, ticketID int64) error {
	err := z.delete(ctx, fmt.Sprintf("/ticket_fields/%d.json", ticketID))

	if err != nil {
		return err
	}

	return nil
}
