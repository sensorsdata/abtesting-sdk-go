package sensorsabtest

import (
	"fmt"
	"time"

	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/abtesting-sdk-go/utils/lru"
)

// 插件版本号标记位
var isFirstEvent = true

// 埋点事件上次触发的时间
var lastTimeEvent string

// 埋点配置
var trackConfig beans.TrackConfig

func loadExperimentFromNetwork(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, beans.Experiment) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	response, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam, sensors.config.EnableRecordRequestCostTime)
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}
	trackConfig = response.TrackConfig
	experiment := beans.Experiment{}
	// 从 result 中查找
	innerExperiment := filterExperiment(requestParam, response.Results)
	if innerExperiment.AbtestExperimentId != "" {
		if isTrack {
			trackABTestEvent(distinctId, isLoginId, innerExperiment, sensors, nil, requestParam.CustomIDs, response.TrackConfig)
		}
		// 回调试验变量给客户
		tempExperiment := beans.Experiment{
			DistinctId:         distinctId,
			IsLoginId:          isLoginId,
			Result:             innerExperiment.Result,
			InternalExperiment: innerExperiment,
		}
		experiment = tempExperiment
	}

	// 从 out_list 中查找
	outExperiments := filterOutList(requestParam, response.OutList)
	for _, outExperiment := range outExperiments {
		if outExperiment.AbtestExperimentId != "" {
			if isTrack {
				trackABTestEvent(distinctId, isLoginId, outExperiment, sensors, nil, requestParam.CustomIDs, response.TrackConfig)
			}
		}
	}

	if experiment.InternalExperiment.AbtestExperimentId != "" { // 说明找到试验了
		return nil, experiment
	}

	return nil, beans.Experiment{
		DistinctId: distinctId,
		IsLoginId:  isLoginId,
		Result:     requestParam.DefaultValue,
	}
}

func loadExperimentFromCache(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, beans.Experiment) {
	var innerExperiment beans.InnerExperiment
	var isRequestNetwork = false
	idKey := getExperimentUserKey(distinctId, requestParam.CustomIDs, isLoginId)
	if isExperimentExpired(idKey, sensors.config.ExperimentCacheTime) {
		idKey := getExperimentUserKey(distinctId, requestParam.CustomIDs, isLoginId)
		// 进行清理缓存
		experimentCache.Remove(idKey)
		isRequestNetwork = true
	} else {
		experiments, ok := loadExperimentCache(idKey)
		if experiments != nil || ok {
			innerExperiment = filterExperiment(requestParam, experiments.([]beans.InnerExperiment))
		}
		if experiments == nil || !ok || innerExperiment.AbtestExperimentId == "" {
			isRequestNetwork = true
		}
	}
	var outExperiments []beans.InnerExperiment
	if isRequestNetwork {
		// 从网络请求试验
		response, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam, sensors.config.EnableRecordRequestCostTime)
		if err != nil {
			return err, beans.Experiment{
				Result: requestParam.DefaultValue,
			}
		}
		trackConfig = response.TrackConfig
		// 缓存试验
		saveExperiment2Cache(idKey, response.Results)
		// 筛选试验
		innerExperiment = filterExperiment(requestParam, response.Results)

		// 从 out_list 中查找
		outExperiments = filterOutList(requestParam, response.OutList)
	}

	experiment := beans.Experiment{
		DistinctId: distinctId,
		IsLoginId:  isLoginId,
		Result:     requestParam.DefaultValue,
	}
	if innerExperiment.AbtestExperimentId != "" {
		if isTrack {
			trackABTestEvent(distinctId, isLoginId, innerExperiment, sensors, nil, requestParam.CustomIDs, trackConfig)
		}
		// 回调试验变量给客户
		tempExperiment := beans.Experiment{
			DistinctId:         distinctId,
			IsLoginId:          isLoginId,
			Result:             innerExperiment.Result,
			InternalExperiment: innerExperiment,
		}
		experiment = tempExperiment
	}

	for _, outExperiment := range outExperiments {
		if outExperiment.AbtestExperimentId != "" {
			if isTrack {
				trackABTestEvent(distinctId, isLoginId, outExperiment, sensors, nil, requestParam.CustomIDs, trackConfig)
			}
		}
	}
	return nil, experiment
}

