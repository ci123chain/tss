package tgo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

func getDynamicRedisAddress(url string) (err error, address []string) {
	header := []string{"Accept:"}
	var ret []byte
	ret, err = curlGet(url, header)
	if err != nil {
		LogErrorw(LogNameApi, "getDynamicRedisAddress curlGet error", err)
		return
	}
	data := new(redisAddressResp)
	err = json.Unmarshal(ret, data)
	if err != nil {
		LogErrorw(LogNameApi, "getDynamicRedisAddress json Unmarshal error", err)
		return
	}
	if data.Success {
		return err, data.Content.Result
	} else {
		LogErrorw(LogNameApi, "getDynamicRedisAddress res error", err)
	}
	return
}

type redisAddressResp struct {
	Success bool                    `json:"success"`
	Code    string                  `json:"code"`
	Msg     string                  `json:"msg"`
	Content redisAddressRespContent `json:"content"`
}

type redisAddressRespContent struct {
	Result []string `json:"result"`
}

func curlGet(url string, header []string) (ret []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ret, err
	}
	for _, v := range header {
		t := strings.Split(v, ":")
		length := len(t)
		if length == 2 {
			req.Header.Add(t[0], t[1])
		} else if length == 1 {
			req.Header.Add(t[0], "")
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return ret, err
	}
	defer resp.Body.Close()
	ret, err = ioutil.ReadAll(resp.Body)

	return ret, err
}
