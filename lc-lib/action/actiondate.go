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

package action

import (
	"fmt"
	"strconv"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

type dateActionNode struct {
}

var _ ast.ProcessArgumentsNode = &dateActionNode{}

func newDateActionNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &dateActionNode{}, nil
}

func (n *dateActionNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentListString("formats", ast.ArgumentRequired),
		ast.NewArgumentBool("remove", ast.ArgumentOptional),
	}
}

func (n *dateActionNode) Init([]any) error {
	return nil
}

func (n *dateActionNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	field := arguments[0].(string)
	formats := arguments[1].([]string)
	remove := false
	if arguments[2] != nil {
		remove = arguments[2].(bool)
	}

	entry, err := subject.Resolve(field, nil)
	if err != nil {
		subject.AddError("date", fmt.Sprintf("Field '%s' could not be resolved: %s", field, err))
		return subject
	}

	var (
		value string
		ok    bool
	)
	value, ok = entry.(string)
	if !ok {
		subject.AddError("date", fmt.Sprintf("Field '%s' is not present or not a string", field))
		return subject
	}

	for _, layout := range formats {
		var (
			result time.Time
			err    error
		)

		switch layout {
		case "UNIX":
			unix, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			result = time.Unix(int64(unix), int64((unix - float64(int64(unix))*1000)))
		default:
			result, err = time.Parse(layout, value)
			if err != nil {
				continue
			}
		}

		// If year 0, we only parsed month/day etc.
		// We do not support parsing of dates without the current date
		// For that, we would likely have a flag to say only time parsed, so we can explicitly set the date
		if result.Year() == 0 {
			result = time.Date(time.Now().Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second(), result.Nanosecond(), result.Location())
		}

		subject.MustResolve("@timestamp", result)
		if remove {
			_, err := subject.Resolve(field, event.ResolveParamUnset)
			if err != nil {
				subject.AddError("date", fmt.Sprintf("Failed to remove field '%s': %s", field, err))
			}
		}
		return subject
	}

	subject.AddError("date", fmt.Sprintf("Field '%s' could not be parsed with any of the given formats", field))
	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("date", newDateActionNode)
}
