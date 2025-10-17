package sensorsabtest

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	sensorsanalytics "github.com/sensorsdata/sa-sdk-go"
)

const (
	SDK_VERSION = "0.2.0"
	LIB_NAME    = "Golang"
)

// BuildAllExperimentsResultParams 构建所有实验结果的参数
type BuildAllExperimentsResultParams struct {
	ExperimentResponse     utils.Response    // 解析后的实验响应
	RawResponseBody        string            // 原始响应体字符串
	DistinctId             string            // 用户标识
	IsLoginId              bool              // 是否为登录ID
	CustomIDs              map[string]string // 自定义ID映射
	EnableAutoTrackABEvent bool              // 是否启用自动埋点
	Timestamp              int64             // 请求时间戳
}

type SensorsABTest struct {
	config           beans.ABTestConfig
	sensorsAnalytics sensorsanalytics.SensorsAnalytics
}

func InitSensorsABTest(abConfig beans.ABTestConfig) (error, SensorsABTest) {
	err, copyConfig := initConfig(abConfig)
	return err, SensorsABTest{
		config:           copyConfig,
		sensorsAnalytics: abConfig.SensorsAnalytics,
	}
}

/*
拉取最新试验计划
*/
func (sensors *SensorsABTest) AsyncFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	err, experiment := loadExperimentFromNetwork(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	return nil, experiment
}

/*
优先从缓存获取试验变量，如果缓存没有则从网络拉取
*/
func (sensors *SensorsABTest) FastFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	err, experiment := loadExperimentFromCache(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}

	}

	return nil, experiment
}

func (sensors *SensorsABTest) TrackABTestTrigger(experiment beans.Experiment, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	return sensors.TrackABTestTriggerWithCustomId(experiment, nil, property)
}

func (sensors *SensorsABTest) TrackABTestTriggerWithCustomId(experiment beans.Experiment, customId map[string]string, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	trackABTestEventOuter(experiment.DistinctId, experiment.IsLoginId, experiment, sensors, property, customId)
	return nil
}

// 检查请求参数是否合法
func checkRequestParams(param beans.RequestParam) error {
	if param.ParamName == "" {
		return errors.New("RequestParam.ParamName must not be empty")
	}

	if param.DefaultValue == nil {
		return errors.New("RequestParam.DefaultValue must not be nil")
	}

	var err error
	// 检查自定义属性
	if len(param.Properties) > 0 {
		err = utils.CheckProperty(param.Properties)
	}

	// 检查自定义主体
	if len(param.CustomIDs) > 0 {
		err = utils.CheckCustomIds(param.CustomIDs)
	}
	return err
}

func checkId(id string) error {
	if id == "" {
		return errors.New("DistinctId must not be empty")
	}
	return nil
}

func initConfig(abConfig beans.ABTestConfig) (error, beans.ABTestConfig) {
	if abConfig.APIUrl == "" {
		return errors.New("APIUrl must not be null or empty"), abConfig
	}

	var config = beans.ABTestConfig{}
	if abConfig.ExperimentCacheSize <= 0 {
		config.ExperimentCacheSize = 4096
	} else {
		config.ExperimentCacheSize = abConfig.ExperimentCacheSize
	}

	if abConfig.EventCacheSize <= 0 {
		config.EventCacheSize = 4096
	} else {
		config.EventCacheSize = abConfig.EventCacheSize
	}

	if abConfig.ExperimentCacheTime <= 0 || abConfig.ExperimentCacheTime > 24*60 {
		config.ExperimentCacheTime = 24 * 60
	} else {
		config.ExperimentCacheTime = abConfig.ExperimentCacheTime
	}

	if abConfig.EventCacheTime <= 0 || abConfig.EventCacheTime > 24*60 {
		config.EventCacheTime = 24 * 60
	} else {
		config.EventCacheTime = abConfig.EventCacheTime
	}

	config.SensorsAnalytics = abConfig.SensorsAnalytics
	config.EnableEventCache = abConfig.EnableEventCache
	config.EnableRecordRequestCostTime = abConfig.EnableRecordRequestCostTime
	config.APIUrl = abConfig.APIUrl
	initCache(config)
	utils.InitTransport(getHTTPTransPortParam(abConfig))
	return nil, config
}

