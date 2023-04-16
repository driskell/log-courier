package ast

import (
	"reflect"

	"github.com/google/cel-go/common/types/ref"
)

type Argument interface {
	Name() string
	IsRequired() bool
	IsExprDisallowed() bool
	Resolve(ref.Val) (any, error)
}

type ArgumentOption int

const (
	ArgumentOptional ArgumentOption = iota
	ArgumentRequired
	ArgumentExprDisallowed
)

type argument struct {
	name  string
	flags ArgumentOption
}

func (a *argument) Name() string {
	return a.name
}

func (a *argument) IsRequired() bool {
	return a.flags&ArgumentRequired != 0
}

func (a *argument) IsExprDisallowed() bool {
	return a.flags&ArgumentExprDisallowed != 0
}

type argumentString struct {
	argument
}

var _ Argument = &argumentString{}
var stringType = reflect.TypeOf("")

func NewArgumentString(name string, flags ArgumentOption) Argument {
	return &argumentString{argument: argument{name, flags}}
}

func (a *argumentString) Resolve(value ref.Val) (any, error) {
	nativeValue, err := value.ConvertToNative(stringType)
	if err != nil {
		return nil, err
	}
	return nativeValue, nil
}

type argumentBool struct {
	argument
}

var _ Argument = &argumentBool{}
var boolType = reflect.TypeOf(true)

func NewArgumentBool(name string, flags ArgumentOption) Argument {
	return &argumentBool{argument: argument{name, flags}}
}

func (a *argumentBool) Resolve(value ref.Val) (any, error) {
	nativeValue, err := value.ConvertToNative(boolType)
	if err != nil {
		return nil, err
	}
	return nativeValue, nil
}

type argumentListString struct {
	argument
}

var _ Argument = &argumentListString{}
var listStringType = reflect.TypeOf([]string{})

func NewArgumentListString(name string, flags ArgumentOption) Argument {
	return &argumentListString{argument: argument{name, flags}}
}

func (a *argumentListString) Resolve(value ref.Val) (any, error) {
	nativeValue, err := value.ConvertToNative(listStringType)
	if err != nil {
		return nil, err
	}
	return nativeValue, nil
}
