package paramutil

import (
	"reflect"
)

func SetDefaults(params interface{}, defaultParams interface{}) {
	defaultParamValues := reflect.ValueOf(defaultParams).Elem()
	paramValues := reflect.ValueOf(params).Elem()
	typeOfT := defaultParamValues.Type()

	for i := 0; i < defaultParamValues.NumField(); i++ {
		defaultParamValue := defaultParamValues.Field(i).Interface()
		paramName := typeOfT.Field(i).Name
		paramRefValue := paramValues.FieldByName(paramName)
		paramValue := paramRefValue.Interface()

		if reflect.TypeOf(defaultParamValue).Kind() == reflect.String && len(paramValue.(string)) == 0 {
			paramRefValue.SetString(defaultParamValue.(string))
		}
	}
}
