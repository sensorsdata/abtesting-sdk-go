package utils

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	KEY_MAX   = 100
	VALUE_MAX = 8192

	NAME_PATTERN_BAD = "^(^distinct_id$|^original_id$|^time$|^properties$|^id$|^first_id$|^second_id$|^users$|^events$|^event$|^user_id$|^date$|^datetime$)$"
	NAME_PATTERN_OK  = "^[a-zA-Z_$][a-zA-Z\\d_$]{0,99}$"
)

var patternBad, patternOk *regexp.Regexp

func init() {
	patternBad, _ = regexp.Compile(NAME_PATTERN_BAD)
	patternOk, _ = regexp.Compile(NAME_PATTERN_OK)
}

func CheckProperty(properties map[string]interface{}) error {
	//check properties
	if properties != nil {
		for k, v := range properties {
			//check key
			err := isKeyValid(k, v)
			if err != nil {
				return err
			}
			//check value
			err = isValueValid(properties, k, v)
			if err != nil {
				return err
			}
			str, ok := v.([]string)
			if ok {
				data, _ := json.Marshal(str)
				properties[k] = string(data)
			}
		}
	}
	return nil
}

func CheckCustomIds(customIds map[string]string) error {
	//check properties
	if customIds != nil {
		for k, v := range customIds {
			//check key
			if strings.HasPrefix(k, "$") {
				return errors.New("'$' 开头的不合法的 ID， key = " + k)
			}
			err := isKeyValid(k, v)
			if err != nil {
				return err
			}
			//check value
			if v == "" {
				return errors.New("ID 属性值为空的不合法属性，key = " + k)
			} else if len(v) > 1024 {
				return errors.New("ID 属性值长度超过 1024 的不合法属性，key = " + k)
			}
		}
	}
	return nil
}

// 检查是否符合命名规范，符合变量命名和不是预置关键字
func checkPattern(name []byte) bool {
	return !patternBad.Match(name) && patternOk.Match(name)
}

// 检查 key 是否合法
func isKeyValid(key string, value interface{}) error {
	if len(key) > KEY_MAX {
		return errors.New("the max length of property key is 100," + "key = " + key)
	}

	if len(key) == 0 {
		return errors.New("The key is empty or null," + "key = " + key + ", value = " + value.(string))
	}
	isMatch := checkPattern([]byte(key))
	if !isMatch {
		return errors.New("property key must be a valid variable name," + "key = " + key)
	}
	return nil
}

// 检查 value 是否合法
func isValueValid(properties map[string]interface{}, key string, value interface{}) error {
	switch v := value.(type) {
	case int:
	case bool:
	case float64:
	case string:
		if len(v) > VALUE_MAX {
			return errors.New("the max length of property value is 8192," + "value = " + v)
		}
	case []string: //value in properties list MUST be string
	case time.Time: //only support time.Time
		properties[key] = v.Format("2006-01-02 15:04:05.999")
	default:
		return errors.New("property value must be a string/int/float64/bool/time.Time/[]string," + "key = " + key)
	}
	return nil
}

func MapToJson(param map[string]string) string {
	dataType, _ := json.Marshal(param)
	dataString := string(dataType)
	return dataString
}

func JsonToMap(str string) map[string]interface{} {

	var tempMap map[string]interface{}
	err := json.Unmarshal([]byte(str), &tempMap)

	if err != nil {
		return nil
	}

	return tempMap
}

func MapEquals(x, y map[string]int) bool {
	if len(x) != len(y) {
		return false
	}
	for k, xv := range x {
		if yv, ok := y[k]; !ok || yv != xv {
			return false
		}
	}
	return true
}

func CompareMaps(mapA, mapB map[string]string) bool {
	if len(mapA) != len(mapB) {
		return false
	}

	for key, valueA := range mapA {
		valueB, ok := mapB[key]
		if !ok || valueA != valueB {
			return false
		}
	}

	return true
}