func getHTTPTransPortParam(abConfig beans.ABTestConfig) beans.HTTPTransportParam {
	param := beans.HTTPTransportParam{}
	if abConfig.HTTPTransportParam.MaxIdleConnsPerHost <= 0 {
		param.MaxIdleConnsPerHost = 5
	} else {
		param.MaxIdleConnsPerHost = abConfig.HTTPTransportParam.MaxIdleConnsPerHost
	}
	if abConfig.HTTPTransportParam.MaxIdleConns <= 0 {
		param.MaxIdleConns = 20
	} else {
		param.MaxIdleConns = abConfig.HTTPTransportParam.MaxIdleConns
	}

	if abConfig.HTTPTransportParam.MaxConnsPerHost <= 0 {
		param.MaxConnsPerHost = 200
	} else {
		param.MaxConnsPerHost = abConfig.HTTPTransportParam.MaxConnsPerHost
	}

	if abConfig.HTTPTransportParam.IdleConnTimeoutMilliSeconds <= 0 {
		param.IdleConnTimeoutMilliSeconds = 30 * 1000
	} else {
		param.IdleConnTimeoutMilliSeconds = abConfig.HTTPTransportParam.IdleConnTimeoutMilliSeconds
	}

	if abConfig.HTTPTransportParam.DialTimeoutMilliSeconds <= 0 {
		param.DialTimeoutMilliSeconds = 30 * 1000
	} else {
		param.DialTimeoutMilliSeconds = abConfig.HTTPTransportParam.DialTimeoutMilliSeconds
	}

	if abConfig.HTTPTransportParam.DialKeepAliveMilliSeconds <= 0 {
		param.DialKeepAliveMilliSeconds = 30 * 1000
	} else {
		param.DialKeepAliveMilliSeconds = abConfig.HTTPTransportParam.DialKeepAliveMilliSeconds
	}
	return param
}

/*
获取用户在所有试验下的分流结果
强制从网络获取最新数据，不使用缓存
*/
func (sensors *SensorsABTest) FetchAllExperiments(distinctId string, isLoginId bool, requestParam beans.FetchAllRequestParam) (error, beans.AllExperimentsResult) {
	// 参数校验
	err := checkId(distinctId)
	if err != nil {
		return err, beans.AllExperimentsResult{}
	}

	// 检查自定义属性
	if len(requestParam.Properties) > 0 {
		err = utils.CheckProperty(requestParam.Properties)
		if err != nil {
			return err, beans.AllExperimentsResult{}
		}
	}

	// 检查自定义主体
	if len(requestParam.CustomIDs) > 0 {
		err = utils.CheckCustomIds(requestParam.CustomIDs)
		if err != nil {
			return err, beans.AllExperimentsResult{}
		}
	}

	// 从网络获取所有试验
	params := buildGetAllRequestParam(distinctId, isLoginId, requestParam)
	experimentResponse, rawResponseBody, err := requestExperimentFromNetwork(sensors, params, int64(requestParam.TimeoutMilliseconds))
	if err != nil {
		return err, beans.NewAllExperimentsResultBuilder().
			DistinctId(distinctId).
			IsLoginId(isLoginId).
			CustomIDs(requestParam.CustomIDs).
			Experiments(make(map[string]beans.InnerExperiment)).
			Timestamp(time.Now().UnixMilli()).Build()
	}

	// 使用通用辅助函数构建结果
	buildParams := BuildAllExperimentsResultParams{
		ExperimentResponse:     experimentResponse,
		RawResponseBody:        rawResponseBody,
		DistinctId:             distinctId,
		IsLoginId:              isLoginId,
		CustomIDs:              requestParam.CustomIDs,
		EnableAutoTrackABEvent: requestParam.EnableAutoTrackABEvent,
	}
	result := sensors.buildAllExperimentsResult(buildParams)

	return nil, result
}

