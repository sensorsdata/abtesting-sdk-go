package sensorsabtest

import (
	"errors"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/abtesting-sdk-go/utils/lru"
	utils2 "github.com/sensorsdata/sa-sdk-go/utils"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// 用户的试验缓存
var experimentCache = lru.New(4096)
var userExperimentsCache = lru.New(4096)
var experimentLock = sync.Mutex{}
var userExperimentTime = lru.New(4096)

// $ABTestTrigger 事件缓存
var eventsTime = lru.New(4096)
var hitExperiments = lru.New(4096)
var eventsLock = sync.Mutex{}

// 筛选试验
func filterExperiment(requestParam beans.RequestParam, experiments []beans.InnerExperiment) beans.InnerExperiment {
	var experimentParam = requestParam.ParamName
	// 遍历试验
	for _, experiment := range experiments {
		// 遍历试验变量
		for _, variable := range experiment.VariableList {
			if experimentParam == variable.Name {
				value, err := castValue(requestParam.DefaultValue, variable)
				if err == nil {
					experiment.Result = value
					return experiment
				}
			}
		}
	}

	return beans.InnerExperiment{
		Result: requestParam.DefaultValue,
	}
}

func filterOutList(requestParam beans.RequestParam, experiments []beans.InnerExperiment) []beans.InnerExperiment {
	var experimentParam = requestParam.ParamName
	var outExperiments = make([]beans.InnerExperiment, len(experiments))
	// 遍历试验
	var index = 0
	for _, experiment := range experiments {
		// 遍历试验变量
		for _, variable := range experiment.VariableList {
			if experimentParam == variable.Name {
				value, err := castValue(requestParam.DefaultValue, variable)
				if err == nil {
					experiment.Result = value
					outExperiments[index] = experiment
					index++
				}
			}
		}
	}

	return outExperiments
}

// 从缓存读取 $ABTestTrigger
func isEventNotExistOrExpired(idEvent string, innerExperiment beans.InnerExperiment, timeout time.Duration) bool {
	eventsLock.Lock()
	defer eventsLock.Unlock()
	lastTime, ok := eventsTime.Get(idEvent)
	if ok {
		expired := (utils2.NowMs() - lastTime.(int64)) > int64(timeout*time.Minute/time.Millisecond)
		if !expired {
			// 如果未过期，则判断 abtest_experiment_result_id 是否相同
			hitExperimentResultId, err := hitExperiments.Get(idEvent)
			if err {
				return hitExperimentResultId.(string) != innerExperiment.AbtestExperimentResultId
			}
		}
		return expired
	}
	return true
}

// 保存 $ABTestTrigger 到缓存中
func saveEvent2Cache(idEvent string, innerExperiment beans.InnerExperiment, sensors *SensorsABTest) {
	// 缓存 $ABTestTrigger 事件
	if sensors.config.EnableEventCache {
		eventsLock.Lock()
		defer eventsLock.Unlock()
		eventsTime.Remove(idEvent)
		eventsTime.Add(idEvent, utils2.NowMs())
		hitExperiments.Add(idEvent, innerExperiment.AbtestExperimentResultId)
	}
}

func castValue(defaultValue interface{}, variables beans.Variables) (interface{}, error) {
	var defaultType = reflect.TypeOf(defaultValue)
	if (variables.Type == "STRING" || variables.Type == "JSON") && "string" == defaultType.String() {
		return variables.Value, nil
	} else if variables.Type == "INTEGER" && "int" == defaultType.String() {
		return strconv.Atoi(variables.Value)
	} else if variables.Type == "INTEGER" && "int8" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 8)
	} else if variables.Type == "INTEGER" && "int16" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 16)
	} else if variables.Type == "INTEGER" && "int32" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 32)
	} else if variables.Type == "INTEGER" && "int64" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 64)
	} else if variables.Type == "BOOLEAN" && "bool" == defaultType.String() {
		return strconv.ParseBool(variables.Value)
	}
	return defaultValue, errors.New("castValue No Type Found")
}

