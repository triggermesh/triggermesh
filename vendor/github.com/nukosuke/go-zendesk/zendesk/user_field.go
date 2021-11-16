package zendesk

import (
	"context"
	"encoding/json"
	"time"
)

// UserField is struct for user_field payload
type UserField struct {
	ID                  int64               `json:"id,omitempty"`
	URL                 string              `json:"url,omitempty"`
	Key                 string              `json:"key,omitempty"`
	Type                string              `json:"type"`
	Title               string              `json:"title"`
	RawTitle            string              `json:"raw_title,omitempty"`
	Description         string              `json:"description,omitempty"`
	RawDescription      string              `json:"raw_description,omitempty"`
	Position            int64               `json:"position,omitempty"`
	Active              bool                `json:"active,omitempty"`
	System              bool                `json:"system,omitempty"`
	RegexpForValidation string              `json:"regexp_for_validation,omitempty"`
	Tag                 string              `json:"tag,omitempty"`
	CustomFieldOptions  []CustomFieldOption `json:"custom_field_options"`
	CreatedAt           time.Time           `json:"created_at,omitempty"`
	UpdatedAt           time.Time           `json:"updated_at,omitempty"`
}

type UserFieldListOptions struct {
	PageOptions
}

type UserFieldAPI interface {
	GetUserFields(ctx context.Context, opts *UserFieldListOptions) ([]UserField, Page, error)
}

// GetUserFields fetch trigger list
//
// https://developer.zendesk.com/rest_api/docs/support/user_fields#list-user-fields
func (z *Client) GetUserFields(ctx context.Context, opts *UserFieldListOptions) ([]UserField, Page, error) {
	var data struct {
		UserFields []UserField `json:"user_fields"`
		Page
	}

	tmp := opts
	if tmp == nil {
		tmp = &UserFieldListOptions{}
	}

	u, err := addOptions("/user_fields.json", tmp)
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
	return data.UserFields, data.Page, nil
}
