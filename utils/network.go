package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func RequestExperiment(url string, requestPrams map[string]interface{}, to time.Duration, enableRecordRequestCostTime bool) (Response, error) {
	var resp *http.Response

	data, _ := json.Marshal(requestPrams)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))

	abRequestStartTime := time.Now().UnixNano() / int64(time.Millisecond)
	req.Header.Add("X-AB-Request-Start-Time", fmt.Sprintf("%v", abRequestStartTime))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: to}
	resp, err := client.Do(req)

	if enableRecordRequestCostTime {
		abRequestEndTime := time.Now().UnixNano() / int64(time.Millisecond)
		recordAbRequestCostTime(resp, abRequestStartTime, abRequestEndTime)
	}

	if err != nil {
		return Response{}, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	response := Response{}
	var resMaps map[string]interface{}
	err = json.Unmarshal(body, &response)
	err = json.Unmarshal(body, &resMaps)
	if err == nil && response.Status == "SUCCESS" {
		if !strings.Contains(string(body), "track_config") {
			response.TrackConfig = beans.TrackConfig{
				ItemSwitch:        false,
				TriggerSwitch:     true,
				PropertySetSwitch: false,
				TriggerContentExt: []string{"abtest_experiment_result_id", "abtest_experiment_version"},
			}
		}
		defaultTrackConfig(&response, resMaps)
		return response, nil
	} else {
		return Response{}, errors.New(string(response.Error))
	}
}

func recordAbRequestCostTime(response *http.Response, abRequestStartTime int64, abRequestEndTime int64) {
	abRequestId := getAbRequestIdFromResponse(response)
	abRequestProcessTime := getAbRequestProcessTimeFromResponse(response)
	abRequestTotalTime := strconv.FormatInt(abRequestEndTime-abRequestStartTime, 10)
	fmt.Println("record ab request time consumption. requestId: ", abRequestId, ", requestTotalTime:", abRequestTotalTime, "ms, abRequestProcessTime:", abRequestProcessTime, "ms")
}

func getAbRequestIdFromResponse(response *http.Response) (abRequestId string) {
	if response != nil && response.Header != nil {
		abRequestId = response.Header.Get("X-AB-Request-Id")
		if abRequestId == "" {
			abRequestId = response.Header.Get("X-Request-Id")
		}
		if abRequestId != "" {
			return abRequestId
		}
	}
	return "unknown (not found)"
}

func getAbRequestProcessTimeFromResponse(response *http.Response) (abRequestProcessTime string) {
	if response != nil && response.Header != nil {
		abRequestProcessTime = response.Header.Get("X-AB-Request-Process-Time")
		if abRequestProcessTime != "" {
			return abRequestProcessTime
		}
	}
	return "unknown (not found)"
}

func defaultTrackConfig(response *Response, resMaps map[string]interface{}) {
	if !response.TrackConfig.TriggerSwitch {
		return
	}
	trackExt := response.TrackConfig.TriggerContentExt

	// 查找 result 试验组
	if resMaps["results"] != nil {
		results := resMaps["results"].([]interface{})
		for _, result := range results {
			// 查找第一个试验
			value := result.(map[string]interface{})
			for _, extConfig := range trackExt {
				if value[extConfig] != nil {
					updateExtValue(response.Results, value["abtest_experiment_id"].(string), extConfig, value[extConfig].(string), len(trackExt))
				}
			}
		}
	}

	// 查找 out_list 试验组
	if resMaps["out_list"] != nil {
		outlists := resMaps["out_list"].([]interface{})
		for _, result := range outlists {
			// 查找第一个试验
			value := result.(map[string]interface{})
			for _, extConfig := range trackExt {
				if value[extConfig] != nil {
					updateExtValue(response.OutList, value["abtest_experiment_id"].(string), extConfig, value[extConfig].(string), len(trackExt))
				}
			}
		}
	}
}

func updateExtValue(innerExperiments []beans.InnerExperiment, experimentId string, ext string, extValue string, configCount int) {
	for index, innerExperiment := range innerExperiments {
		if innerExperiment.AbtestExperimentId == experimentId {
			if innerExperiment.TrackExtValue == nil {
				innerExperiment.TrackExtValue = make(map[string]interface{}, configCount)
			}
			innerExperiment.TrackExtValue["$"+ext] = extValue
			innerExperiments[index] = innerExperiment
			break
		}
	}
}

type Response struct {
	Status      string                  `json:"status"`
	ErrorType   string                  `json:"error_type"`
	Error       string                  `json:"error"`
	Results     []beans.InnerExperiment `json:"results"`
	TrackConfig beans.TrackConfig       `json:"track_config"`
	OutList     []beans.InnerExperiment `json:"out_list"`
}
