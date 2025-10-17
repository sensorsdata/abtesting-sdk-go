package beans

import (
	"encoding/json"
)

type Experiment struct {
	// distinct_id 标识
	DistinctId string
	// 是否是登录 id
	IsLoginId bool
	// 自定义主体 ID
	CustomIDs map[string]string
	// 试验变量值
	Result             interface{}
	InternalExperiment InnerExperiment
}

// 在代码中有一些浅拷贝操作，新增字段时需要注意
type InnerExperiment struct {
	// 试验 ID
	AbtestExperimentId string `json:"abtest_experiment_id"`
	// 试验内分组 ID
	AbtestExperimentGroupId string `json:"abtest_experiment_group_id"`
	// 标识哪个版本的试验分组
	AbtestExperimentResultId string `json:"abtest_experiment_result_id"`
	// 试验类型
	ExperimentType string `json:"experiment_type"`
	// 命中主体
	SubjectName string `json:"subject_name"`
	// 主体 ID
	SubjectId string `json:"subject_id"`
	// 试验版本
	AbtestExperimentVersion string `json:"abtest_experiment_version"`
	// 是否粘性
	Stickiness string `json:"stickiness"`
	// 是否缓存
	Cacheable bool `json:"cacheable"`
	// 是否是对照组
	IsControlGroup bool `json:"is_control_group"`
	// 是否白名单
	IsWhiteList  bool        `json:"is_white_list"`
	VariableList []Variables `json:"variables"`
	// 试验变量值
	Result interface{}
	// TrackExt
	TrackExtValue map[string]interface{}
}

type UserExperiment struct {
	// 试验 ID
	AbtestExperimentId string `json:"abtest_experiment_id"`
	// 试验内分组 ID
	AbtestExperimentGroupId string `json:"abtest_experiment_group_id"`
	// 标识哪个版本的试验分组
	AbtestExperimentResultId string `json:"abtest_experiment_result_id"`
	// 是否缓存
	Cacheable bool `json:"cacheable"`
	// 是否是对照组
	IsControlGroup bool `json:"is_control_group"`
	// 是否白名单
	IsWhiteList bool `json:"is_white_list"`
}

type HitExperiment struct {
	// 试验 ID
	AbtestExperimentId string `json:"abtest_experiment_id"`
	// 试验内分组 ID
	AbtestExperimentGroupId string `json:"abtest_experiment_group_id"`
	// 标识哪个版本的试验分组
	AbtestExperimentResultId string `json:"abtest_experiment_result_id"`
}

type TrackConfig struct {
	ItemSwitch        bool     `json:"item_switch" default:"false"`
	TriggerSwitch     bool     `json:"trigger_switch" default:"true"`
	PropertySetSwitch bool     `json:"property_set_switch" default:"false"`
	TriggerContentExt []string `json:"trigger_content_ext" default:"[\"abtest_experiment_result_id\", \"abtest_experiment_version\"]"`
}

type Variables struct {
	// 试验参数
	Name string `json:"name"`
	// 变量值均为字符串
	Value string `json:"value"`
	// 变量类型
	Type string `json:"type"`
}

// AllExperimentsResult represents the result of fetching all experiments.
// Its fields are unexported to ensure immutability and it should be created via the builder.
type AllExperimentsResult struct {
	// distinct_id 标识
	distinctId string

	// 是否是登录 id
	isLoginId bool

	// 自定义主体 ID
	customIDs map[string]string

	// 埋点回调函数,在fetchAll的时候自动生成,GetValue 时会自动调用
	trackCallback func(paramName string, experiment InnerExperiment)

	// 全部的试验结果，用于支持 GetValue 方法
	experiments map[string]InnerExperiment

	// 原始网络响应 body，用于跨服务传递
	responseBody string

	// 请求时间
	timestamp int64
}

// DistinctId returns the distinct_id.
func (result *AllExperimentsResult) DistinctId() string {
	return result.distinctId
}

// IsLoginId returns whether the id is a login id.
func (result *AllExperimentsResult) IsLoginId() bool {
	return result.isLoginId
}

// CustomIDs returns the custom IDs.
func (result *AllExperimentsResult) CustomIDs() map[string]string {
	// 返回一个副本以防止外部修改 map
	if result.customIDs == nil {
		return nil
	}
	copiedCustomIDs := make(map[string]string, len(result.customIDs))
	for k, v := range result.customIDs {
		copiedCustomIDs[k] = v
	}
	return copiedCustomIDs
}

