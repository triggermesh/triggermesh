package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TicketAudit is struct for ticket_audit payload
type TicketAudit struct {
	ID        int64          `json:"id,omitempty"`
	TicketID  int64          `json:"ticket_id,omitempty"`
	Metadata  interface{}    `json:"metadata,omitempty"`
	Via       TicketAuditVia `json:"via,omitempty"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	AuthorID  int64          `json:"author_id,omitempty"`
	Events    []interface{}  `json:"events,omitempty"`
}

// TicketAuditVia is struct for via payload
type TicketAuditVia struct {
	Channel string `json:"channel,omitempty"`
	Source  struct {
		To   interface{} `json:"to,omitempty"`
		From interface{} `json:"from,omitempty"`
		Ref  string      `json:"ref,omitempty"`
	} `json:"source,omitempty"`
}

// TicketAuditAPI an interface containing all of the ticket audit related zendesk methods
type TicketAuditAPI interface {
	GetAllTicketAudits(ctx context.Context, opts CursorOption) ([]TicketAudit, Cursor, error)
	GetTicketAudits(ctx context.Context, ticketID int64, opts PageOptions) ([]TicketAudit, Page, error)
	GetTicketAudit(ctx context.Context, TicketID, ID int64) (TicketAudit, error)
}

// GetAllTicketAudits list all ticket audits
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_audits#list-all-ticket-audits
func (z *Client) GetAllTicketAudits(ctx context.Context, opts CursorOption) ([]TicketAudit, Cursor, error) {
	var result struct {
		Audits []TicketAudit `json:"audits"`
		Cursor
	}

	u, err := addOptions("/ticket_audits.json", opts)
	if err != nil {
		return []TicketAudit{}, Cursor{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return []TicketAudit{}, Cursor{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return []TicketAudit{}, Cursor{}, err
	}

	return result.Audits, result.Cursor, err
}

// GetTicketAudits list audits for a ticket
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_audits#list-audits-for-a-ticket
func (z *Client) GetTicketAudits(ctx context.Context, ticketID int64, opts PageOptions) ([]TicketAudit, Page, error) {
	var result struct {
		Audits []TicketAudit `json:"audits"`
		Page
	}

	u, err := addOptions(fmt.Sprintf("/tickets/%d/audits.json", ticketID), opts)
	if err != nil {
		return []TicketAudit{}, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return []TicketAudit{}, Page{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return []TicketAudit{}, Page{}, err
	}

	return result.Audits, result.Page, err
}

// GetTicketAudit show audit
// ref: https://developer.zendesk.com/rest_api/docs/support/ticket_audits#show-audit
func (z *Client) GetTicketAudit(ctx context.Context, ticketID, ID int64) (TicketAudit, error) {
	var result struct {
		Audit TicketAudit `json:"audit"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/tickets/%d/audits/%d.json", ticketID, ID))
	if err != nil {
		return TicketAudit{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return TicketAudit{}, err
	}

	return result.Audit, err
}
