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

package eventbridge

import (
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// deleteAllRules deletes all rules from the given event bus.
func deleteAllRules(ebClient eventbridgeiface.EventBridgeAPI, eventBusName string) {
	in := &eventbridge.ListRulesInput{
		EventBusName: &eventBusName,
	}

	rules, err := ebClient.ListRules(in)
	if err != nil {
		framework.FailfWithOffset(3, "Failed to list rules for event bus %q: %s", *in.EventBusName, err)
	}

	for _, rule := range rules.Rules {
		deleteRule(ebClient, *rule.Name, eventBusName)
	}
}

// deleteRule deletes a rule and all its targets.
func deleteRule(ebClient eventbridgeiface.EventBridgeAPI, name, eventBusName string) {
	removeAllTargets(ebClient, name, eventBusName)

	rule := &eventbridge.DeleteRuleInput{
		Name:         &name,
		EventBusName: &eventBusName,
	}

	if _, err := ebClient.DeleteRule(rule); err != nil {
		framework.FailfWithOffset(4, "Failed to delete rule %q: %s", *rule.Name, err)
	}
}

// removeAllTargets removes all targets from the given rule.
func removeAllTargets(ebClient eventbridgeiface.EventBridgeAPI, ruleName, eventBusName string) {
	in := &eventbridge.ListTargetsByRuleInput{
		Rule:         &ruleName,
		EventBusName: &eventBusName,
	}

	trgts, err := ebClient.ListTargetsByRule(in)
	if err != nil {
		framework.FailfWithOffset(5, "Failed to list targets for rule %q: %s", *in.Rule, err)
	}

	trgtIDs := make([]*string, len(trgts.Targets))
	for i, trgt := range trgts.Targets {
		trgtIDs[i] = trgt.Id
	}

	trgtsToRemove := &eventbridge.RemoveTargetsInput{
		Rule:         &ruleName,
		EventBusName: &eventBusName,
		Ids:          trgtIDs,
	}

	if _, err := ebClient.RemoveTargets(trgtsToRemove); err != nil {
		framework.FailfWithOffset(5, "Failed to remove targets from rule %q: %s", *trgtsToRemove.Rule, err)
	}
}
