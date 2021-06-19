package sensorsabtest

import (
	"errors"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/sa-sdk-go"
)

const (
	SDK_VERSION = "0.0.1"
	LIB_NAME    = "Golang"
)

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
func (sensors *SensorsABTest) AsyncFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, interface{}, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, experiment := loadExperimentFromNetwork(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, requestParam.DefaultValue, beans.Experiment{}
	}

	return nil, variable, experiment
}

/*
	优先从缓存获取试验变量，如果缓存没有则从网络拉取
*/
func (sensors *SensorsABTest) FastFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, interface{}, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, experiment := loadExperimentFromCache(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, requestParam.DefaultValue, beans.Experiment{}
	}

	return nil, variable, experiment
}

func (sensors *SensorsABTest) TrackABTestTrigger(experiment beans.Experiment, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	trackABTestEvent(experiment.DistinctId, experiment.IsLoginId, experiment, sensors, property)
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

	return nil
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
		config.ExperimentCacheTime = abConfig.ExperimentCacheTime
	}

	if abConfig.EventCacheSize <= 0 {
		config.EventCacheSize = 4096
	} else {
		config.EventCacheTime = abConfig.EventCacheTime
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
	config.APIUrl = abConfig.APIUrl
	initCache(config)
	return nil, config
}