// 从缓存读取试验
func loadExperimentCache(idKey string) (interface{}, bool) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	// 读取用户试验
	var userExperiments []beans.UserExperiment
	userCache, ok := userExperimentsCache.Get(idKey)
	if ok {
		userExperiments = userCache.([]beans.UserExperiment)
	}
	if len(userExperiments) == 0 {
		return nil, false
	}

	var innerExperiments = make([]beans.InnerExperiment, len(userExperiments))
	var index = 0
	for _, userExperiment := range userExperiments {
		tempExperiment, ok := experimentCache.Get(getExperimentKey1(userExperiment))
		if ok {
			innerExperiment := tempExperiment.(beans.InnerExperiment)
			innerExperiment.Cacheable = userExperiment.Cacheable
			innerExperiment.IsControlGroup = userExperiment.IsControlGroup
			innerExperiment.IsWhiteList = userExperiment.IsWhiteList
			innerExperiments[index] = innerExperiment
			index++
		}
	}

	return innerExperiments, true
}

// 保存试验到缓存
func saveExperiment2Cache(idKey string, experiments []beans.InnerExperiment) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	var userExperiments []beans.UserExperiment
	userCache, ok := userExperimentsCache.Get(idKey)
	if !ok {
		userExperiments = make([]beans.UserExperiment, len(experiments))
	} else {
		userExperiments = userCache.([]beans.UserExperiment)
	}
	var index = 0
	for _, innerExperiment := range experiments {
		if !innerExperiment.Cacheable && innerExperiment.SubjectId != "" { //新 SaaS 环境
			continue
		}
		// 保存单个试验试验
		experimentKey := getExperimentKey(innerExperiment)
		experimentCache.Add(experimentKey, innerExperiment)
		// 记录映射关系
		userExperiment := beans.UserExperiment{
			AbtestExperimentId:       innerExperiment.AbtestExperimentId,
			AbtestExperimentGroupId:  innerExperiment.AbtestExperimentGroupId,
			AbtestExperimentResultId: innerExperiment.AbtestExperimentResultId,
			Cacheable:                innerExperiment.Cacheable,
			IsControlGroup:           innerExperiment.IsControlGroup,
			IsWhiteList:              innerExperiment.IsWhiteList,
		}
		userExperiments[index] = userExperiment
		index++
	}

	// 保存用户映射试验
	userExperimentsCache.Add(idKey, userExperiments)
	userExperimentTime.Add(idKey, utils2.NowMs())
}

// 拼接网络请求参数
func buildRequestParam(distinctId string, isLoginId bool, requestParam beans.RequestParam) map[string]interface{} {
	var params = make(map[string]interface{})
	if isLoginId {
		params["login_id"] = distinctId
	} else {
		params["anonymous_id"] = distinctId
	}

	params["abtest_lib_version"] = SDK_VERSION
	params["platform"] = LIB_NAME
	if requestParam.Properties != nil && len(requestParam.Properties) > 0 {
		params["custom_properties"] = requestParam.Properties
	}
	if requestParam.CustomIDs != nil && len(requestParam.CustomIDs) > 0 {
		params["custom_ids"] = requestParam.CustomIDs
	}

	return params
}

// 拼接缓存唯一标识
func getEventKey(distinctId string, customIds map[string]interface{}, innerExperiment beans.InnerExperiment) string {
	if innerExperiment.SubjectId != "" {
		return innerExperiment.SubjectId + "$" + innerExperiment.SubjectName + "$" + innerExperiment.AbtestExperimentId
	} else {
		return distinctId + "$" + utils.MapToJson(customIds) + "$" + innerExperiment.AbtestExperimentId + "$" + innerExperiment.AbtestExperimentGroupId
	}
}

func getExperimentUserKey(distinctId string, customIds map[string]interface{}, isLoginId bool) string {
	return distinctId + "$" + utils.MapToJson(customIds) + "$" + strconv.FormatBool(isLoginId)
}

func getExperimentKey(experiment beans.InnerExperiment) string {
	return experiment.AbtestExperimentId + "$" + experiment.AbtestExperimentGroupId + "$" + experiment.AbtestExperimentResultId
}

func getExperimentKey1(experiment beans.UserExperiment) string {
	return experiment.AbtestExperimentId + "$" + experiment.AbtestExperimentGroupId + "$" + experiment.AbtestExperimentResultId
}

func isExperimentExpired(idKey string, timeout time.Duration) bool {
	lastTime, ok := userExperimentTime.Get(idKey)
	if ok {
		return (utils2.NowMs() - lastTime.(int64)) > int64(timeout*time.Minute/time.Millisecond)
	}
	return true
}
