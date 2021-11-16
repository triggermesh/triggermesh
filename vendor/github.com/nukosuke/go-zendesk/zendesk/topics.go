package zendesk

import "time"

type Topic struct {
	ID            int64     `json:"id"`
	URL           string    `json:"url"`
	HTMLURL       string    `json:"html_url"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Position      int       `json:"position"`
	FollowerCount int       `json:"follower_count"`
	ManageableBy  string    `json:"manageable_by"`
	UserSegmentID int64     `json:"user_segment_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
