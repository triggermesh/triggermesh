package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Group is struct for group payload
// https://developer.zendesk.com/rest_api/docs/support/groups
type Group struct {
	ID        int64     `json:"id,omitempty"`
	URL       string    `json:"url,omitempty"`
	Name      string    `json:"name"`
	Deleted   bool      `json:"deleted,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// GroupAPI an interface containing all methods associated with zendesk groups
type GroupAPI interface {
	GetGroups(ctx context.Context) ([]Group, Page, error)
	GetGroup(ctx context.Context, groupID int64) (Group, error)
	CreateGroup(ctx context.Context, group Group) (Group, error)
	UpdateGroup(ctx context.Context, groupID int64, group Group) (Group, error)
	DeleteGroup(ctx context.Context, groupID int64) error
}

// GetGroups fetches group list
// https://developer.zendesk.com/rest_api/docs/support/groups#list-groups
func (z *Client) GetGroups(ctx context.Context) ([]Group, Page, error) {
	var data struct {
		Groups []Group `json:"groups"`
		Page
	}

	body, err := z.get(ctx, "/groups.json")
	if err != nil {
		return []Group{}, Page{}, err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return []Group{}, Page{}, err
	}
	return data.Groups, data.Page, nil
}

// CreateGroup creates new group
// https://developer.zendesk.com/rest_api/docs/support/groups#create-group
func (z *Client) CreateGroup(ctx context.Context, group Group) (Group, error) {
	var data, result struct {
		Group Group `json:"group"`
	}
	data.Group = group

	body, err := z.post(ctx, "/groups.json", data)
	if err != nil {
		return Group{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Group{}, err
	}
	return result.Group, nil
}

// GetGroup gets a specified group
// ref: https://developer.zendesk.com/rest_api/docs/support/groups#show-group
func (z *Client) GetGroup(ctx context.Context, groupID int64) (Group, error) {
	var result struct {
		Group Group `json:"group"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/groups/%d.json", groupID))

	if err != nil {
		return Group{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Group{}, err
	}

	return result.Group, err
}

// UpdateGroup updates a group with the specified group
// ref: https://developer.zendesk.com/rest_api/docs/support/groups#update-group
func (z *Client) UpdateGroup(ctx context.Context, groupID int64, group Group) (Group, error) {
	var result, data struct {
		Group Group `json:"group"`
	}
	data.Group = group

	body, err := z.put(ctx, fmt.Sprintf("/groups/%d.json", groupID), data)

	if err != nil {
		return Group{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Group{}, err
	}

	return result.Group, err
}

// DeleteGroup deletes the specified group
// ref: https://developer.zendesk.com/rest_api/docs/support/groups#delete-group
func (z *Client) DeleteGroup(ctx context.Context, groupID int64) error {
	err := z.delete(ctx, fmt.Sprintf("/groups/%d.json", groupID))

	if err != nil {
		return err
	}

	return nil
}
