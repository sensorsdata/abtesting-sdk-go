package beans

type Experiment struct {
	// distinct_id 标识
	DistinctId string
	// 是否是登录 id
	IsLoginId bool
	// 试验变量值
	Result             interface{}
	InternalExperiment InnerExperiment
}

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
