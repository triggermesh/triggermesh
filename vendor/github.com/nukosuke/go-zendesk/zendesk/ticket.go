package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CustomField struct {
	ID int64 `json:"id"`
	// Valid types are string or []string.
	Value interface{} `json:"value"`
}

// UnmarshalJSON Custom Unmarshal function required because a custom field's value can be
// a string or array of strings.
func (cf *CustomField) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	cf.ID = int64(temp["id"].(float64))

	switch v := temp["value"].(type) {
	case string, nil, bool:
		cf.Value = v
	case []interface{}:
		var list []string

		for _, v := range temp["value"].([]interface{}) {
			if s, ok := v.(string); ok {
				list = append(list, s)
			} else {
				return fmt.Errorf("%T is an invalid type for custom field value", v)
			}
		}

		cf.Value = list
	default:
		return fmt.Errorf("%T is an invalid type for custom field value", v)
	}

	return nil
}

type Ticket struct {
	ID              int64         `json:"id,omitempty"`
	URL             string        `json:"url,omitempty"`
	ExternalID      string        `json:"external_id,omitempty"`
	Type            string        `json:"type,omitempty"`
	Subject         string        `json:"subject,omitempty"`
	RawSubject      string        `json:"raw_subject,omitempty"`
	Description     string        `json:"description,omitempty"`
	Priority        string        `json:"priority,omitempty"`
	Status          string        `json:"status,omitempty"`
	Recipient       string        `json:"recipient,omitempty"`
	RequesterID     int64         `json:"requester_id,omitempty"`
	SubmitterID     int64         `json:"submitter_id,omitempty"`
	AssigneeID      int64         `json:"assignee_id,omitempty"`
	OrganizationID  int64         `json:"organization_id,omitempty"`
	GroupID         int64         `json:"group_id,omitempty"`
	CollaboratorIDs []int64       `json:"collaborator_ids,omitempty"`
	FollowerIDs     []int64       `json:"follower_ids,omitempty"`
	EmailCCIDs      []int64       `json:"email_cc_ids,omitempty"`
	ForumTopicID    int64         `json:"forum_topic_id,omitempty"`
	ProblemID       int64         `json:"problem_id,omitempty"`
	HasIncidents    bool          `json:"has_incidents,omitempty"`
	DueAt           time.Time     `json:"due_at,omitempty"`
	Tags            []string      `json:"tags,omitempty"`
	CustomFields    []CustomField `json:"custom_fields,omitempty"`

	Via *Via `json:"via,omitempty"`

	SatisfactionRating struct {
		ID      int64  `json:"id"`
		Score   string `json:"score"`
		Comment string `json:"comment"`
	} `json:"satisfaction_rating,omitempty"`

	SharingAgreementIDs []int64   `json:"sharing_agreement_ids,omitempty"`
	FollowupIDs         []int64   `json:"followup_ids,omitempty"`
	ViaFollowupSourceID int64     `json:"via_followup_source_id,omitempty"`
	MacroIDs            []int64   `json:"macro_ids,omitempty"`
	TicketFormID        int64     `json:"ticket_form_id,omitempty"`
	BrandID             int64     `json:"brand_id,omitempty"`
	AllowChannelback    bool      `json:"allow_channelback,omitempty"`
	AllowAttachments    bool      `json:"allow_attachments,omitempty"`
	IsPublic            bool      `json:"is_public,omitempty"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`

	// Collaborators is POST only
	Collaborators Collaborators `json:"collaborators,omitempty"`

	// Comment is POST only and required
	Comment TicketComment `json:"comment,omitempty"`
	
	// Requester is POST only and can be used to create a ticket for a nonexistent requester
	Requester Requester `json:"requester,omitempty"`

	// TODO: TicketAudit (POST only) #126
}

// Requester is the struct that can be passed to create a new requester on ticket creation
// https://develop.zendesk.com/hc/en-us/articles/360059146153#creating-a-ticket-with-a-new-requester
type Requester struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	LocaleID string `json:"locale_id,omitempty"`
}

