/*
Copyright 2020-2021 TriggerMesh Inc.

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

package zendesksource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgapis "knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/nukosuke/go-zendesk/zendesk"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/secret"
)

func (r *Reconciler) ensureZendeskTargetAndTrigger(ctx context.Context) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)
	status := &src.(*v1alpha1.ZendeskSource).Status

	isDeployed := status.GetCondition(v1alpha1.ConditionDeployed).IsTrue()
	url := status.Address.URL

	// skip this cycle if the URL couldn't yet be determined
	if !isDeployed || url == nil {
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonNoURL, "The receive adapter isn't ready yet")
		return nil
	}

	spec := src.(pkgapis.HasSpec).GetUntypedSpec().(v1alpha1.ZendeskSourceSpec)

	sg := secret.NewGetter(r.secretClient(src.GetNamespace()))

	secrets, err := sg.Get(spec.Token, spec.WebhookPassword)
	if err != nil {
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonNoSecret, "Cannot obtain Zendesk secrets")
		return fmt.Errorf("obtaining secrets: %w", err)
	}

	apiToken := secrets[0]
	webhookPassword := secrets[1]

	client, err := zendeskClient(spec.Email, spec.Subdomain, apiToken)
	if err != nil {
		return fmt.Errorf("getting Zendesk client: %w", err)
	}

	title := targetTitle(src)

	currentTarget, err := ensureTarget(ctx, status, client,
		desiredTarget(title, url.String(), spec.WebhookUsername, webhookPassword),
	)
	if err != nil {
		return err
	}

	err = ensureTrigger(ctx, status, client,
		desiredTrigger(title, strconv.FormatInt(currentTarget.ID, 10)),
	)
	if err != nil {
		return err
	}

	status.MarkTargetSynced()
	return nil
}

func desiredTarget(title, url, webhookUsername, webhookPassword string) *zendesk.Target {
	return &zendesk.Target{
		Title:       title,
		Type:        "url_target_v2",
		TargetURL:   url,
		Method:      "post",
		Username:    webhookUsername,
		Password:    webhookPassword,
		ContentType: "application/json",
	}
}

func desiredTrigger(title, targetID string) *zendesk.Trigger {
	trg := &zendesk.Trigger{
		Title: title,
		Actions: []zendesk.TriggerAction{{
			Field: "notification_target",
			Value: []interface{}{
				targetID,
				triggerPayloadJSON,
			},
		}},
	}
	trg.Conditions.All = []zendesk.TriggerCondition{{
		Field:    "update_type",
		Operator: "is",
		Value:    "Create",
	}}
	trg.Conditions.Any = make([]zendesk.TriggerCondition, 0)

	return trg
}

func ensureTarget(ctx context.Context, status *v1alpha1.ZendeskSourceStatus,
	client *zendesk.Client, desired *zendesk.Target) (*zendesk.Target, error) {

	// TODO: It could happen that the target already exists but is in a
	// different page. We will need to support pagination in a future
	// release of this source.
	targets, _, err := client.GetTargets(ctx)
	switch {
	case isDenied(err):
		return nil, controller.NewPermanentError(formatError(err))

	case err != nil:
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to list Targets")
		return nil, fmt.Errorf("retrieving Zendesk Targets: %w", formatError(err))
	}

	for _, t := range targets {
		if t.Title == desired.Title {
			target, err := syncTarget(ctx, client, &t, desired) //nolint:scopelint,gosec
			if err != nil {
				status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to update Target")
			}
			return target, err
		}
	}

	target, err := client.CreateTarget(ctx, *desired)
	if err != nil {
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to create Target")
		return nil, fmt.Errorf("creating Zendesk Target: %w", formatError(err))
	}

	event.Normal(ctx, ReasonTargetCreated, "Zendesk Target %q was created", target.Title)
	return &target, nil
}

func syncTarget(ctx context.Context, client *zendesk.Client, current, desired *zendesk.Target) (*zendesk.Target, error) {
	// copy fields which are set by the API
	desired.URL = current.URL
	desired.ID = current.ID
	desired.CreatedAt = current.CreatedAt
	desired.Active = true

	// GetTargets doesn't return the webhook password, so we exclude it
	// from the comparison
	desiredPw := desired.Password
	desired.Password = ""

	if *current == *desired {
		return current, nil
	}

	desired.Password = desiredPw

	target, err := client.UpdateTarget(ctx, current.ID, *desired)
	if err != nil {
		return nil, fmt.Errorf("updating Zendesk Target: %w", formatError(err))
	}

	event.Normal(ctx, ReasonTargetUpdated, "Zendesk Target %q was updated", current.Title)
	return &target, nil
}

func ensureTrigger(ctx context.Context, status *v1alpha1.ZendeskSourceStatus,
	client *zendesk.Client, desired *zendesk.Trigger) error {

	// TODO: It could happen that the trigger already exists but is in a
	// different page. We will need to support pagination in a future
	// release of this source.
	triggers, _, err := client.GetTriggers(ctx, &zendesk.TriggerListOptions{})
	if err != nil {
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to list Triggers")
		return fmt.Errorf("retrieving Zendesk Triggers: %w", formatError(err))
	}

	for _, t := range triggers {
		if t.Title == desired.Title {
			err := syncTrigger(ctx, client, &t, desired) //nolint:scopelint,gosec
			if err != nil {
				status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to update Trigger")
			}
			return err
		}
	}

	trigger, err := client.CreateTrigger(ctx, *desired)
	if err != nil {
		status.MarkTargetNotSynced(v1alpha1.ZendeskReasonFailedSync, "Unable to create Trigger")
		return fmt.Errorf("creating Zendesk Trigger: %w", formatError(err))
	}

	event.Normal(ctx, ReasonTargetCreated, "Zendesk Trigger %q was created", trigger.Title)
	return nil
}

func syncTrigger(ctx context.Context, client *zendesk.Client, current, desired *zendesk.Trigger) error {
	// copy fields which are set by the API
	desired.ID = current.ID
	desired.Position = current.Position
	desired.CreatedAt = current.CreatedAt
	desired.UpdatedAt = current.UpdatedAt
	desired.Active = true

	if cmp.Equal(current, desired) {
		return nil
	}

	if _, err := client.UpdateTrigger(ctx, current.ID, *desired); err != nil {
		return fmt.Errorf("updating Zendesk Trigger: %w", formatError(err))
	}

	event.Normal(ctx, ReasonTargetUpdated, "Zendesk Trigger %q was updated", current.Title)
	return nil
}

func (r *Reconciler) ensureNoZendeskTargetAndTrigger(ctx context.Context) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)

	title := targetTitle(src)

	spec := src.(pkgapis.HasSpec).GetUntypedSpec().(v1alpha1.ZendeskSourceSpec)

	sg := secret.NewGetter(r.secretClient(src.GetNamespace()))

	secrets, err := sg.Get(spec.Token)
	switch {
	case apierrors.IsNotFound(err):
		// the finalizer is unlikely to recover from a missing Secret,
		// so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedTargetDelete, "Secret missing while finalizing Zendesk Target %q. "+
			"Ignoring: %s", title, err)
		return nil

	case err != nil:
		return fmt.Errorf("reading Zendesk API token: %w", err)
	}

	apiToken := secrets[0]

	client, err := zendeskClient(spec.Email, spec.Subdomain, apiToken)
	if err != nil {
		return fmt.Errorf("getting Zendesk client: %w", err)
	}

	if err := ensureNoTrigger(ctx, client, title); err != nil {
		return err
	}

	return ensureNoTarget(ctx, client, title)
}

func ensureNoTrigger(ctx context.Context, client *zendesk.Client, title string) error {
	triggers, _, err := client.GetTriggers(ctx, &zendesk.TriggerListOptions{})
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return to
		// allow the reconciler to remove the finalizer
		event.Warn(ctx, ReasonFailedTargetDelete, "Authorization error finalizing Zendesk Trigger %q. "+
			"Ignoring: %s", title, formatError(err))
		return nil

	case isNotFound(err):
		event.Warn(ctx, ReasonFailedTargetDelete, "Resource not found while finalizing Zendesk Trigger %q. "+
			"Ignoring: %s", title, formatError(err))
		return nil

	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedTargetDelete,
			"Error retrieving Zendesk Triggers: %s", formatError(err))
	}

	var currentTrigger *zendesk.Trigger
	for _, t := range triggers {
		if t.Title == title {
			currentTrigger = &t //nolint:scopelint,exportloopref,gosec
			break
		}
	}
	if currentTrigger == nil {
		return nil
	}

	if err := client.DeleteTrigger(ctx, currentTrigger.ID); err != nil {
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedTargetDelete,
			"Error finalizing Zendesk Trigger %q: %s", title, formatError(err))
	}
	event.Normal(ctx, ReasonTargetDeleted, "Zendesk Trigger %q was deleted", title)

	return nil
}

func ensureNoTarget(ctx context.Context, client *zendesk.Client, title string) error {
	targets, _, err := client.GetTargets(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return to
		// allow the reconciler to remove the finalizer
		event.Warn(ctx, ReasonFailedTargetDelete, "Authorization error finalizing Zendesk Target %q. "+
			"Ignoring: %s", title, formatError(err))
		return nil

	case isNotFound(err):
		event.Warn(ctx, ReasonFailedTargetDelete, "Resource not found while finalizing Zendesk Target %q. "+
			"Ignoring: %s", title, formatError(err))
		return nil

	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedTargetDelete,
			"Error retrieving Zendesk Targets: %s", formatError(err))
	}

	var currentTarget *zendesk.Target
	for _, t := range targets {
		if t.Title == title {
			currentTarget = &t //nolint:scopelint,exportloopref,gosec
			break
		}
	}
	if currentTarget == nil {
		return nil
	}

	if err := client.DeleteTarget(ctx, currentTarget.ID); err != nil {
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedTargetDelete,
			"Error finalizing Zendesk Target %q: %s", title, formatError(err))
	}
	event.Normal(ctx, ReasonTargetDeleted, "Zendesk Target %q was deleted", title)

	return nil
}

// targetTitle returns a Zendesk Target/Trigger title suitable for the given
// source object.
func targetTitle(src metav1.Object) string {
	return "io.triggermesh.zendesksource." + src.GetNamespace() + "." + src.GetName()
}

// zendeskClient returns an initialized Zendesk client.
func zendeskClient(email, subdomain, apiToken string) (*zendesk.Client, error) {
	cred := zendesk.NewAPITokenCredential(email, apiToken)
	client, err := zendesk.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Zendesk client: %w", err)
	}
	if err := client.SetSubdomain(subdomain); err != nil {
		return nil, fmt.Errorf("setting Zendesk subdomain: %w", err)
	}
	client.SetCredential(cred)

	return client, nil
}

// isDenied returns whether the given error indicates that a request was denied
// due to authentication issues.
func isDenied(err error) bool {
	if zdErr := (zendesk.Error{}); errors.As(err, &zdErr) {
		s := zdErr.Status()
		return s == http.StatusUnauthorized || s == http.StatusForbidden
	}
	return false
}

// isNotFound returns whether the given error indicates that a Zendesk resource
// (account, target, trigger) does not exist.
func isNotFound(err error) bool {
	if zdErr := (zendesk.Error{}); errors.As(err, &zdErr) {
		return zdErr.Status() == http.StatusNotFound
	}
	return false
}

// formatError formats Zendesk errors.
func formatError(origErr error) error {
	if zdErr := (zendesk.Error{}); errors.As(origErr, &zdErr) {
		rawErrBody, err := ioutil.ReadAll(zdErr.Body())
		if err != nil {
			return origErr
		}

		fmtErr := &zendeskError{}
		if err := json.Unmarshal(rawErrBody, fmtErr); err != nil {
			return origErr
		}

		return fmtErr
	}

	return origErr
}

// zendeskError represents the payload returned with Zendesk API errors.
type zendeskError struct {
	Err struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	} `json:"error"`
}

// Error implements the error interface.
// Returns the Zendesk error body in a human-readable format.
func (b *zendeskError) Error() string {
	return b.Err.Title + ": " + b.Err.Message
}

const triggerPayloadJSON = `{
  "ticket": {
    "id": {{ticket.id}},
    "external_id": "{{ticket.external_id}}",
    "title": "{{ticket.title}}",
    "url": "{{ticket.url}}",
    "description": "{{ticket.description}}",
    "via": "{{ticket.via}}",
    "status": "{{ticket.status}}",
    "priority": "{{ticket.priority}}",
    "ticket_type": "{{ticket.ticket_type}}",
    "group_name": "{{ticket.group.name}}",
    "brand_name": "{{ticket.brand.name}}",
    "due_date": "{{ticket.due_date}}",
    "account": "{{ticket.account}}",
    "assignee": {
      "email": "{{ticket.assignee.email}}",
      "name": "{{ticket.assignee.name}}",
      "first_name": "{{ticket.assignee.first_name}}",
      "last_name": "{{ticket.assignee.last_name}}"
    },
    "requester": {
      "name": "{{ticket.requester.name}}",
      "first_name": "{{ticket.requester.first_name}}",
      "last_name": "{{ticket.requester.last_name}}",
      "email": "{{ticket.requester.email}}",
      "language": "{{ticket.requester.language}}",
      "phone": "{{ticket.requester.phone}}",
      "external_id": "{{ticket.requester.external_id}}",
      "field": "{{ticket.requester_field}}",
      "details": "{{ticket.requester.details}}"
    },
    "organization": {
      "name": "{{ticket.organization.name}}",
      "external_id": "{{ticket.organization.external_id}}",
      "details": "{{ticket.organization.details}}",
      "notes": "{{ticket.organization.notes}}"
    },
    "ccs": "{{ticket.ccs}}",
    "cc_names": "{{ticket.cc_names}}",
    "tags": "{{ticket.tags}}",
    "current_holiday_name": "{{ticket.current_holiday_name}}",
    "ticket_field_id": "{{ticket.ticket_field_ID}}",
    "ticket_field_option_title_id": "{{ticket.ticket_field_option_title_ID}}"
  },
  "current_user": {
    "name": "{{current_user.name}}",
    "first_name": "{{current_user.first_name}}",
    "email": "{{current_user.email}}",
    "organization": {
      "name": "{{current_user.organization.name}}",
      "notes": "{{current_user.organization.notes}}",
      "details": "{{current_user.organization.details}}"
    },
    "external_id": "{{current_user.external_id}}",
    "phone": "{{current_user.phone}}",
    "details": "{{current_user.details}}",
    "notes": "{{current_user.notes}}",
    "language": "{{current_user.language}}"
  },
  "satisfaction": {
    "current_rating": "{{satisfaction.current_rating}}",
    "current_comment": "{{satisfaction.current_comment}}"
  }
}`
