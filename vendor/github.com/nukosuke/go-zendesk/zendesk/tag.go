package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tag is an alias for string
type Tag string

// TagAPI an interface containing all tag related methods
type TagAPI interface {
	GetTicketTags(ctx context.Context, ticketID int64) ([]Tag, error)
	GetOrganizationTags(ctx context.Context, organizationID int64) ([]Tag, error)
	GetUserTags(ctx context.Context, userID int64) ([]Tag, error)
	AddTicketTags(ctx context.Context, ticketID int64, tags []Tag) ([]Tag, error)
	AddOrganizationTags(ctx context.Context, organizationID int64, tags []Tag) ([]Tag, error)
	AddUserTags(ctx context.Context, userID int64, tags []Tag) ([]Tag, error)
}

// GetTicketTags get ticket tag list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#show-tags
func (z *Client) GetTicketTags(ctx context.Context, ticketID int64) ([]Tag, error) {
	var result struct {
		Tags []Tag `json:"tags"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/tickets/%d/tags.json", ticketID))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Tags, err
}

// GetOrganizationTags get organization tag list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#show-tags
func (z *Client) GetOrganizationTags(ctx context.Context, organizationID int64) ([]Tag, error) {
	var result struct {
		Tags []Tag `json:"tags"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/organizations/%d/tags.json", organizationID))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Tags, err
}

// GetUserTags get user tag list
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#show-tags
func (z *Client) GetUserTags(ctx context.Context, userID int64) ([]Tag, error) {
	var result struct {
		Tags []Tag `json:"tags"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/users/%d/tags.json", userID))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Tags, err
}

// AddTicketTags add tags to ticket
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#add-tags
func (z *Client) AddTicketTags(ctx context.Context, ticketID int64, tags []Tag) ([]Tag, error) {
	var data, result struct {
		Tags []Tag `json:"tags"`
	}
	data.Tags = tags

	body, err := z.put(ctx, fmt.Sprintf("/tickets/%d/tags", ticketID), data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result.Tags, nil
}

// AddOrganizationTags add tags to organization
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#add-tags
func (z *Client) AddOrganizationTags(ctx context.Context, organizationID int64, tags []Tag) ([]Tag, error) {
	var data, result struct {
		Tags []Tag `json:"tags"`
	}
	data.Tags = tags

	body, err := z.put(ctx, fmt.Sprintf("/organizations/%d/tags", organizationID), data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result.Tags, nil
}

// AddUserTags add tags to user
//
// ref: https://developer.zendesk.com/rest_api/docs/support/tags#add-tags
func (z *Client) AddUserTags(ctx context.Context, userID int64, tags []Tag) ([]Tag, error) {
	var data, result struct {
		Tags []Tag `json:"tags"`
	}
	data.Tags = tags

	body, err := z.put(ctx, fmt.Sprintf("/users/%d/tags", userID), data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result.Tags, nil
}
