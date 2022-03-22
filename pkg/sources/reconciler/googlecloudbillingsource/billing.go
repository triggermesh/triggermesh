/*
Copyright 2021 TriggerMesh Inc.

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

package googlecloudbillingsource

import (
	"context"
	"fmt"

	billing "cloud.google.com/go/billing/budgets/apiv1"
	budgets "google.golang.org/genproto/googleapis/cloud/billing/budgets/v1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

// ensureBudgetNotification ensures the existence of a notification
// configuration targetting the given Pub/Sub topic on a Cloud Billing budget.
// Required permissions:
// - billing.budgets.get
// - billing.budgets.update
func ensureBudgetNotification(ctx context.Context, cli *billing.BudgetClient, topicResName *v1alpha1.GCloudResourceName) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudBillingSource)
	status := &src.Status

	budgetRequest := &budgets.GetBudgetRequest{
		Name: generateBudgetID(src),
	}

	budget, err := cli.GetBudget(ctx, budgetRequest)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Billing API: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Budget does not exists: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Cannot obtain budget configuration %q: %s", src.Spec.BudgetID, toErrMsg(err))
	}

	// SchemaVersion Only "1.0" is accepted. It represents the JSON schema as defined in
	// https://cloud.google.com/billing/docs/how-to/budgets-programmatic-notifications#notification_format.
	budget.NotificationsRule = &budgets.NotificationsRule{
		SchemaVersion: "1.0",
		PubsubTopic:   generateTopicResourceName(src, topicResName.Resource),
	}
	budgetUpdatedRequest := budgets.UpdateBudgetRequest{
		Budget: budget,
	}

	_, err = cli.UpdateBudget(ctx, &budgetUpdatedRequest)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Billing API: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Budget does not exists: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Cannot create budget notification %q: %s", src.Spec.BudgetID, toErrMsg(err))
	}

	event.Normal(ctx, ReasonSubscribed, "Created Billing budget notification %q", budget.DisplayName)
	status.MarkSubscribed()
	return nil
}

// ensureNoBudgetNotification ensures that the notification
// configuration is deleted.
// Required permissions:
// - billing.budgets.get
// - billing.budgets.update
func ensureNoBudgetNotification(ctx context.Context, cli *billing.BudgetClient) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudBillingSource)
	status := &src.Status

	budgetRequest := &budgets.GetBudgetRequest{
		Name: generateBudgetID(src),
	}

	budget, err := cli.GetBudget(ctx, budgetRequest)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Billing API: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Budget does not exist: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Cannot obtain budget configuration %q: %s", src.Spec.BudgetID, toErrMsg(err))
	}

	budget.NotificationsRule = &budgets.NotificationsRule{}
	budgetUpdatedRequest := budgets.UpdateBudgetRequest{
		Budget: budget,
	}

	_, err = cli.UpdateBudget(ctx, &budgetUpdatedRequest)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Billing API: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Budget does not exist: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingBudgetNotification(src.Spec.BudgetID, err))
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Cannot delete budget notification %q: %s", src.Spec.BudgetID, toErrMsg(err))
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted Billing budget notification "+budget.DisplayName)
	return nil
}

// failCreatingBillingNotification returns a reconciler event which indicates
// that a billing budget notification could not be retrieved or created.
func failCreatingBudgetNotification(budgetID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating Budget Notification %q: %s", budgetID, toErrMsg(origErr))
}

// Generates the resource name for the topic used by a CloudBillingSource.
func generateTopicResourceName(s *v1alpha1.GoogleCloudBillingSource, topicID string) string {
	return fmt.Sprintf("projects/%s/topics/%s", *s.Spec.PubSub.Project, topicID)
}

// Generates the budgetId for the budget request used by a CloudBillingSource.
func generateBudgetID(s *v1alpha1.GoogleCloudBillingSource) string {
	return fmt.Sprintf("billingAccounts/%s/budgets/%s", s.Spec.BillingAccountID, s.Spec.BudgetID)
}
