package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Macro is information about zendesk macro
type Macro struct {
	Actions     []MacroAction `json:"actions"`
	Active      bool          `json:"active"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
	Description interface{}   `json:"description"`
	ID          int64         `json:"id,omitempty"`
	Position    int           `json:"position,omitempty"`
	Restriction interface{}   `json:"restriction"`
	Title       string        `json:"title"`
	UpdatedAt   time.Time     `json:"updated_at,omitempty"`
	URL         string        `json:"url,omitempty"`
}

// MacroAction is definition of what the macro does to the ticket
//
// ref: https://develop.zendesk.com/hc/en-us/articles/360056760874-Support-API-Actions-reference
type MacroAction struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// MacroListOptions is parameters used of GetMacros
type MacroListOptions struct {
	Access       string `json:"access"`
	Active       string `json:"active"`
	Category     int    `json:"category"`
	GroupID      int    `json:"group_id"`
	Include      string `json:"include"`
	OnlyViewable bool   `json:"only_viewable"`

	PageOptions

	// SortBy can take "created_at", "updated_at", "usage_1h", "usage_24h",
	// "usage_7d", "usage_30d", "alphabetical"
	SortBy string `url:"sort_by,omitempty"`

	// SortOrder can take "asc" or "desc"
	SortOrder string `url:"sort_order,omitempty"`
}

// MacroAPI an interface containing all macro related methods
type MacroAPI interface {
	GetMacros(ctx context.Context, opts *MacroListOptions) ([]Macro, Page, error)
	GetMacro(ctx context.Context, macroID int64) (Macro, error)
	CreateMacro(ctx context.Context, macro Macro) (Macro, error)
	UpdateMacro(ctx context.Context, macroID int64, macro Macro) (Macro, error)
	DeleteMacro(ctx context.Context, macroID int64) error
}

// GetMacros get macro list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/macros#list-macros
func (z *Client) GetMacros(ctx context.Context, opts *MacroListOptions) ([]Macro, Page, error) {
	var data struct {
		Macros []Macro `json:"macros"`
		Page
	}

	tmp := opts
	if tmp == nil {
		tmp = &MacroListOptions{}
	}

	u, err := addOptions("/macros.json", tmp)
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
	return data.Macros, data.Page, nil
}

// GetMacro gets a specified macro
//
// ref: https://developer.zendesk.com/rest_api/docs/support/macros#show-macro
func (z *Client) GetMacro(ctx context.Context, macroID int64) (Macro, error) {
	var result struct {
		Macro Macro `json:"macro"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/macros/%d.json", macroID))
	if err != nil {
		return Macro{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Macro{}, err
	}

	return result.Macro, err
}

// CreateMacro create a new macro
//
// ref: https://developer.zendesk.com/rest_api/docs/support/macros#create-macro
func (z *Client) CreateMacro(ctx context.Context, macro Macro) (Macro, error) {
	var data, result struct {
		Macro Macro `json:"macro"`
	}
	data.Macro = macro

	body, err := z.post(ctx, "/macros.json", data)
	if err != nil {
		return Macro{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Macro{}, err
	}
	return result.Macro, nil
}

// UpdateMacro update an existing macro
// ref: https://developer.zendesk.com/rest_api/docs/support/macros#update-macro
func (z *Client) UpdateMacro(ctx context.Context, macroID int64, macro Macro) (Macro, error) {
	var data, result struct {
		Macro Macro `json:"macro"`
	}
	data.Macro = macro

	path := fmt.Sprintf("/macros/%d.json", macroID)
	body, err := z.put(ctx, path, data)
	if err != nil {
		return Macro{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Macro{}, err
	}

	return result.Macro, nil
}

// DeleteMacro deletes the specified macro
// ref: https://developer.zendesk.com/rest_api/docs/support/macros#delete-macro
func (z *Client) DeleteMacro(ctx context.Context, macroID int64) error {
	err := z.delete(ctx, fmt.Sprintf("/macros/%d.json", macroID))

	if err != nil {
		return err
	}

	return nil
}
