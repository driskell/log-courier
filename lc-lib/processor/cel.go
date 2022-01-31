/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package processor

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celext "github.com/google/cel-go/ext"

	"github.com/driskell/log-courier/lc-lib/processor/ext"
)

var celEnv *cel.Env
var celErr error

// cachedCelEnv returns a globally cached cel.Env for use in checking and parsing
func cachedCelEnv() (*cel.Env, error) {
	if celEnv != nil || celErr != nil {
		return celEnv, celErr
	}

	return cel.NewEnv(
		cel.Declarations(
			decls.NewVar("event", decls.NewMapType(decls.String, decls.Any)),
		),
		celext.Strings(),
		celext.Encoders(),
		ext.JsonEncoder(),
	)
}

// ParseExpression parses an expression using cel-go and returns the evaluatable program
func ParseExpression(expression string) (cel.Program, error) {
	env, err := cachedCelEnv()
	if err != nil {
		return nil, err
	}

	// Parse using the environment
	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	// Likely this does nothing at the moment as we don't prepare any declarations
	// But keep it here in case we improve the environment
	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	return env.Program(checked)
}
