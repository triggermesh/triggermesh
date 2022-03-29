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

// Package iam contains helpers to interact with IAM objects.
package iam

import "github.com/google/uuid"

const latestPolicyLanguageVersion = "2012-10-17"

// Policy mirrors the structure of an IAM Policy for easy marshaling and
// unmarshaling to/from JSON.
// See https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html#access_policies-json
type Policy struct {
	Version   string            `json:"Version"`
	ID        string            `json:"Id,omitempty"`
	Statement []PolicyStatement `json:"Statement,omitempty"`
}

// PolicyStatement is a Statement element in a Policy.
type PolicyStatement struct {
	Sid       string                   `json:"Sid,omitempty"`
	Effect    PolicyStatementEffect    `json:"Effect"`
	Principal PolicyStatementPrincipal `json:"Principal,omitempty"`
	Action    []string                 `json:"Action"`
	Resource  []string                 `json:"Resource"`
	Condition PolicyStatementCondition `json:"Condition,omitempty"`
}

// PolicyStatementEffect represents the Effect element of a Statement.
type PolicyStatementEffect string

// List of acceptable PolicyStatementEffect values.
const (
	EffectAllow PolicyStatementEffect = "Allow"
)

// PolicyStatementPrincipal is the Principal element of a Statement.
// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_principal.html
type PolicyStatementPrincipal struct {
	Service []string `json:"Service"`
}

// PolicyStatementCondition is the Condition element of a Statement.
// See https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html
type PolicyStatementCondition struct {
	ArnEquals    map[string]string `json:"ArnEquals,omitempty"`
	StringEquals map[string]string `json:"StringEquals,omitempty"`
}

// NewPolicy returns a new Policy with the given Statements applied to it.
func NewPolicy(stmts ...PolicyStatement) Policy {
	return Policy{
		Version:   latestPolicyLanguageVersion,
		ID:        uuid.New().String(),
		Statement: stmts,
	}
}

// NewPolicyStatement returns a new PolicyStatement with the given options
// applied to it.
func NewPolicyStatement(effect PolicyStatementEffect, opts ...PolicyStatementOpt) PolicyStatement {
	p := PolicyStatement{
		Sid:    uuid.New().String(),
		Effect: effect,
	}

	for _, opt := range opts {
		opt(&p)
	}

	return p
}

// PolicyStatementOpt is a functional option for a PolicyStatement.
type PolicyStatementOpt func(*PolicyStatement)

// PrincipalService adds a "Service" to the Principal.
func PrincipalService(service string) PolicyStatementOpt {
	return func(s *PolicyStatement) {
		ps := &s.Principal.Service
		if *ps == nil {
			valPs := make([]string, 0, 1)
			*ps = valPs
		}
		*ps = append(*ps, service)
	}
}

// Action adds an Action.
func Action(action string) PolicyStatementOpt {
	return func(s *PolicyStatement) {
		a := &s.Action
		if *a == nil {
			*a = make([]string, 0, 1)
		}
		*a = append(*a, action)
	}
}

// Resource adds a Resource.
func Resource(resource string) PolicyStatementOpt {
	return func(s *PolicyStatement) {
		r := &s.Resource
		if *r == nil {
			*r = make([]string, 0, 1)
		}
		*r = append(*r, resource)
	}
}

// ConditionArnEquals sets a Condition of type "ArnEquals".
func ConditionArnEquals(key, val string) PolicyStatementOpt {
	return func(s *PolicyStatement) {
		aec := &s.Condition.ArnEquals
		if *aec == nil {
			valAec := make(map[string]string, 1)
			*aec = valAec
		}
		(*aec)[key] = val
	}
}

// ConditionStringEquals sets a Condition of type "StringEquals".
func ConditionStringEquals(key, val string) PolicyStatementOpt {
	return func(s *PolicyStatement) {
		sec := &s.Condition.StringEquals
		if *sec == nil {
			valSec := make(map[string]string, 1)
			*sec = valSec
		}
		(*sec)[key] = val
	}
}
