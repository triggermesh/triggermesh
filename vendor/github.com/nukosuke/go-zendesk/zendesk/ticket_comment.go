package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TicketCommentAPI is an interface containing all ticket comment related API methods
type TicketCommentAPI interface {
	CreateTicketComment(ctx context.Context, ticketID int64, ticketComment TicketComment) (TicketComment, error)
	ListTicketComments(ctx context.Context, ticketID int64) ([]TicketComment, error)
}

// TicketComment is a struct for ticket comment payload
// Via and Metadata are currently unused
// https://developer.zendesk.com/rest_api/docs/support/ticket_comments
type TicketComment struct {
	ID          int64                  `json:"id,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Body        string                 `json:"body,omitempty"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	PlainBody   string                 `json:"plain_body,omitempty"`
	Public      *bool                  `json:"public,omitempty"`
	AuthorID    int64                  `json:"author_id,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	Uploads     []string               `json:"uploads,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	Via *Via `json:"via,omitempty"`
}

// NewPublicTicketComment generates and returns a new TicketComment
func NewPublicTicketComment(body string, authorID int64) TicketComment {
	public := true

	return TicketComment{
		Body:     body,
		Public:   &public,
		AuthorID: authorID,
	}
}

// NewPrivateTicketComment generates and returns a new private TicketComment
func NewPrivateTicketComment(body string, authorID int64) TicketComment {
	public := false

	return TicketComment{
		Body:     body,
		Public:   &public,
		AuthorID: authorID,
	}
}

// CreateTicketComment creates a comment on a ticket
//
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_comments#create-ticket-comment
func (z *Client) CreateTicketComment(ctx context.Context, ticketID int64, ticketComment TicketComment) (TicketComment, error) {
	type comment struct {
		Ticket struct {
			TicketComment TicketComment `json:"comment"`
		} `json:"ticket"`
	}

	data := &comment{}
	data.Ticket.TicketComment = ticketComment

	body, err := z.put(ctx, fmt.Sprintf("/tickets/%d.json", ticketID), data)
	if err != nil {
		return TicketComment{}, err
	}

	result := TicketComment{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return TicketComment{}, err
	}

	return result, err
}

// ListTicketComments gets a list of comment for a specified ticket
//
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_comments#list-comments
func (z *Client) ListTicketComments(ctx context.Context, ticketID int64) ([]TicketComment, error) {
	var result struct {
		TicketComments []TicketComment `json:"comments"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/tickets/%d/comments.json", ticketID))
	if err != nil {
		return []TicketComment{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return []TicketComment{}, err
	}

	return result.TicketComments, err
}
