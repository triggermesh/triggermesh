/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awseventbridgesource

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/triggermesh/triggermesh/pkg/apis"
	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/aws/sqs"
)

const (
	eventbusARNResourcePrefix = "event-bus/"
	ruleARNResourcePrefix     = "rule/"
)

const awsTagOwner = "owned-by"

// EnsureRule ensures that an EventBrige event rule exists for the event bus.
//
// Required permissions:
// - events:ListRuleNamesByTarget
// - events:ListTagsForResource
// - events:DescribeRule
// - events:PutRule
// - events:TagResource
func EnsureRule(ctx context.Context,
	cli eventbridgeiface.EventBridgeAPI, queue *sqsQueue) (*apis.ARN /*rule*/, error) {

	if skip.Skip(ctx) {
		return nil, nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	status := &typedSrc.Status

	currentRule, err := getRule(ctx, cli, typedSrc, queue.arn)
	switch {
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonNoEventBus, "Event bus does not exist")
		return nil, controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"The event bus does not exist: %s", toErrMsg(err)))
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to EventBridge API got rejected")
		return nil, controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error getting event rule: %s", toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot obtain current event rule")
		// wrap any other error to fail the reconciliation
		return nil, fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error getting current event rule: %s", toErrMsg(err)))
	}

	desiredRule := makeRule(typedSrc)

	if equalRules(ctx, desiredRule, currentRule) {
		ruleARN, err := arnStrToARN(*currentRule.Arn)
		if err != nil {
			return nil, fmt.Errorf("converting ARN string to structured ARN: %w", err)
		}

		status.MarkSubscribed(*ruleARN)
		return ruleARN, nil
	}

	ruleName := desiredRule.Name
	if currentRule != nil {
		ruleName = currentRule.Name
	}

	in := &eventbridge.PutRuleInput{
		Name:         ruleName,
		Description:  desiredRule.Description,
		EventBusName: desiredRule.EventBusName,
		EventPattern: desiredRule.EventPattern,
		Tags:         ruleTags(src),
	}

	rule, err := cli.PutRuleWithContext(ctx, in)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to EventBridge API got rejected")
		return nil, controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error creating/updating event rule: %s", toErrMsg(err)))
	case isInvalidEventPatternError(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonInvalidEventPattern, "Provided event pattern in invalid")
		return nil, controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error creating rule with an invalid event pattern: %s", toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot create/update event rule")
		return nil, fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error creating/updating event rule: %s", toErrMsg(err)))
	}

	event.Normal(ctx, ReasonSubscribed, "Configured rule %q for event bus %q", *in.Name, typedSrc.Spec.ARN)

	ruleARN, err := arnStrToARN(*rule.RuleArn)
	if err != nil {
		return nil, fmt.Errorf("converting ARN string to structured ARN: %w", err)
	}
	status.MarkSubscribed(*ruleARN)

	return ruleARN, nil
}

// SetRuleTarget applies a SQS target to the given rule.
//
// Required permissions:
// - events:ListTargetsByRule
// - events:PutTargets
func SetRuleTarget(ctx context.Context, cli eventbridgeiface.EventBridgeAPI, ruleARN *apis.ARN, queueARN string) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	status := &typedSrc.Status

	eventbusARN := typedSrc.Spec.ARN

	eventbusName := eventbusName(eventbusARN)
	ruleName := ruleName(*ruleARN, eventbusName)

	currentTargets, err := allTargetsForRule(ctx, cli, eventbusName, ruleName)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to EventBridge API got rejected")
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error getting targets for rule %q: %s", ruleName, toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot obtain current targets")
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error getting current targets for rule %q: %s", ruleName, toErrMsg(err)))
	}

	trgtsToPut := targetsToPut(currentTargets, makeTarget(queueARN, src))
	if len(trgtsToPut) == 0 {
		return nil
	}

	in := &eventbridge.PutTargetsInput{
		EventBusName: &eventbusName,
		Rule:         &ruleName,
		Targets:      trgtsToPut,
	}

	_, err = cli.PutTargetsWithContext(ctx, in)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to EventBridge API got rejected")
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error updating targets for rule %q: %s", ruleName, toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot update targets")
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error updating targets for rule %q: %s", ruleName, toErrMsg(err)))
	}

	event.Normal(ctx, ReasonSubscribed, "Updated target for rule %q", ruleName)

	return nil
}

