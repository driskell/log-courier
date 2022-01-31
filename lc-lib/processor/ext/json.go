/*
 * Copyright 2022 Jason Woods and contributors
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

package ext

import (
	"encoding/json"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"

	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// JsonEncoder returns a cel-go extension for JSON encoding and decoding
func JsonEncoder() cel.EnvOption {
	return cel.Lib(jsonLib{})
}

var interfaceType interface{}

type jsonLib struct{}

func (jsonLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Declarations(
			decls.NewFunction("json.decode",
				decls.NewOverload("json_decode",
					[]*exprpb.Type{decls.Bytes},
					decls.Any)),
			decls.NewFunction("json.encode",
				decls.NewOverload("json_encode",
					[]*exprpb.Type{decls.Any},
					decls.Bytes)),
		),
	}
}

func (jsonLib) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{
		cel.Functions(
			&functions.Overload{
				Operator: "json.decode",
				Unary:    jsonDecode,
			},
			&functions.Overload{
				Operator: "json_decode",
				Unary:    jsonDecode,
			},
			&functions.Overload{
				Operator: "json.encode",
				Unary:    jsonEncode,
			},
			&functions.Overload{
				Operator: "json_encode",
				Unary:    jsonEncode,
			},
		),
	}
}

func jsonDecode(val ref.Val) ref.Val {
	vVal, ok := val.(types.Bytes)
	if !ok {
		return types.MaybeNoSuchOverloadErr(val)
	}
	var v interface{}
	err := json.Unmarshal([]byte(vVal), &v)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.DefaultTypeAdapter.NativeToValue(v)
}

func jsonEncode(val ref.Val) ref.Val {
	v, err := val.ConvertToNative(reflect.ValueOf(&interfaceType).Type().Elem())
	if err != nil {
		return types.NewErr(err.Error())
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.Bytes(bytes)
}
