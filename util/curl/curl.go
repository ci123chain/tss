package curl

//curl 封装转发

import (
	json "github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type ReturnJson struct {
	Code    int
	Message string
	Data    interface{}
}

func CurlGet(url string, header []string) (ret []byte, err error) {
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

func CurlGetWitchTimeOut(url string, header []string, timeout time.Duration) (ret []byte, err error) {
	client := &http.Client{}
	client.Timeout = timeout
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

func CurlGetReturnJson(url string, header []string) (r ReturnJson) {
	r = ReturnJson{}
	ret, err := CurlGet(url, header)
	if err == nil {
		data, err := json.NewJson(ret)
		if err == nil {
			r.Code, err = data.Get("code").Int()
			r.Message, err = data.Get("message").String()
			r.Data = data.Get("data").Interface()
		}
	}

	return r
}

func CurlPost(url string, header []string, data string) (ret []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return ret, err
	}
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value; charset=utf-8")
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

func CurlPostNew(url string, header []string, data string) (ret []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return ret, err
	}
	isSetContentType := false
	for _, v := range header {
		t := strings.Split(v, ":")
		length := len(t)
		if length == 2 {
			if strings.ToLower(t[0]) == "content-type" {
				isSetContentType = true
			}
			req.Header.Add(t[0], t[1])
		} else if length == 1 {
			req.Header.Add(t[0], "")
		}
	}
	if !isSetContentType {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value; charset=utf-8")
	}
	resp, err := client.Do(req)
	if err != nil {
		return ret, err
	}
	defer resp.Body.Close()
	ret, err = ioutil.ReadAll(resp.Body)

	return ret, err
}

func CurlPostReturnJson(url string, header []string, data string) (r ReturnJson) {
	r = ReturnJson{}
	ret, err := CurlPost(url, header, data)
	if err == nil {
		data, err := json.NewJson(ret)
		if err == nil {
			r.Code, err = data.Get("code").Int()
			r.Message, err = data.Get("message").String()
			r.Data = data.Get("data").Interface()
		}

	}

	return r
}

func CurlPut(url string, header []string, data string) (ret []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, strings.NewReader(data))
	if err != nil {
		return ret, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value; charset=utf-8")
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

func CurlPutReturnJson(url string, header []string, data string) (r ReturnJson) {
	r = ReturnJson{}
	ret, err := CurlPut(url, header, data)
	if err == nil {
		data, err := json.NewJson(ret)
		if err == nil {
			r.Code, err = data.Get("code").Int()
			r.Message, err = data.Get("message").String()
			r.Data = data.Get("data").Interface()
		}
	}

	return r
}

func CurlDelete(url string, header []string) (ret []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
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

func CurlDeleteReturnJson(url string, header []string) (r ReturnJson) {
	r = ReturnJson{}
	ret, err := CurlDelete(url, header)
	if err == nil {
		data, err := json.NewJson(ret)
		if err == nil {
			r.Code, err = data.Get("code").Int()
			r.Message, err = data.Get("message").String()
			r.Data = data.Get("data").Interface()
		}
	}

	return r
}