// Via is information about source of Ticket or TicketComment
type Via struct {
	Channel string `json:"channel"`
	Source  struct {
		From map[string]interface{} `json:"from"`
		To   map[string]interface{} `json:"to"`
		Rel  string                 `json:"rel"`
	} `json:"source"`
}

type TicketListOptions struct {
	PageOptions

	// SortBy can take "assignee", "assignee.name", "created_at", "group", "id",
	// "locale", "requester", "requester.name", "status", "subject", "updated_at"
	SortBy string `url:"sort_by,omitempty"`

	// SortOrder can take "asc" or "desc"
	SortOrder string `url:"sort_order,omitempty"`
}

// TicketAPI an interface containing all ticket related methods
type TicketAPI interface {
	GetTickets(ctx context.Context, opts *TicketListOptions) ([]Ticket, Page, error)
	GetTicket(ctx context.Context, id int64) (Ticket, error)
	GetMultipleTickets(ctx context.Context, ticketIDs []int64) ([]Ticket, error)
	CreateTicket(ctx context.Context, ticket Ticket) (Ticket, error)
	UpdateTicket(ctx context.Context, ticketID int64, ticket Ticket) (Ticket, error)
	DeleteTicket(ctx context.Context, ticketID int64) error
}

// GetTickets get ticket list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#list-tickets
func (z *Client) GetTickets(ctx context.Context, opts *TicketListOptions) ([]Ticket, Page, error) {
	var data struct {
		Tickets []Ticket `json:"tickets"`
		Page
	}

	tmp := opts
	if tmp == nil {
		tmp = &TicketListOptions{}
	}

	u, err := addOptions("/tickets.json", tmp)
	if err != nil {
		return nil, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return nil, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, Page{}, err
	}
	return data.Tickets, data.Page, nil
}

// GetTicket gets a specified ticket
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#show-ticket
func (z *Client) GetTicket(ctx context.Context, ticketID int64) (Ticket, error) {
	var result struct {
		Ticket Ticket `json:"ticket"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/tickets/%d.json", ticketID))
	if err != nil {
		return Ticket{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Ticket{}, err
	}

	return result.Ticket, err
}

// GetMultipleTickets gets multiple specified tickets
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#show-multiple-tickets
func (z *Client) GetMultipleTickets(ctx context.Context, ticketIDs []int64) ([]Ticket, error) {
	var result struct {
		Tickets []Ticket `json:"tickets"`
	}

	var req struct {
		IDs string `url:"ids,omitempty"`
	}
	idStrs := make([]string, len(ticketIDs))
	for i := 0; i < len(ticketIDs); i++ {
		idStrs[i] = strconv.FormatInt(ticketIDs[i], 10)
	}
	req.IDs = strings.Join(idStrs, ",")

	u, err := addOptions("/tickets/show_many.json", req)
	if err != nil {
		return nil, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result.Tickets, nil
}

// CreateTicket create a new ticket
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#create-ticket
func (z *Client) CreateTicket(ctx context.Context, ticket Ticket) (Ticket, error) {
	var data, result struct {
		Ticket Ticket `json:"ticket"`
	}
	data.Ticket = ticket

	body, err := z.post(ctx, "/tickets.json", data)
	if err != nil {
		return Ticket{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Ticket{}, err
	}
	return result.Ticket, nil
}

// UpdateTicket update an existing ticket
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#update-ticket
func (z *Client) UpdateTicket(ctx context.Context, ticketID int64, ticket Ticket) (Ticket, error) {
	var data, result struct {
		Ticket Ticket `json:"ticket"`
	}
	data.Ticket = ticket

	path := fmt.Sprintf("/tickets/%d.json", ticketID)
	body, err := z.put(ctx, path, data)
	if err != nil {
		return Ticket{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Ticket{}, err
	}

	return result.Ticket, nil
}

// DeleteTicket deletes the specified ticket
// ref: https://developer.zendesk.com/rest_api/docs/support/tickets#delete-ticket
func (z *Client) DeleteTicket(ctx context.Context, ticketID int64) error {
	err := z.delete(ctx, fmt.Sprintf("/tickets/%d.json", ticketID))

	if err != nil {
		return err
	}

	return nil
}
