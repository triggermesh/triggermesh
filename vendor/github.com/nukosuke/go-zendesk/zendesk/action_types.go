package zendesk

// action field types which defined by system
// https://developer.zendesk.com/rest_api/docs/core/triggers#actions-reference
const (
	// ActionFieldStatus status
	ActionFieldStatus = iota
	// ActionFieldType type
	ActionFieldType
	// ActionFieldPriority priority
	ActionFieldPriority
	// ActionFieldGroupID group_id
	ActionFieldGroupID
	// ActionFieldAssigneeID assignee_id
	ActionFieldAssigneeID
	// ActionFieldSetTags set_tags
	ActionFieldSetTags
	// ActionFieldCurrentTags current_tags
	ActionFieldCurrentTags
	// ActionFieldRemoveTags remove_tags
	ActionFieldRemoveTags
	// ActionFieldSatisfactionScore satisfaction_score
	ActionFieldSatisfactionScore
	// ActionFieldNotificationUser notification_user
	ActionFieldNotificationUser
	// ActionFieldNotificationGroup notification_group
	ActionFieldNotificationGroup
	// ActionFieldNotificationTarget notification_target
	ActionFieldNotificationTarget
	// ActionFieldTweetRequester tweet_requester
	ActionFieldTweetRequester
	// ActionFieldCC cc
	ActionFieldCC
	// ActionFieldLocaleID locale_id
	ActionFieldLocaleID
	// ActionFieldSubject subject
	ActionFieldSubject
	// ActionFieldCommentValue comment_value
	ActionFieldCommentValue
	// ActionFieldCommentValueHTML comment_value_html
	ActionFieldCommentValueHTML
	// ActionFieldCommentModeIsPublic comment_mode_is_public
	ActionFieldCommentModeIsPublic
	// ActionFieldTicketFormID ticket_form_id
	ActionFieldTicketFormID
)

var actionFieldText = map[int]string{
	ActionFieldStatus:              "status",
	ActionFieldType:                "type",
	ActionFieldPriority:            "priority",
	ActionFieldGroupID:             "group_id",
	ActionFieldAssigneeID:          "assignee_id",
	ActionFieldSetTags:             "set_tags",
	ActionFieldCurrentTags:         "current_tags",
	ActionFieldRemoveTags:          "remove_tags",
	ActionFieldSatisfactionScore:   "satisfaction_score",
	ActionFieldNotificationUser:    "notification_user",
	ActionFieldNotificationGroup:   "notification_group",
	ActionFieldNotificationTarget:  "notification_target",
	ActionFieldTweetRequester:      "tweet_requester",
	ActionFieldCC:                  "cc",
	ActionFieldLocaleID:            "locale_id",
	ActionFieldSubject:             "subject",
	ActionFieldCommentValue:        "comment_value",
	ActionFieldCommentValueHTML:    "comment_value_html",
	ActionFieldCommentModeIsPublic: "comment_mode_is_public",
	ActionFieldTicketFormID:        "ticket_form_id",
}

// ActionFieldText takes field type and returns field name string
func ActionFieldText(fieldType int) string {
	return actionFieldText[fieldType]
}
