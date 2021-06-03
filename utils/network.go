package utils

import (
	"../beans"
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

func RequestExperiment(url string, requestPrams map[string]interface{}, to time.Duration) ([]beans.Experiment, error) {
	var resp *http.Response

	data, _ := json.Marshal(requestPrams)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))

	client := &http.Client{Timeout: to}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	response := Response{}
	err = json.Unmarshal(body, &response)
	if err == nil && response.Status == "SUCCESS" {
		return response.Results, nil
	} else {
		return nil, errors.New(string(response.Error))
	}
}

type Response struct {
	Status    string             `json:"status"`
	ErrorType string             `json:"error_type"`
	Error     string             `json:"error"`
	Results   []beans.Experiment `json:"results"`
}