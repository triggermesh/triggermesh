package zendesk

// condition field types which are defined by system
// https://developer.zendesk.com/rest_api/docs/core/triggers#conditions-reference
const (
	// ConditionFieldGroupID group_id
	ConditionFieldGroupID = iota
	// ConditionFieldAssigneeID assignee_id
	ConditionFieldAssigneeID
	// ConditionFieldRequesterID requester_id
	ConditionFieldRequesterID
	// ConditionFieldOrganizationID organization_id
	ConditionFieldOrganizationID
	// ConditionFieldCurrentTags current_tags
	ConditionFieldCurrentTags
	// ConditionFieldViaID via_id
	ConditionFieldViaID
	// ConditionFieldRecipient recipient
	ConditionFieldRecipient
	// ConditionFieldType type
	ConditionFieldType
	// ConditionFieldStatus status
	ConditionFieldStatus
	// ConditionFieldPriority priority
	ConditionFieldPriority
	// ConditionFieldDescriptionIncludesWord description_includes_word
	ConditionFieldDescriptionIncludesWord
	// ConditionFieldLocaleID locale_id
	ConditionFieldLocaleID
	// ConditionFieldSatisfactionScore satisfaction_score
	ConditionFieldSatisfactionScore
	// ConditionFieldSubjectIncludesWord subject_includes_word
	ConditionFieldSubjectIncludesWord
	// ConditionFieldCommentIncludesWord comment_includes_word
	ConditionFieldCommentIncludesWord
	// ConditionFieldCurrentViaID current_via_id
	ConditionFieldCurrentViaID
	// ConditionFieldUpdateType update_type
	ConditionFieldUpdateType
	// ConditionFieldCommentIsPublic comment_is_public
	ConditionFieldCommentIsPublic
	// ConditionFieldTicketIsPublic ticket_is_public
	ConditionFieldTicketIsPublic
	// ConditionFieldReopens reopens
	ConditionFieldReopens
	// ConditionFieldReplies
	ConditionFieldReplies
	// ConditionFieldAgentStations agent_stations
	ConditionFieldAgentStations
	// ConditionFieldGroupStations group_stations
	ConditionFieldGroupStations
	// ConditionFieldInBusinessHours in_business_hours
	ConditionFieldInBusinessHours
	// ConditionFieldRequesterTwitterFollowersCount requester_twitter_followers_count
	ConditionFieldRequesterTwitterFollowersCount
	// ConditionFieldRequesterTwitterStatusesCount requester_twitter_statuses_count
	ConditionFieldRequesterTwitterStatusesCount
	// ConditionFieldRequesterTwitterVerified requester_twitter_verified
	ConditionFieldRequesterTwitterVerified
	// ConditionFieldTicketTypeID ticket_type_id
	ConditionFieldTicketTypeID
	// ConditionFieldExactCreatedAt exact_created_at
	ConditionFieldExactCreatedAt
	// ConditionFieldNew NEW
	ConditionFieldNew
	// ConditionFieldOpen OPEN
	ConditionFieldOpen
	// ConditionFieldPending PENDING
	ConditionFieldPending
	// ConditionFieldSolved SOLVED
	ConditionFieldSolved
	// ConditionFieldClosed CLOSED
	ConditionFieldClosed
	// ConditionFieldAssignedAt assigned_at
	ConditionFieldAssignedAt
	// ConditionFieldUpdatedAt updated_at
	ConditionFieldUpdatedAt
	// ConditionFieldRequesterUpdatedAt requester_updated_at
	ConditionFieldRequesterUpdatedAt
	// ConditionFieldAssigneeUpdatedAt
	ConditionFieldAssigneeUpdatedAt
	// ConditionFieldDueDate due_date
	ConditionFieldDueDate
	// ConditionFieldUntilDueDate until_due_date
	ConditionFieldUntilDueDate
)

var conditionFieldText = map[int]string{
	ConditionFieldGroupID:                        "group_id",
	ConditionFieldAssigneeID:                     "assignee_id",
	ConditionFieldRequesterID:                    "requester_id",
	ConditionFieldOrganizationID:                 "organization_id",
	ConditionFieldCurrentTags:                    "current_tags",
	ConditionFieldViaID:                          "via_id",
	ConditionFieldRecipient:                      "recipient",
	ConditionFieldType:                           "type",
	ConditionFieldStatus:                         "status",
	ConditionFieldPriority:                       "priority",
	ConditionFieldDescriptionIncludesWord:        "description_includes_word",
	ConditionFieldLocaleID:                       "locale_id",
	ConditionFieldSatisfactionScore:              "satisfaction_score",
	ConditionFieldSubjectIncludesWord:            "subject_includes_word",
	ConditionFieldCommentIncludesWord:            "comment_includes_word",
	ConditionFieldCurrentViaID:                   "current_via_id",
	ConditionFieldUpdateType:                     "update_type",
	ConditionFieldCommentIsPublic:                "comment_is_public",
	ConditionFieldTicketIsPublic:                 "ticket_is_public",
	ConditionFieldReopens:                        "reopens",
	ConditionFieldReplies:                        "replies",
	ConditionFieldAgentStations:                  "agent_stations",
	ConditionFieldGroupStations:                  "group_stations",
	ConditionFieldInBusinessHours:                "in_business_hours",
	ConditionFieldRequesterTwitterFollowersCount: "requester_twitter_followers_count",
	ConditionFieldRequesterTwitterStatusesCount:  "requester_twitter_statuses_count",
	ConditionFieldRequesterTwitterVerified:       "requester_twitter_verified",
	ConditionFieldTicketTypeID:                   "ticket_type_id",
	ConditionFieldExactCreatedAt:                 "exact_created_at",
	ConditionFieldNew:                            "NEW",
	ConditionFieldOpen:                           "OPEN",
	ConditionFieldPending:                        "PENDING",
	ConditionFieldSolved:                         "SOLVED",
	ConditionFieldClosed:                         "CLOSED",
	ConditionFieldAssignedAt:                     "assigned_at",
	ConditionFieldUpdatedAt:                      "updated_at",
	ConditionFieldRequesterUpdatedAt:             "requester_updated_at",
	ConditionFieldAssigneeUpdatedAt:              "assignee_updated_at",
	ConditionFieldDueDate:                        "due_date",
	ConditionFieldUntilDueDate:                   "until_due_date",
}

// ConditionFieldText takes field type and returns field name string
func ConditionFieldText(fieldType int) string {
	return conditionFieldText[fieldType]
}
