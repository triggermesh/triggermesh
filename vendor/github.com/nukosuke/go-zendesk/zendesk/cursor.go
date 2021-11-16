package zendesk

// Cursor is struct for cursor-based pagination
type Cursor struct {
	AfterURL     string `json:"after_url"`
	AfterCursor  string `json:"after_cursor"`
	BeforeURL    string `json:"before_url"`
	BeforeCursor string `json:"before_cursor"`
}

// CursorOption is options for list methods for cursor-based pagination resources
// It's used to create query string.
//
// https://developer.zendesk.com/rest_api/docs/support/incremental_export#cursor-based-incremental-exports
type CursorOption struct {
	StartTime int64  `url:"start_time,omitempty"`
	Cursor    string `url:"cursor,omitempty"`
}
