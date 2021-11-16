package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// UserFields is a dictionary of custom user related fields
type UserFields map[string]interface{}

// User is zendesk user JSON payload format
// https://developer.zendesk.com/rest_api/docs/support/users
type User struct {
	ID                   int64      `json:"id,omitempty"`
	URL                  string     `json:"url,omitempty"`
	Email                string     `json:"email,omitempty"`
	Name                 string     `json:"name"`
	Active               bool       `json:"active,omitempty"`
	Alias                string     `json:"alias,omitempty"`
	ChatOnly             bool       `json:"chat_only,omitempty"`
	CustomRoleID         int64      `json:"custom_role_id,omitempty"`
	DefaultGroupID       int64      `json:"default_group_id,omitempty"`
	Details              string     `json:"details,omitempty"`
	ExternalID           string     `json:"external_id,omitempty"`
	IanaTimezone         string     `json:"iana_time_zone,omitempty"`
	Locale               string     `json:"locale,omitempty"`
	LocaleID             int64      `json:"locale_id,omitempty"`
	Moderator            bool       `json:"moderator,omitempty"`
	Notes                string     `json:"notes,omitempty"`
	OnlyPrivateComments  bool       `json:"only_private_comments,omitempty"`
	OrganizationID       int64      `json:"organization_id,omitempty"`
	Phone                string     `json:"phone,omitempty"`
	Photo                Attachment `json:"photo,omitempty"`
	RestrictedAgent      bool       `json:"restricted_agent,omitempty"`
	Role                 string     `json:"role,omitempty"`
	RoleType             int64      `json:"role_type,omitempty"`
	Shared               bool       `json:"shared,omitempty"`
	SharedAgent          bool       `json:"shared_agent,omitempty"`
	SharedPhoneNumber    bool       `json:"shared_phone_number,omitempty"`
	Signature            string     `json:"signature,omitempty"`
	Suspended            bool       `json:"suspended,omitempty"`
	Tags                 []string   `json:"tags,omitempty"`
	TicketRestriction    string     `json:"ticket_restriction,omitempty"`
	Timezone             string     `json:"time_zone,omitempty"`
	TwoFactorAuthEnabled bool       `json:"two_factor_auth_enabled,omitempty"`
	UserFields           UserFields `json:"user_fields"`
	Verified             bool       `json:"verified,omitempty"`
	ReportCSV            bool       `json:"report_csv,omitempty"`
	LastLoginAt          time.Time  `json:"last_login_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at,omitempty"`
}

const (
	// UserRoleEndUser end-user
	UserRoleEndUser = iota
	// UserRoleAgent agent
	UserRoleAgent
	// UserRoleAdmin admin
	UserRoleAdmin
)

var userRoleText = map[int]string{
	UserRoleEndUser: "end-user",
	UserRoleAgent:   "agent",
	UserRoleAdmin:   "admin",
}

// UserListOptions is options for GetUsers
//
// ref: https://developer.zendesk.com/rest_api/docs/support/users#list-users
type UserListOptions struct {
	PageOptions
	Role          string   `url:"role,omitempty"`
	Roles         []string `url:"role[],omitempty"`
	PermissionSet int64    `url:"permission_set,omitempty"`
}

// UserRoleText takes role type and returns role name string
func UserRoleText(role int) string {
	return userRoleText[role]
}

// GetManyUsersOptions is options for GetManyUsers
//
// ref: https://developer.zendesk.com/api-reference/ticketing/users/users/#show-many-users
type GetManyUsersOptions struct {
	ExternalIDs string `json:"external_ids,omitempty" url:"external_ids,omitempty"`
	IDs         string `json:"ids,omitempty" url:"ids,omitempty"`
}

// UserRelated contains user related data
//
// ref: https://developer.zendesk.com/api-reference/ticketing/users/users/#show-user-related-information
type UserRelated struct {
	AssignedTickets           int64 `json:"assigned_tickets"`
	RequestedTickets          int64 `json:"requested_tickets"`
	CCDTickets                int64 `json:"ccd_tickets"`
	OrganizationSubscriptions int64 `json:"organization_subscriptions"`
}

// UserAPI an interface containing all user related methods
type UserAPI interface {
	GetManyUsers(ctx context.Context, opts *GetManyUsersOptions) ([]User, Page, error)
	GetUsers(ctx context.Context, opts *UserListOptions) ([]User, Page, error)
	GetUser(ctx context.Context, userID int64) (User, error)
	CreateUser(ctx context.Context, user User) (User, error)
	UpdateUser(ctx context.Context, userID int64, user User) (User, error)
	GetUserRelated(ctx context.Context, userID int64) (UserRelated, error)
}

// GetUsers fetch user list
func (z *Client) GetUsers(ctx context.Context, opts *UserListOptions) ([]User, Page, error) {
	var data struct {
		Users []User `json:"users"`
		Page
	}

	tmp := opts
	if tmp == nil {
		tmp = &UserListOptions{}
	}

	u, err := addOptions("/users.json", tmp)
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
	return data.Users, data.Page, nil
}

// GetManyUsers fetch user list
// https://developer.zendesk.com/api-reference/ticketing/users/users/#show-many-users
func (z *Client) GetManyUsers(ctx context.Context, opts *GetManyUsersOptions) ([]User, Page, error) {
	var (
		data struct {
			Users []User `json:"users"`
			Page
		}
	)

	tmp := opts
	if tmp == nil {
		tmp = new(GetManyUsersOptions)
	}

	u, err := addOptions("/users/show_many.json", tmp)
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
	return data.Users, data.Page, nil
}

//TODO: GetUsersByGroupID, GetUsersByOrganizationID

// CreateUser creates new user
// ref: https://developer.zendesk.com/rest_api/docs/core/triggers#create-trigger
func (z *Client) CreateUser(ctx context.Context, user User) (User, error) {
	var data, result struct {
		User User `json:"user"`
	}
	data.User = user

	body, err := z.post(ctx, "/users.json", data)
	if err != nil {
		return User{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return User{}, err
	}
	return result.User, nil
}

// TODO: CreateOrUpdateManyUsers(users []User)

// GetUser get an existing user
// ref: https://developer.zendesk.com/rest_api/docs/support/users#show-user
func (z *Client) GetUser(ctx context.Context, userID int64) (User, error) {
	var result struct {
		User User `json:"user"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/users/%d.json", userID))
	if err != nil {
		return User{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return User{}, err
	}
	return result.User, nil
}

// UpdateUser update an existing user
// ref: https://developer.zendesk.com/rest_api/docs/support/users#update-user
func (z *Client) UpdateUser(ctx context.Context, userID int64, user User) (User, error) {
	var data, result struct {
		User User `json:"user"`
	}
	data.User = user

	body, err := z.put(ctx, fmt.Sprintf("/users/%d.json", userID), data)
	if err != nil {
		return User{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return User{}, err
	}
	return result.User, nil
}

// GetUserRelated retrieves user related user information
// ref: https://developer.zendesk.com/api-reference/ticketing/users/users/#show-user-related-information
func (z *Client) GetUserRelated(ctx context.Context, userID int64) (UserRelated, error) {
	var data struct {
		UserRelated UserRelated `json:"user_related"`
	}

	body, err := z.get(ctx, fmt.Sprintf("/users/%d/related.json", userID))
	if err != nil {
		return UserRelated{}, err
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return UserRelated{}, err
	}

	return data.UserRelated, nil
}
