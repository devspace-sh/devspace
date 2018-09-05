package configutil

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	"github.com/covexo/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v2"
)

//SaveConfig writes the data of a config to its yaml file
func SaveConfig() error {
	configExists, _ := ConfigExists()

	// just in case someone has set a pointer to one of the structs to nil, merge empty an empty config object into all configs
	baseConfig := makeConfig()
	merge(config, baseConfig, unsafe.Pointer(&config), unsafe.Pointer(baseConfig))
	merge(configRaw, baseConfig, unsafe.Pointer(&configRaw), unsafe.Pointer(baseConfig))
	merge(overwriteConfig, baseConfig, unsafe.Pointer(&overwriteConfig), unsafe.Pointer(baseConfig))
	merge(overwriteConfigRaw, baseConfig, unsafe.Pointer(&overwriteConfigRaw), unsafe.Pointer(baseConfig))

	configMapRaw, overwriteMapRaw, configErr := getConfigAndOverwriteMaps(config, configRaw, overwriteConfig, overwriteConfigRaw)

	configMap, _ := configMapRaw.(map[interface{}]interface{})
	overwriteMap, _ := overwriteMapRaw.(map[interface{}]interface{})

	if configErr != nil {
		return configErr
	}

	if config.Cluster.UseKubeConfig != nil && *config.Cluster.UseKubeConfig {
		clusterConfig := map[string]bool{
			"useKubeConfig": true,
		}
		_, configHasCluster := configMap["cluster"]

		if configHasCluster {
			configMap["cluster"] = clusterConfig
		}
		_, overwriteConfigHasCluster := overwriteMap["cluster"]

		if overwriteConfigHasCluster {
			overwriteMap["cluster"] = clusterConfig
		}
	}
	configYaml, yamlErr := yaml.Marshal(configMap)

	if yamlErr != nil {
		return yamlErr
	}
	configDir := filepath.Dir(workdir + configPath)

	os.MkdirAll(configDir, os.ModePerm)

	if !configExists {
		fsutil.WriteToFile([]byte(configGitignore), filepath.Join(configDir, ".gitignore"))
	}
	writeErr := ioutil.WriteFile(workdir+configPath, configYaml, os.ModePerm)

	if writeErr != nil {
		return writeErr
	}

	if overwriteMap != nil {
		overwriteConfigYaml, yamlErr := yaml.Marshal(overwriteMap)

		if yamlErr != nil {
			return yamlErr
		}
		return ioutil.WriteFile(workdir+overwriteConfigPath, overwriteConfigYaml, os.ModePerm)
	}
	return nil
}

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

		for i := 0; i < objectValueRef.Len(); i++ {
			val := objectValueRef.Index(i)
			//TODO: remove overwriteValues and write them into returnOverwriteSlice
			returnSlice = append(returnSlice, val)
		}

		if len(returnSlice) > 0 && len(returnOverwriteSlice) > 0 {
			return returnSlice, returnOverwriteSlice, nil
		} else if len(returnSlice) > 0 {
			return returnSlice, nil, nil
		} else if len(returnOverwriteSlice) > 0 {
			return nil, returnOverwriteSlice, nil
		} else {
			return nil, nil, nil
		}
	case reflect.Map:
		valueMap := objectValue.(map[interface{}]interface{})
		returnMap := map[interface{}]interface{}{}
		returnOverwriteMap := map[interface{}]interface{}{}

		for key, val := range valueMap {
			key = getYamlKey(key.(string))
			valType := reflect.TypeOf(val)

			cleanVal, cleanOverwriteVal, err := getConfigAndOverwriteMaps(
				val,
				getValueOrZero(objectRawValue, key, valType),
				getValueOrZero(overwriteValue, key, valType),
				getValueOrZero(overwriteRawValue, key, valType),
			)

			if err != nil {
				return nil, nil, err
			}

			if cleanVal != nil {
				returnMap[key] = cleanVal
			}

			if cleanOverwriteVal != nil {
				returnOverwriteMap[key] = cleanOverwriteVal
			}
		}

		if len(returnMap) > 0 && len(returnOverwriteMap) > 0 {
			return returnMap, returnOverwriteMap, nil
		} else if len(returnMap) > 0 {
			return returnMap, nil, nil
		} else if len(returnOverwriteMap) > 0 {
			return nil, returnOverwriteMap, nil
		} else {
			return nil, nil, nil
		}
	case reflect.Struct:
		returnMap := map[interface{}]interface{}{}
		returnOverwriteMap := map[interface{}]interface{}{}

		for i := 0; i < objectValueRef.NumField(); i++ {
			fieldName := getYamlKey(objectValueRef.Type().Field(i).Name)

			fieldValue := objectValueRef.Field(i).Interface()
			fieldRawValue := objectRawValueRef.Field(i).Interface()
			overwriteFieldValue := overwriteValueRef.Field(i).Interface()
			overwriteRawFieldValue := overwriteRawValueRef.Field(i).Interface()

			fieldValueClean, overwriteFieldValueClean, err := getConfigAndOverwriteMaps(
				fieldValue,
				fieldRawValue,
				overwriteFieldValue,
				overwriteRawFieldValue,
			)

			if err != nil {
				return nil, nil, err
			}

			if fieldValueClean != nil {
				returnMap[fieldName] = fieldValueClean
			}

			if overwriteFieldValueClean != nil {
				returnOverwriteMap[fieldName] = overwriteFieldValueClean
			}
		}

		if len(returnMap) > 0 && len(returnOverwriteMap) > 0 {
			return returnMap, returnOverwriteMap, nil
		} else if len(returnMap) > 0 {
			return returnMap, nil, nil
		} else if len(returnOverwriteMap) > 0 {
			return nil, returnOverwriteMap, nil
		} else {
			return nil, nil, nil
		}
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
		} else {
			return nil, nil, nil
		}
	}
}

func getValueOrZero(valueMap interface{}, key interface{}, refType reflect.Type) interface{} {
	valueMapValidated := valueMap.(map[interface{}]interface{})
	value, valueExists := valueMapValidated[key]

	if !valueExists {
		value = reflect.Zero(refType).Interface()
	}
	return value
}

func getYamlKey(key string) string {
	return strings.ToLower(key[0:1]) + key[1:]
}

func getPointerValue(object interface{}) (interface{}, bool) {
	if object != nil {
		objectType := reflect.TypeOf(object)
		objectKind := objectType.Kind()

		if objectKind == reflect.Ptr {
			objectValueRef := reflect.ValueOf(object)

			if objectValueRef.IsNil() {
				zeroValue := reflect.Zero(objectValueRef.Type()).Interface()

				return zeroValue, true
			} else {
				return objectValueRef.Elem().Interface(), false
			}
		}
	}
	return object, false
}
