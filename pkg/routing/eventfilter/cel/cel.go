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

package cel

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
)

var errVarType = errors.New("variable definition doesn't match expected format: \"$json_path.(type)\"")

// CompileExpression accepts the expression string from the Filter spec,
// parses variables and their types, compiles expression into CEL Program
func CompileExpression(expression string) (ConditionalFilter, error) {
	expr, vars, err := parseExpressionString(expression)
	if err != nil {
		return ConditionalFilter{}, err
	}
	prog, err := newCEL(expr, vars)
	if err != nil {
		return ConditionalFilter{}, err
	}
	return ConditionalFilter{
		Expression: &prog,
		Variables:  vars,
	}, nil
}

// parseExpressionString breaks inline expression string into Google CEL expression
// and a set of variable definitions, e.g.:
// '$foo.(string) == "bar"' becomes
// expr: foo == "bar", vars: ["foo": string]
func parseExpressionString(expression string) (string, []Variable, error) {
	var vars []Variable
	var cleanExpr string

	for i := 0; i < len(expression); i++ {
		expression = expression[i:]

		variable := strings.Index(expression, "$")
		if variable == -1 {
			cleanExpr += expression
			break
		}

		// start looking for the variable type after the variable name
		typ := strings.Index(expression[variable:], ".(")
		if typ == -1 {
			return "", []Variable{}, errVarType
		}
		typ += variable

		end := strings.Index(expression[variable:], ")")
		if end == -1 {
			return "", []Variable{}, errVarType
		}
		end += variable

		i = end
		cleanExpr += expression[:variable]

		safeCELName := strings.ReplaceAll(expression[variable+1:typ], ".", "_")
		// integer as the variable name first symbol causes issue with matching
		// var types. String prefix ensures that we don't have first integer symbol.
		safeCELName = "var_" + safeCELName

		if variable+1 > typ || typ+2 > end {
			return "", []Variable{}, errVarType
		}

		vars = append(vars, Variable{
			Name: safeCELName,
			Path: expression[variable+1 : typ],
			Type: expression[typ+2 : end],
		})
		cleanExpr += safeCELName
	}
	return cleanExpr, vars, nil
}

// newCEL creates CEL env, sets its variables, compiles expression string
// and validates expression result type
func newCEL(expr string, vars []Variable) (cel.Program, error) {
	declVars := []*exprpb.Decl{}
	for _, variable := range vars {
		primitiveType := exprpb.Type_PrimitiveType(exprpb.Type_PrimitiveType_value[strings.ToUpper(variable.Type)])
		declVars = append(declVars, decls.NewVar(variable.Name, decls.NewPrimitiveType(primitiveType)))
	}

	env, err := cel.NewEnv(
		cel.Declarations(declVars...),
	)
	if err != nil {
		return nil, err
	}

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	if !proto.Equal(ast.ResultType(), decls.Bool) {
		return nil, fmt.Errorf("expression %q must return bool type, got %s", expr, ast.ResultType().String())
	}

	return env.Program(ast)
}
