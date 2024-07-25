package jsonengine

import jsonvalue "github.com/Andrew-M-C/go.jsonvalue"

const (
	ErrNotFound          = jsonvalue.ErrNotFound
	ErrTypeNotMatch      = jsonvalue.ErrTypeNotMatch
	ErrImportTargetValue = jsonvalue.Error("import target value error")
	ErrIllegalOperator   = jsonvalue.Error("illegal operator")
)