// 从网络加载试验
func requestExperimentOnNetwork(apiUrl string, distinctId string, isLoginId bool, requestParam beans.RequestParam, enableRecordRequestCostTime bool) (utils.Response, error) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	return utils.RequestExperiment(apiUrl, buildRequestParam(distinctId, isLoginId, requestParam), time.Duration(requestParam.TimeoutMilliseconds)*time.Millisecond, enableRecordRequestCostTime)
}

func trackABTestEventOuter(distinctId string, isLoginId bool, experiment beans.Experiment, sensors *SensorsABTest, properties map[string]interface{}, customIDs map[string]interface{}) {
	trackABTestEvent(distinctId, isLoginId, experiment.InternalExperiment, sensors, properties, customIDs, trackConfig)
}

func trackABTestEvent(distinctId string, isLoginId bool, innerExperiment beans.InnerExperiment, sensors *SensorsABTest, properties map[string]interface{}, customIDs map[string]interface{}, config beans.TrackConfig) {
	if sensors.config.SensorsAnalytics.C == nil {
		return
	}
	// 是白名单，则不触发 $ABTestTrigger 事件
	if innerExperiment.IsWhiteList {
		return
	}

	isNewSaas := innerExperiment.SubjectId != ""
	if isNewSaas && !config.TriggerSwitch {
		return
	}

	idEvent := getEventKey(distinctId, customIDs, innerExperiment)
	if isNewSaas && innerExperiment.Cacheable || !isNewSaas {
		// 如果在缓存中，则不触发 $ABTestTrigger 事件
		ok := isEventNotExistOrExpired(idEvent, innerExperiment, sensors.config.EventCacheTime)
		if !ok {
			return
		}
		saveEvent2Cache(idEvent, innerExperiment, sensors)
	}

	if properties == nil {
		properties = map[string]interface{}{
			"$abtest_experiment_id":       innerExperiment.AbtestExperimentId,
			"$abtest_experiment_group_id": innerExperiment.AbtestExperimentGroupId,
		}
	} else {
		properties["$abtest_experiment_id"] = innerExperiment.AbtestExperimentId
		properties["$abtest_experiment_group_id"] = innerExperiment.AbtestExperimentGroupId
	}

	// 拼接 abtest_result
	if config.PropertySetSwitch && innerExperiment.AbtestExperimentResultId != "-1" {
		properties["abtest_result"] = []string{innerExperiment.AbtestExperimentResultId}
	}

	// 拼接 trigger_content_ext
	if innerExperiment.TrackExtValue != nil {
		for key, value := range innerExperiment.TrackExtValue {
			properties[key] = value
		}
	}

	currentTime := time.Now().Format("2006-01-02")
	if isFirstEvent || currentTime != lastTimeEvent {
		properties["$lib_plugin_version"] = []string{"golang_abtesting:" + SDK_VERSION}
		isFirstEvent = false
		lastTimeEvent = currentTime
	}
	if innerExperiment.SubjectName == "DEVICE" {
		properties["anonymous_id"] = innerExperiment.SubjectId
	}
	err := sensors.sensorsAnalytics.Track(distinctId, "$ABTestTrigger", properties, isLoginId)
	if err != nil {
		fmt.Println("$ABTestTrigger track failed, error : ", err)
	}
	sensors.sensorsAnalytics.Flush()
}

// 初始化缓存大小
func initCache(config beans.ABTestConfig) {
	if config.EventCacheSize != 0 {
		eventsTime = lru.New(config.EventCacheSize)
		hitExperiments = lru.New(config.EventCacheSize)
	}

	if config.ExperimentCacheSize != 0 {
		experimentCache = lru.New(config.ExperimentCacheSize)
		userExperimentTime = lru.New(config.ExperimentCacheSize)
		userExperimentsCache = lru.New(config.ExperimentCacheSize)
	}
}