// Timestamp returns the timestamp.
func (result *AllExperimentsResult) Timestamp() int64 {
	return result.timestamp
}

func (result *AllExperimentsResult) SetTrackCallback(callback func(paramName string, experiment InnerExperiment)) {
	result.trackCallback = callback
}

// 安全的取值方法，自动埋点
func (result *AllExperimentsResult) GetValue(paramName string, defaultValue interface{}) interface{} {
	var res interface{}
	var hitExperiment InnerExperiment
	if experiment, exists := result.experiments[paramName]; exists {
		res = experiment.Result
		hitExperiment = experiment
	} else {
		res = defaultValue
	}
	// 如果开启了埋点，则进行埋点
	if result.trackCallback != nil {
		result.trackCallback(paramName, hitExperiment)
	}
	return res
}

// 获取 experiment对象，不会自动埋点，主要用于获取手动埋点的参数
func (result *AllExperimentsResult) GetExperiment(paramName string, defaultValue interface{}) Experiment {
	if experiment, exists := result.experiments[paramName]; exists {
		return Experiment{
			DistinctId:         result.DistinctId(),
			IsLoginId:          result.IsLoginId(),
			CustomIDs:          result.CustomIDs(),
			Result:             experiment.Result,
			InternalExperiment: experiment,
		}

	}
	return Experiment{
		Result: defaultValue,
	}
}

// 检查是否包含指定参数
func (result *AllExperimentsResult) HasParam(paramName string) bool {
	_, exists := result.experiments[paramName]
	return exists
}

// DumpData 用于跨服务传递的序列化数据结构
type DumpData struct {
	DistinctId   string            `json:"distinct_id"`
	IsLoginId    bool              `json:"is_login_id"`
	CustomIDs    map[string]string `json:"custom_ids,omitempty"`
	ResponseBody string            `json:"response_body"`
	Timestamp    int64             `json:"timestamp"`
}

// Dump 序列化完整的上下文信息（包括 distinct_id、is_login_id、custom_ids 和响应体）
// 返回 JSON 字符串，用于跨服务传递
func (result *AllExperimentsResult) Dump() (string, error) {
	data := DumpData{
		DistinctId:   result.distinctId,
		IsLoginId:    result.isLoginId,
		CustomIDs:    result.customIDs,
		ResponseBody: result.responseBody,
		Timestamp:    result.timestamp,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// AllExperimentsResultBuilder is used to construct an AllExperimentsResult object.
type AllExperimentsResultBuilder struct {
	distinctId    string
	isLoginId     bool
	customIDs     map[string]string
	trackCallback func(paramName string, experiment InnerExperiment)
	experiments   map[string]InnerExperiment
	responseBody  string
	timestamp     int64
}

// NewAllExperimentsResultBuilder creates a new builder.
func NewAllExperimentsResultBuilder() *AllExperimentsResultBuilder {
	return &AllExperimentsResultBuilder{}
}

func (b *AllExperimentsResultBuilder) DistinctId(distinctId string) *AllExperimentsResultBuilder {
	b.distinctId = distinctId
	return b
}

func (b *AllExperimentsResultBuilder) IsLoginId(isLoginId bool) *AllExperimentsResultBuilder {
	b.isLoginId = isLoginId
	return b
}

func (b *AllExperimentsResultBuilder) CustomIDs(customIDs map[string]string) *AllExperimentsResultBuilder {
	b.customIDs = customIDs
	return b
}

func (b *AllExperimentsResultBuilder) TrackCallback(callback func(paramName string, experiment InnerExperiment)) *AllExperimentsResultBuilder {
	b.trackCallback = callback
	return b
}

func (b *AllExperimentsResultBuilder) Experiments(experiments map[string]InnerExperiment) *AllExperimentsResultBuilder {
	b.experiments = experiments
	return b
}

func (b *AllExperimentsResultBuilder) ResponseBody(responseBody string) *AllExperimentsResultBuilder {
	b.responseBody = responseBody
	return b
}

func (b *AllExperimentsResultBuilder) Timestamp(timestamp int64) *AllExperimentsResultBuilder {
	b.timestamp = timestamp
	return b
}

// Build creates and returns the immutable AllExperimentsResult.
func (b *AllExperimentsResultBuilder) Build() AllExperimentsResult {
	return AllExperimentsResult{
		distinctId:    b.distinctId,
		isLoginId:     b.isLoginId,
		customIDs:     b.customIDs,
		trackCallback: b.trackCallback,
		experiments:   b.experiments,
		responseBody:  b.responseBody,
		timestamp:     b.timestamp,
	}
}
