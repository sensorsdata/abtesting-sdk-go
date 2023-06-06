package utils

import (
	"reflect"
)

func GetValue(value reflect.Value, defaultType reflect.Type) interface{} {
	if "string" == defaultType.String() {
		return value.String()
	} else if "int" == defaultType.String() {
		return value.Int()
	} else if "int8" == defaultType.String() {
		return value.Int()
	} else if "int16" == defaultType.String() {
		return value.Int()
	} else if "int32" == defaultType.String() {
		return value.Int()
	} else if "int64" == defaultType.String() {
		return value.Int()
	} else if "bool" == defaultType.String() {
		return value.Bool()
	}
	return nil
}
