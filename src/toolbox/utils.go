package toolbox

import (
	"reflect"
)

func Flatten(args ...interface{}) (flattenArgs []interface{}) {
	for _, arg := range args {
		if IsListType(arg) {
			valVal := reflect.ValueOf(arg)
			if valVal.Len() == 0 {
				if flattenArgs == nil {
					flattenArgs = []interface{}{}
				}
			} else {
				for i := 0; i < valVal.Len(); i++ {
					flattenArgs = append(flattenArgs, valVal.Index(i).Interface())
				}
			}
		} else {
			flattenArgs = append(flattenArgs, arg)
		}
	}

	return
}

func IsListType(val interface{}) bool {
	valVal := reflect.ValueOf(val)
	return valVal.Kind() == reflect.Array || valVal.Kind() == reflect.Slice
}