// EnsureNoRule ensures that the EventBrige event rule is removed from the
// event bus.
//
// Required permissions:
// - sqs:GetQueueUrl
// - sqs:GetQueueAttributes
// - events:ListRuleNamesByTarget
// - events:ListTagsForResource
// - events:DescribeRule
// - events:ListTargetsByRule
// - events:RemoveTargets
// - events:DeleteRule
func EnsureNoRule(ctx context.Context, cli eventbridgeiface.EventBridgeAPI,
	sqsCli sqsiface.SQSAPI) (string /*queue name*/, error) {

	if skip.Skip(ctx) {
		return "", nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	// find queue

	queueARN, err := currentQueueTarget(ctx, sqsCli)
	switch {
	case isNotFound(err) || isDenied(err):
		// acceptable here, we might have a fallback value to work with
	case err != nil:
		// other errors are expected to be retriable
		return "", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error finding ARN of current queue target: %s", toErrMsg(err))
	}

	if queueARN == nil {
		return "", reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed,
			"Current queue target could not be determined, skipping finalization")
	}

	queueName := queueARN.Resource

	// find rule

	currentRule, err := getRule(ctx, cli, typedSrc, queueARN.String())
	switch {
	case isNotFound(err):
		event.Normal(ctx, ReasonUnsubscribed, "The event bus does not exist, skipping finalization")
		return queueName, nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error getting event rule. Ignoring: %s", toErrMsg(err))
		return queueName, nil
	case err != nil:
		return "", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error reading current event rule: %s", toErrMsg(err))
	}

	if currentRule == nil {
		event.Normal(ctx, ReasonUnsubscribed, "The rule could not be found, skipping finalization")
		return queueName, nil
	}

	ruleARN, err := arnStrToARN(*currentRule.Arn)
	if err != nil {
		return "", fmt.Errorf("converting ARN string to structured ARN: %w", err)
	}

	// find targets

	eventbusARN := typedSrc.Spec.ARN

	eventbusName := eventbusName(eventbusARN)
	ruleName := ruleName(*ruleARN, eventbusName)

	currentTargets, err := allTargetsForRule(ctx, cli, eventbusName, ruleName)
	switch {
	case isDenied(err):
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error getting targets for rule %q. Ignoring: %s", ruleName, toErrMsg(err))
		return queueName, nil
	case err != nil:
		return "", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error getting current targets for rule %q: %s", ruleName, toErrMsg(err))
	}

	// remove targets

	if len(currentTargets) > 0 {
		var currentTargetsIDs []string
		for _, t := range currentTargets {
			currentTargetsIDs = append(currentTargetsIDs, *t.Id)
		}

		rmTargetsInput := &eventbridge.RemoveTargetsInput{
			EventBusName: &eventbusName,
			Rule:         &ruleName,
			Ids:          aws.StringSlice(currentTargetsIDs),
		}

		_, err = cli.RemoveTargetsWithContext(ctx, rmTargetsInput)
		switch {
		case isDenied(err):
			event.Warn(ctx, ReasonFailedUnsubscribe,
				"Authorization error removing targets from rule %q. Ignoring: %s", ruleName, toErrMsg(err))
			return queueName, nil
		case err != nil:
			return "", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
				"Error removing targets from rule %q: %s", ruleName, toErrMsg(err))
		}
	}

	// remove rule

	rmRuleInput := &eventbridge.DeleteRuleInput{
		EventBusName: &eventbusName,
		Name:         &ruleName,
	}

	_, err = cli.DeleteRuleWithContext(ctx, rmRuleInput)
	switch {
	case isDenied(err):
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error getting targets for rule %q. Ignoring: %s", ruleName, toErrMsg(err))
		return queueName, nil
	case err != nil:
		return "", fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error getting current targets for rule %q: %s", ruleName, toErrMsg(err)))
	}

	event.Normal(ctx, ReasonUnsubscribed,
		"Removed rule %q from event bus %q", *currentRule.Name, eventbusARN)

	return queueName, nil
}

