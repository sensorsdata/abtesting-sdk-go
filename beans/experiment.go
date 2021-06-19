package beans

type Experiment struct {
	// distinct_id 标识
	DistinctId string
	// 是否是登录 id
	IsLoginId bool
	// 试验变量值
	Result interface{}
	// 试验 ID
	AbtestExperimentId string `json:"abtest_experiment_id"`
	// 试验内分组 ID
	AbtestExperimentGroupId string `json:"abtest_experiment_group_id"`
	// 是否是对照组
	IsControlGroup bool `json:"is_control_group"`
	// 是否白名单
	IsWhiteList  bool        `json:"is_white_list"`
	VariableList []Variables `json:"variables"`
}

type Variables struct {
	// 试验参数
	Name string `json:"name"`
	// 变量值均为字符串
	Value string `json:"value"`
	// 变量类型
	Type string `json:"type"`
}
