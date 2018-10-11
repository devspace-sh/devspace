package configutil

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/covexo/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v2"
)

//SaveConfig writes the data of a config to its yaml file
func SaveConfig() error {
	workdir, _ := os.Getwd()
	configMapRaw, overwriteMapRaw, err := getConfigAndOverwriteMaps(config, configRaw, overwriteConfig, overwriteConfigRaw)

	configMap, _ := configMapRaw.(map[interface{}]interface{})
	overwriteMap, _ := overwriteMapRaw.(map[interface{}]interface{})

	if err != nil {
		return err
	}

	configYaml, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(workdir + ConfigPath)
	os.MkdirAll(configDir, os.ModePerm)

	// Check if .gitignore exists
	_, err = os.Stat(filepath.Join(configDir, ".gitignore"))
	if os.IsNotExist(err) {
		fsutil.WriteToFile([]byte(configGitignore), filepath.Join(configDir, ".gitignore"))
	}

	writeErr := ioutil.WriteFile(workdir+ConfigPath, configYaml, os.ModePerm)
	if writeErr != nil {
		return writeErr
	}

	if overwriteMap != nil {
		overwriteConfigYaml, err := yaml.Marshal(overwriteMap)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(workdir+OverwriteConfigPath, overwriteConfigYaml, os.ModePerm)
	}

	return nil
}

// TODO: Think about removing configRaw & overwriteConfigRaw
func getConfigAndOverwriteMaps(config interface{}, configRaw interface{}, overwriteConfig interface{}, overwriteConfigRaw interface{}) (interface{}, interface{}, error) {
	object, isObjectNil := getPointerValue(config)

	objectType := reflect.TypeOf(object)
	objectKind := objectType.Kind()
	overwriteObject, isOverwriteObjectNil := getPointerValue(overwriteConfig)
	overwriteObjectKind := reflect.TypeOf(overwriteObject).Kind()

	if objectKind != overwriteObjectKind && !isObjectNil && !isOverwriteObjectNil {
		return nil, nil, errors.New("config (type: " + objectKind.String() + ") and overwriteConfig (type: " + overwriteObjectKind.String() + ") must be instances of the same type.")
	}
	objectValueRef := reflect.ValueOf(object)
	objectValue := objectValueRef.Interface()

	overwriteValueRef := reflect.ValueOf(overwriteObject)
	overwriteValue := overwriteValueRef.Interface()

	objectRaw, isObjectRawNil := getPointerValue(configRaw)
	objectRawValueRef := reflect.ValueOf(objectRaw)
	objectRawValue := objectRawValueRef.Interface()

	overwriteObjectRaw, _ := getPointerValue(overwriteConfigRaw)
	overwriteRawValueRef := reflect.ValueOf(overwriteObjectRaw)
	overwriteRawValue := overwriteRawValueRef.Interface()

	switch objectKind {
	case reflect.Slice:
		returnSlice := []interface{}{}
		returnOverwriteSlice := []interface{}{}
		var err error

	OUTER:
		for i := 0; i < objectValueRef.Len(); i++ {
			valRef := objectValueRef.Index(i)
			val := valRef.Interface()

			if valRef.Type().Kind() == reflect.Ptr {
				for ii := 0; ii < overwriteValueRef.Len(); ii++ {
					if val == overwriteValueRef.Index(ii).Interface() {
						continue OUTER
					}
				}
			}

			if val != nil {
				//to remove nil values
				_, val, err = getConfigAndOverwriteMaps(val, val, val, val)

				if err != nil {
					return nil, nil, err
				}
				returnSlice = append(returnSlice, val)
			}
		}

		for i := 0; i < overwriteValueRef.Len(); i++ {
			val := overwriteValueRef.Index(i).Interface()

			if val != nil {
				//to remove nil values
				_, val, err = getConfigAndOverwriteMaps(val, val, val, val)

				if err != nil {
					return nil, nil, err
				}

				returnOverwriteSlice = append(returnOverwriteSlice, val)
			}
		}

		if len(returnSlice) > 0 && len(returnOverwriteSlice) > 0 {
			return returnSlice, returnOverwriteSlice, nil
		} else if len(returnSlice) > 0 {
			return returnSlice, nil, nil
		} else if len(returnOverwriteSlice) > 0 {
			return nil, returnOverwriteSlice, nil
		}
		return nil, nil, nil
	case reflect.Map:
		returnMap := map[interface{}]interface{}{}
		returnOverwriteMap := map[interface{}]interface{}{}
		genericPointerType := reflect.TypeOf(&returnMap)

		for _, keyRef := range objectValueRef.MapKeys() {
			key := keyRef.Interface()
			val := getMapValue(objectValue, key, genericPointerType)
			yamlKey := getYamlKey(key.(string))
			valType := reflect.TypeOf(val)
			overwriteVal, _ := getPointerValue(getMapValue(overwriteValue, key, valType))
			valRaw, _ := getPointerValue(getMapValue(objectRawValue, key, valType))
			overwriteValRaw, _ := getPointerValue(getMapValue(overwriteRawValue, key, valType))

			var err error

			val, overwriteVal, err = getConfigAndOverwriteMaps(
				val,
				valRaw,
				overwriteVal,
				overwriteValRaw,
			)

			if err != nil {
				return nil, nil, err
			}

			valRef := reflect.ValueOf(val)

			if !isZero(valRef) {
				returnMap[yamlKey] = val
			}

			overwriteValRef := reflect.ValueOf(overwriteVal)

			if !isZero(overwriteValRef) {
				returnOverwriteMap[yamlKey] = overwriteVal
			}
		}

		if len(returnMap) > 0 && len(returnOverwriteMap) > 0 {
			return returnMap, returnOverwriteMap, nil
		} else if len(returnMap) > 0 {
			return returnMap, nil, nil
		} else if len(returnOverwriteMap) > 0 {
			return nil, returnOverwriteMap, nil
		}
		return nil, nil, nil
	case reflect.Struct:
		returnMap := map[interface{}]interface{}{}
		returnOverwriteMap := map[interface{}]interface{}{}

		for i := 0; i < objectValueRef.NumField(); i++ {
			field := objectValueRef.Field(i)
			typeField := objectValueRef.Type().Field(i)
			yamlKey := getYamlKey(typeField.Name)

			if field.CanInterface() {
				fieldType := typeField.Type.Elem()
				fieldRaw := objectRawValueRef.Field(i).Elem()
				overwriteField := overwriteValueRef.Field(i).Elem()
				overwriteFieldRaw := overwriteRawValueRef.Field(i).Elem()

				fieldValue := field.Interface()
				fieldRawValue := reflect.New(fieldType).Interface()
				overwriteFieldValue := reflect.New(fieldType).Interface()
				overwriteRawFieldValue := reflect.New(fieldType).Interface()

				saveOverwriteField := false
				isFieldNil := true
				isFieldRawNil := true

				if !isZero(field) {
					isFieldNil = false
				}

				if !isZero(fieldRaw) {
					fieldRawValue = fieldRaw.Interface()
					isFieldRawNil = false
				}

				if !isZero(overwriteField) {
					overwriteFieldValue = overwriteField.Interface()
					saveOverwriteField = true
				}

				if !isZero(overwriteFieldRaw) {
					overwriteRawFieldValue = overwriteFieldRaw.Interface()
				}
				var fieldValueClean, overwriteFieldValueClean interface{}
				var err error

				fieldKind := fieldType.Kind()

				if fieldKind == reflect.Map || fieldKind == reflect.Slice || fieldKind == reflect.Struct {
					fieldValueClean, overwriteFieldValueClean, err = getConfigAndOverwriteMaps(
						fieldValue,
						fieldRawValue,
						overwriteFieldValue,
						overwriteRawFieldValue,
					)

					if err != nil {
						return nil, nil, err
					}
				} else {
					saveField := ((!isFieldNil && !saveOverwriteField) || !isFieldRawNil)

					if saveField {
						fieldValueClean = fieldValue
					}

					if saveOverwriteField {
						overwriteFieldValueClean = overwriteFieldValue
					}
				}

				if fieldValueClean != nil {
					returnMap[yamlKey] = fieldValueClean
				}

				if overwriteFieldValueClean != nil {
					returnOverwriteMap[yamlKey] = overwriteFieldValueClean
				}
			}
		}

		if len(returnMap) > 0 && len(returnOverwriteMap) > 0 {
			return returnMap, returnOverwriteMap, nil
		} else if len(returnMap) > 0 {
			return returnMap, nil, nil
		} else if len(returnOverwriteMap) > 0 {
			return nil, returnOverwriteMap, nil
		}
		return nil, nil, nil
	default:
		saveOverwriteValue := !isOverwriteObjectNil
		saveValue := ((!isObjectNil && !saveOverwriteValue) || !isObjectRawNil)

		//TODO: Determine overwritten values and set objectValue accordingly

		if saveValue && saveOverwriteValue {
			return objectValue, overwriteValue, nil
		} else if saveOverwriteValue {
			return nil, overwriteValue, nil
		} else if saveValue {
			return objectValue, nil, nil
		}
		return nil, nil, nil
	}
}

