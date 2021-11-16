package zendesk

import (
	"context"
	"encoding/json"
	"time"
)

type (
	// GroupMembership is struct for group membership payload
	// https://developer.zendesk.com/api-reference/ticketing/groups/group_memberships/
	GroupMembership struct {
		ID        int64     `json:"id,omitempty"`
		URL       string    `json:"url,omitempty"`
		UserID    int64     `json:"user_id"`
		GroupID   int64     `json:"group_id"`
		Default   bool      `json:"default"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`
	}

	// GroupMembershipListOptions is a struct for options for group membership list
	// ref: https://developer.zendesk.com/api-reference/ticketing/groups/group_memberships/
	GroupMembershipListOptions struct {
		PageOptions
		GroupID int64 `json:"group_id,omitempty" url:"group_id,omitempty"`
		UserID  int64 `json:"user_id,omitempty" url:"user_id,omitempty"`
	}

	// GroupMembershipAPI is an interface containing group membership related methods
	GroupMembershipAPI interface {
		GetGroupMemberships(context.Context, *GroupMembershipListOptions) ([]GroupMembership, Page, error)
	}
)

// GetGroupMemberships gets the memberships of the specified group
// ref: https://developer.zendesk.com/api-reference/ticketing/groups/group_memberships/
func (z *Client) GetGroupMemberships(ctx context.Context, opts *GroupMembershipListOptions) ([]GroupMembership, Page, error) {
	var result struct {
		GroupMemberships []GroupMembership `json:"group_memberships"`
		Page
	}

	tmp := opts
	if tmp == nil {
		tmp = new(GroupMembershipListOptions)
	}

	u, err := addOptions("/group_memberships.json", tmp)
	if err != nil {
		return nil, Page{}, err
	}

	body, err := z.get(ctx, u)
	if err != nil {
		return nil, Page{}, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, Page{}, err
	}

	return result.GroupMemberships, result.Page, nil
}