// 通用的构建 AllExperimentsResult 的辅助函数
func (sensors *SensorsABTest) buildAllExperimentsResult(params BuildAllExperimentsResultParams) beans.AllExperimentsResult {
	// 构建参数名到试验的映射
	experimentsMap := make(map[string]beans.InnerExperiment)

	// 处理 results（主要试验结果）
	for _, experiment := range params.ExperimentResponse.Results {
		// 为每个试验变量创建映射
		for _, variable := range experiment.VariableList {
			// 类型转换处理
			value, err := castValueFromString(variable.Value, variable)
			if err == nil {
				//如果不存在或者是白名单的情况，则加入映射
				if _, ok := experimentsMap[variable.Name]; !ok || experiment.IsWhiteList {
					// 创建一个新的试验对象，设置正确的结果值
					experimentCopy := experiment
					experimentCopy.Result = value
					experimentsMap[variable.Name] = experimentCopy
				}
			}
		}
	}

	// 创建 out_list 参数映射（用于埋点，但不返回结果值）
	outListMap := make(map[string][]beans.InnerExperiment)
	for _, experiment := range params.ExperimentResponse.OutList {
		for _, variable := range experiment.VariableList {
			outListMap[variable.Name] = append(outListMap[variable.Name], experiment)
		}
	}

	// 创建埋点回调函数（如果启用）
	var trackCallback func(string, beans.InnerExperiment)
	if params.EnableAutoTrackABEvent {
		// 闭包捕获当前上下文
		capturedDistinctId := params.DistinctId
		capturedIsLoginId := params.IsLoginId
		capturedCustomIDs := params.CustomIDs
		capturedTrackConfig := params.ExperimentResponse.TrackConfig
		capturedSensors := sensors
		capturedOutListMap := outListMap

		trackCallback = func(paramName string, experiment beans.InnerExperiment) {
			// 先为主要试验（results）埋点
			// 不为 0 值
			if experiment.AbtestExperimentId != "" {
				trackABTestEvent(capturedDistinctId, capturedIsLoginId, experiment, capturedSensors, nil, capturedCustomIDs, capturedTrackConfig)
			}
			// 检查 out_list 中是否也有相同参数的试验，如果有也要埋点
			if outExperiments, exists := capturedOutListMap[paramName]; exists {
				for _, outExperiment := range outExperiments {
					trackABTestEvent(capturedDistinctId, capturedIsLoginId, outExperiment, capturedSensors, nil, capturedCustomIDs, capturedTrackConfig)
				}
			}
		}
	}

	timestamp := params.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}

	// 使用 Builder 创建结果对象
	return beans.NewAllExperimentsResultBuilder().
		DistinctId(params.DistinctId).
		IsLoginId(params.IsLoginId).
		CustomIDs(params.CustomIDs).
		TrackCallback(trackCallback).
		Experiments(experimentsMap).
		ResponseBody(params.RawResponseBody).
		Timestamp(timestamp).
		Build()
}

/*
从原始响应体字符串加载 AllExperimentsResult
支持跨服务传递分流结果，避免重复请求
rawResponseBody: 原始网络响应的 body 字符串
customIds: 自定义主体 ID
*/
func (sensors *SensorsABTest) loadAllExperimentsFromResponseBody(data beans.DumpData, enableAutoTrackABEvent bool) (error, beans.AllExperimentsResult) {
	// 解析原始响应体
	experimentResponse, err := utils.ParseResponse(data.ResponseBody)
	if err != nil {
		return err, beans.AllExperimentsResult{}
	}

	// 参数校验
	err = checkId(data.DistinctId)
	if err != nil {
		return err, beans.AllExperimentsResult{}
	}

	// 使用通用辅助函数构建结果
	buildParams := BuildAllExperimentsResultParams{
		ExperimentResponse:     experimentResponse,
		RawResponseBody:        data.ResponseBody,
		DistinctId:             data.DistinctId,
		IsLoginId:              data.IsLoginId,
		CustomIDs:              data.CustomIDs,
		EnableAutoTrackABEvent: enableAutoTrackABEvent,
		Timestamp:              data.Timestamp,
	}
	result := sensors.buildAllExperimentsResult(buildParams)

	return nil, result
}

/*
从序列化的 JSON 字符串加载 AllExperimentsResult
*/
func (sensors *SensorsABTest) LoadAllExperiments(distinctId string, isLoginId bool, param beans.LoadDumpedParam, dumpData string) (error, beans.AllExperimentsResult) {
	// 解析序列化数据
	var data beans.DumpData
	err := json.Unmarshal([]byte(dumpData), &data)
	if err != nil {
		return err, beans.AllExperimentsResult{}
	}

	// 验证必要字段
	if data.ResponseBody == "" {
		return errors.New("invalid serialized data: missing response_body field"), beans.AllExperimentsResult{}
	}
	if data.DistinctId == "" {
		return errors.New("invalid serialized data: missing distinct_id field"), beans.AllExperimentsResult{}
	}

	// 校验传入的用户信息和序列化数据中的用户信息是否一致
	if data.DistinctId != distinctId || data.IsLoginId != isLoginId {
		return errors.New("user identity (distinctId, isLoginId) mismatch"), beans.AllExperimentsResult{}
	}

	// 校验 CustomIDs 是否一致
	if !utils.CompareMaps(data.CustomIDs, param.CustomIDs) {
		return errors.New("user identity (CustomIDs) mismatch"), beans.AllExperimentsResult{}
	}

	// 调用原有的方法，传入解析出的参数（包括 CustomIDs）
	return sensors.loadAllExperimentsFromResponseBody(data, param.EnableAutoTrackABEvent)
}
