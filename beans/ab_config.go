package beans

import (
	"github.com/sensorsdata/sa-sdk-go"
	"time"
)

type ABTestConfig struct {
	/*
		试验缓存时间，单位是分钟
	*/
	ExperimentCacheTime time.Duration
	/*
		试验总缓存用户量限制
	*/
	ExperimentCacheSize int

	/*
		$ABTestTrigger 事件缓存时间，单位是分钟
	*/
	EventCacheTime time.Duration
	/*
		$ABTestTrigger 事件总缓存用户量限制
	*/
	EventCacheSize int

	/*
		开启 A/B 事件缓存
	*/
	EnableEventCache bool

	/**
	开启请求耗时记录
	*/
	EnableRecordRequestCostTime bool

	/*
		API 地址
	*/
	APIUrl string

	/**
	用于 SDK 埋点 SensorsAnalytics
	*/
	SensorsAnalytics sensorsanalytics.SensorsAnalytics
}