// getRule looks up the current event rule based on the given event bus and queue.
func getRule(ctx context.Context, cli eventbridgeiface.EventBridgeAPI,
	src *v1alpha1.AWSEventBridgeSource, queueARN string) (*eventbridge.Rule, error) {

	eventbusARN := src.Spec.ARN

	matchingRules, err := allRulesForTarget(ctx, cli, eventbusName(eventbusARN), queueARN)
	if err != nil {
		return nil, err
	}

	if len(matchingRules) == 0 {
		// edge case: allRulesForTarget returned no match because the
		// target was deleted, but the rule may still exist. Attempt a
		// match on the rule ARN from the source's status, if set.
		if src.Status.EventRuleARN == nil {
			return nil, nil
		}

		matchingRules = []string{
			ruleName(*src.Status.EventRuleARN, eventbusName(eventbusARN)),
		}
	}

	for _, rule := range matchingRules {
		ruleARN := eventbusARN
		ruleARN.Resource = ruleARNResourcePrefix + eventbusName(eventbusARN) + "/" + rule

		owns, err := assertResourceOwnership(ctx, cli, ruleARN.String(), src)
		if err != nil {
			return nil, fmt.Errorf("asserting ownership of event rule %q: %w", rule, err)
		}
		if !owns {
			continue
		}

		in := &eventbridge.DescribeRuleInput{
			EventBusName: aws.String(eventbusName(eventbusARN)),
			Name:         &rule,
		}

		out, err := cli.DescribeRuleWithContext(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("describing event rule %q: %w", rule, err)
		}

		outRule := &eventbridge.Rule{
			Arn:                out.Arn,
			Description:        out.Description,
			EventBusName:       out.EventBusName,
			EventPattern:       out.EventPattern,
			ManagedBy:          out.ManagedBy,
			Name:               out.Name,
			RoleArn:            out.RoleArn,
			ScheduleExpression: out.ScheduleExpression,
			State:              out.State,
		}

		return outRule, nil
	}

	return nil, nil
}

// allRulesForTarget returns the names of all the existing event rules of the
// given event bus that match the specified SQS target.
func allRulesForTarget(ctx context.Context, cli eventbridgeiface.EventBridgeAPI,
	eventbusName, queueARN string) ([]string, error) {

	var matchingRules []string

	in := &eventbridge.ListRuleNamesByTargetInput{
		EventBusName: &eventbusName,
		TargetArn:    &queueARN,
	}

	out := &eventbridge.ListRuleNamesByTargetOutput{}

	var err error

	initialRequest := true

	for out.NextToken != nil || initialRequest {
		in.NextToken = out.NextToken

		out, err = cli.ListRuleNamesByTargetWithContext(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("listing rules with target %q: %w", queueARN, err)
		}

		if initialRequest {
			initialRequest = false
		}

		matchingRules = append(matchingRules, aws.StringValueSlice(out.RuleNames)...)
	}

	return matchingRules, nil
}

// allTargetsForRule returns all targets of the given rule.
func allTargetsForRule(ctx context.Context, cli eventbridgeiface.EventBridgeAPI,
	eventbusName, ruleName string) ([]*eventbridge.Target, error) {

	var targets []*eventbridge.Target

	in := &eventbridge.ListTargetsByRuleInput{
		EventBusName: &eventbusName,
		Rule:         &ruleName,
	}

	out := &eventbridge.ListTargetsByRuleOutput{}

	var err error

	initialRequest := true

	for out.NextToken != nil || initialRequest {
		in.NextToken = out.NextToken

		out, err = cli.ListTargetsByRuleWithContext(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("listing targets for rule %q: %w", ruleName, err)
		}

		if initialRequest {
			initialRequest = false
		}

		targets = append(targets, out.Targets...)
	}

	return targets, nil
}

