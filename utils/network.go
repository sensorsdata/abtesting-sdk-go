package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sensorsdata/abtesting-sdk-go/beans"
)

var httpTransport = &http.Transport{}

func InitTransport(httpTrans beans.HTTPTransportParam) {
	httpTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(httpTrans.DialTimeoutMilliSeconds) * time.Millisecond,
			KeepAlive: time.Duration(httpTrans.DialKeepAliveMilliSeconds) * time.Millisecond,
		}).DialContext,
		MaxIdleConns:        httpTrans.MaxIdleConns,
		MaxIdleConnsPerHost: httpTrans.MaxIdleConnsPerHost,
		MaxConnsPerHost:     httpTrans.MaxConnsPerHost,
		IdleConnTimeout:     time.Duration(httpTrans.IdleConnTimeoutMilliSeconds) * time.Millisecond,
	}
}

// 通用的HTTP请求执行函数，避免重复代码
func executeHttpRequest(url string, requestParams map[string]interface{}, timeout time.Duration, enableRecordRequestCostTime bool) (*http.Response, error) {
	data, err := json.Marshal(requestParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request params: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	abRequestStartTime := time.Now().UnixNano() / int64(time.Millisecond)
	req.Header.Add("X-AB-Request-Start-Time", fmt.Sprintf("%v", abRequestStartTime))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout, Transport: httpTransport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if enableRecordRequestCostTime {
		recordRequestCostTime(resp, abRequestStartTime)
	}

	return resp, nil
}

// 统一的实验请求函数，返回解析后的实验响应和原始响应体字符串
func RequestExperiment(url string, requestParams map[string]interface{}, timeout time.Duration, enableRecordRequestCostTime bool) (Response, string, error) {
	resp, err := executeHttpRequest(url, requestParams, timeout, enableRecordRequestCostTime)
	if err != nil {
		return Response{}, "", err
	}

	return processResponse(resp)
}

func truncateBody(arr []byte, maxLen int) string {
	bodyStr := string(arr)
	if len(bodyStr) > maxLen {
		return bodyStr[:maxLen]
	}
	return bodyStr
}

// 通用的响应处理函数，读取并验证HTTP响应
func processHttpResponse(resp *http.Response) (string, error) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("close body error: ", err)
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := string(body)

	if !isStatusCodeValid(resp.StatusCode) {
		return bodyStr, fmt.Errorf("response status code is not valid, status code: %d, response: %s", resp.StatusCode, truncateBody(body, 200))
	}

	return bodyStr, nil
}

// 返回解析后的实验响应和原始响应体字符串的处理函数
func processResponse(resp *http.Response) (Response, string, error) {
	rawBodyStr, err := processHttpResponse(resp)
	if err != nil {
		return Response{}, rawBodyStr, err
	}

	// 解析实验响应
	experimentResponse, err := ParseResponse(rawBodyStr)
	return experimentResponse, rawBodyStr, err
}

func recordRequestCostTime(resp *http.Response, abRequestStartTime int64) {
	abRequestEndTime := time.Now().UnixNano() / int64(time.Millisecond)
	recordAbRequestCostTime(resp, abRequestStartTime, abRequestEndTime)
}

func isStatusCodeValid(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
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

// 从原始响应体字符串解析实验响应
func ParseResponse(rawBodyStr string) (Response, error) {
	experimentResponse := Response{}
	var responseMaps map[string]interface{}

	bodyBytes := []byte(rawBodyStr)

	err := json.Unmarshal(bodyBytes, &experimentResponse)
	if err != nil {
		return Response{}, err
	}

	err = json.Unmarshal(bodyBytes, &responseMaps)
	if err != nil {
		return Response{}, err
	}

	if experimentResponse.Status == "SUCCESS" {
		if !strings.Contains(rawBodyStr, "track_config") {
			experimentResponse.TrackConfig = beans.TrackConfig{
				ItemSwitch:        false,
				TriggerSwitch:     true,
				PropertySetSwitch: false,
				TriggerContentExt: []string{"abtest_experiment_result_id", "abtest_experiment_version"},
			}
		}
		defaultTrackConfig(&experimentResponse, responseMaps)
		return experimentResponse, nil
	} else {
		return Response{}, errors.New(experimentResponse.Error)
	}
}
