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
	config           beans.ABConfig
	sensorsAnalytics sensorsanalytics.SensorsAnalytics
}

func InitSensorsABTesting(abConfig beans.ABConfig) SensorsABTest {
	return SensorsABTest{
		config:           abConfig,
		sensorsAnalytics: abConfig.SensorsAnalytics,
	}
}

/*
	拉取最新试验变量，由 SDK 内部触发 $ABTestTrigger 埋点事件
*/
func (sensors *SensorsABTest) AsyncFetchABTest(requestParam beans.RequestParam, defaultValue interface{}) (error, interface{}) {
	_, err := checkRequestParams(requestParam)
	if err != nil {
		return err, defaultValue
	}

	err, variable, _ := loadExperimentFromNetwork(sensors, requestParam, defaultValue, true)

	if err != nil {
		return err, defaultValue
	}

	return nil, variable
}

/*
	拉取最新试验计划，SDK 不触发 $ABTestTrigger 埋点事件
*/
func (sensors *SensorsABTest) AsyncFetchABTestExperiment(requestParam beans.RequestParam, defaultValue interface{}) (error error, variable interface{}, experiment beans.Experiment) {
	_, err := checkRequestParams(requestParam)
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromNetwork(sensors, requestParam, defaultValue, false)

	if err != nil {
		return err, defaultValue, beans.Experiment{}
	}

	return nil, variable, exper
}

/*
	优先从缓存获取试验变量，如果缓存没有则从网络拉取，并且 SDK 内部触发 $ABTestTrigger 埋点事件
*/
func (sensors *SensorsABTest) FastFetchABTest(requestParam beans.RequestParam, defaultValue interface{}) (error, interface{}) {
	_, err := checkRequestParams(requestParam)
	if err != nil {
		return err, defaultValue
	}

	err, variable, _ := loadExperimentFromCache(sensors, requestParam, defaultValue, true)

	if err != nil {
		return err, defaultValue
	}

	return nil, variable
}

/*
	优先从缓存获取试验变量，如果缓存没有则从网络拉取，并且 SDK 不触发 $ABTestTrigger 埋点事件
*/
func (sensors *SensorsABTest) FastFetchABTestExperiment(requestParam beans.RequestParam, defaultValue interface{}) (error error, variable interface{}, experiment beans.Experiment) {
	_, err := checkRequestParams(requestParam)
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromCache(sensors, requestParam, defaultValue, false)

	if err != nil {
		return err, defaultValue, beans.Experiment{}
	}

	return nil, variable, exper
}

func (sensors *SensorsABTest) TrackABTestTrigger(distinctId string, isLoginId bool, experiment beans.Experiment, property map[string]interface{}) {
	trackABTestEvent(distinctId, isLoginId, experiment, sensors, property)
}

// 检查请求参数是否合法
func checkRequestParams(requestParam beans.RequestParam) (bool, error) {
	if requestParam.AnonymousId == "" && requestParam.LoginId == "" {
		return false, errors.New("AnonymousId and LoginId must not be empty")
	}
	return true, nil
}