// makeRule returns an event rule for the given source.
func makeRule(src *v1alpha1.AWSEventBridgeSource) *eventbridge.Rule {
	// Rule names can contain a maximum of 64 characters consisting of
	// alphanumeric characters, periods (.), hyphens (-) and underscores (_).
	// For convenience, we use the same naming convention as the SQS queue.
	ruleName := queueName

	eventPattern := `{"account":["` + src.Spec.ARN.AccountID + `"]}` // catch-all
	if ep := src.Spec.EventPattern; ep != nil {
		eventPattern = *ep
	}

	return &eventbridge.Rule{
		Name:         aws.String(ruleName(src)),
		Description:  aws.String("Managed by TriggerMesh"),
		EventBusName: aws.String(eventbusName(src.Spec.ARN)),
		EventPattern: aws.String(eventPattern),
	}
}

// makeTarget returns a target for the given SQS queue.
func makeTarget(queueARN string, src commonv1alpha1.Reconcilable) *eventbridge.Target {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.GetNamespace() + "/" + src.GetName()))

	return &eventbridge.Target{
		Arn: &queueARN,
		Id:  aws.String(strconv.FormatUint(uint64(nsNameChecksum), 10)),
	}
}

// targetsToPut returns a list of rule targets that should be created/updated,
// based on the given list of existing targets.
func targetsToPut(currentTrgts []*eventbridge.Target, desiredTrgt *eventbridge.Target) []*eventbridge.Target {
	for _, t := range currentTrgts {
		if *t.Id == *desiredTrgt.Id {
			if *t.Arn != *desiredTrgt.Arn {
				return []*eventbridge.Target{t}
			}
			return nil
		}
	}

	return []*eventbridge.Target{desiredTrgt}
}

// equalRules asserts the equality of two rules.
func equalRules(ctx context.Context, desired, current *eventbridge.Rule) bool {
	cmpFn := cmp.Equal
	if logger := logging.FromContext(ctx); logger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		cmpFn = diffLoggingCmp(logger)
	}

	return cmpFn(desired, current,
		cmpopts.IgnoreFields(eventbridge.Rule{}, "State", "Name", "Arn"),
	)
}

// cmpFunc can compare the equality of two interfaces. The function signature
// is the same as cmp.Equal.
type cmpFunc func(x, y interface{}, opts ...cmp.Option) bool

// diffLoggingCmp compares the equality of two interfaces and logs the diff at
// the Debug level.
func diffLoggingCmp(logger *zap.SugaredLogger) cmpFunc {
	return func(desired, current interface{}, opts ...cmp.Option) bool {
		if diff := cmp.Diff(desired, current, opts...); diff != "" {
			logger.Debug("Rules differ (-desired, +current)\n" + diff)
			return false
		}
		return true
	}
}

// assertResourceOwnership returns whether an EventBridge resource is owned by
// the given source.
func assertResourceOwnership(ctx context.Context, cli eventbridgeiface.EventBridgeAPI,
	resourceARN string, src commonv1alpha1.Reconcilable) (bool, error) {

	in := &eventbridge.ListTagsForResourceInput{
		ResourceARN: &resourceARN,
	}

	out, err := cli.ListTagsForResourceWithContext(ctx, in)
	if err != nil {
		return false, fmt.Errorf("listing tags for resource %q: %w", resourceARN, err)
	}

	for _, tag := range out.Tags {
		if *tag.Key == awsTagOwner {
			return *tag.Value == sourceID(src), nil
		}
	}

	return false, nil
}

// ruleTags returns a set of tags containing information from the given source
// instance to set on an EventBridge event rule.
func ruleTags(src commonv1alpha1.Reconcilable) []*eventbridge.Tag {
	return []*eventbridge.Tag{{
		Key:   aws.String(awsTagOwner),
		Value: aws.String(sourceID(src)),
	}}
}