func getMapValue(valueMap interface{}, key interface{}, refType reflect.Type) interface{} {
	valueMapValue, _ := getPointerValue(valueMap)
	mapRef := reflect.ValueOf(valueMapValue)
	keyRef := reflect.ValueOf(key)
	mapValue := mapRef.MapIndex(keyRef)

	if isZero(mapValue) {
		mapValue = reflect.New(refType)
	}
	return mapValue.Interface()
}

//isZero is a reflect function from: https://github.com/golang/go/issues/7501
func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func getYamlKey(key string) string {
	re := ""

	for i := 0; i < len(key); i++ {
		letter := key[i : i+1]
		lowerLetter := strings.ToLower(letter)

		if i == 0 || (letter != lowerLetter) {
			re = re + lowerLetter
		} else {
			re = re + key[i:]
			break
		}
	}
	return re
}

func getPointerValue(object interface{}) (interface{}, bool) {
	if object != nil {
		objectType := reflect.TypeOf(object)
		objectKind := objectType.Kind()

		if objectKind == reflect.Ptr {
			objectValueRef := reflect.ValueOf(object)

			if objectValueRef.IsNil() {
				pointerValueType := objectValueRef.Type().Elem()
				newInstance, _ := getPointerValue(reflect.New(pointerValueType).Interface())

				return newInstance, true
			}
			value, _ := getPointerValue(objectValueRef.Elem().Interface())

			return value, false
		}
	}
	return object, false
}