// eventbusName extracts the name of the EventBrige event bus from its given ARN.
func eventbusName(arn apis.ARN) string {
	return strings.TrimPrefix(arn.Resource, eventbusARNResourcePrefix)
}

// ruleName extracts the name of the EventBrige event rule from its given ARN.
func ruleName(arn apis.ARN, eventbusName string) string {
	return strings.TrimPrefix(arn.Resource, ruleARNResourcePrefix+eventbusName+"/")
}

// currentQueueTarget returns the ARN of the SQS queue that is currently used
// as target of the rule.
// This function may return both an error and a valid ARN to indicate that the
// returned value is a fallback value (e.g. value read from the source's status
// instead of from AWS). The caller decides how to handle that error.
func currentQueueTarget(ctx context.Context, cli sqsiface.SQSAPI) (*apis.ARN, error) {
	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	if dest := typedSrc.Spec.Destination; dest != nil {
		if userProvidedQueue := dest.SQS; userProvidedQueue != nil {
			return &userProvidedQueue.QueueARN, nil
		}
	}

	var queueARN *apis.ARN
	if qa := typedSrc.Status.QueueARN; qa != nil {
		queueARN = qa
	}

	queueName := queueName(typedSrc)

	// Prefer getting the queue's ARN from AWS directly if it still exists,
	// as it is more reliable than what may be stored in the source's status.
	queueURL, err := sqs.QueueURL(cli, queueName)
	if err != nil {
		return queueARN, fmt.Errorf("getting SQS queue: %w", err)
	}

	getAttrs := []string{awssqs.QueueAttributeNameQueueArn}
	queueAttrs, err := sqs.QueueAttributes(cli, queueURL, getAttrs)
	if err != nil {
		return queueARN, fmt.Errorf("getting attributes of SQS queue: %w", err)
	}

	queueARNAttr := queueAttrs[awssqs.QueueAttributeNameQueueArn]

	queueARNStruct, err := arnStrToARN(queueARNAttr)
	if err != nil {
		return queueARN, fmt.Errorf("converting ARN string to structured ARN: %w", err)
	}

	return queueARNStruct, nil
}

// isNotFound returns whether the given error indicates that some resource was
// not found.
func isNotFound(err error) bool {
	if k8sErr := apierrors.APIStatus(nil); errors.As(err, &k8sErr) {
		return k8sErr.Status().Reason == metav1.StatusReasonNotFound
	}
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		errcode := awsErr.Code()
		return errcode == awssqs.ErrCodeQueueDoesNotExist ||
			errcode == eventbridge.ErrCodeResourceNotFoundException
	}
	return false
}

// isDenied returns whether the given error indicates that a request to the AWS
// API could not be authorized.
func isDenied(err error) bool {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		if awsErr == credentials.ErrStaticCredentialsEmpty {
			return true
		}

		if awsReqFail := awserr.RequestFailure(nil); errors.As(err, &awsReqFail) {
			code := awsReqFail.StatusCode()
			return code == http.StatusUnauthorized || code == http.StatusForbidden
		}
	}
	return false
}

// isInvalidEventPatternError returns whether the given error indicates that
// the provided event pattern in invalid.
func isInvalidEventPatternError(err error) bool {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		return awsErr.Code() == eventbridge.ErrCodeInvalidEventPatternException
	}
	return false
}

// toErrMsg attempts to extract the message from the given error if it is an
// AWS error.
// Those errors are particularly verbose and include a unique request ID that
// causes an infinite loop of reconciliations when appended to a status
// condition. Some AWS errors are not recoverable without manual intervention
// (e.g. invalid secrets) so there is no point letting that behaviour happen.
func toErrMsg(err error) string {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		return awserr.SprintError(awsErr.Code(), awsErr.Message(), "", awsErr.OrigErr())
	}
	return err.Error()
}
