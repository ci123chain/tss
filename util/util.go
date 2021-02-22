package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-api-frame/pconst"
	"go-api-frame/tgo"
	"io/ioutil"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

func GetInfluxDBRightInterval(startTime, endTime int64) string {
	//根据官方的Graf，900秒的跨度对应2500ms，我们根据该比例计算出合适的interval
	betweenTime := endTime - startTime   // 单位为纳秒
	betweenTime = betweenTime / 1e9      // 转为秒
	interval := betweenTime * 2500 / 900 // 当前跨度内对应的interval毫秒
	intervalStr := fmt.Sprintf("%sms", strconv.Itoa(int(interval)))
	return intervalStr
}

func GetPrometheusRightInterval(startTime, endTime int64) time.Duration {
	//根据官方的Graf，900秒的跨度对应2500ms，我们根据该比例计算出合适的interval
	betweenTime := endTime - startTime   // 单位为秒
	interval := betweenTime * 2500 / 900 // 当前跨度内对应的interval毫秒
	intervalTime := time.Duration(interval) * time.Millisecond
	if intervalTime < 10*time.Second {
		intervalTime = 10 * time.Second
	}
	return intervalTime
}

func GetPrometheusSumInterval(startTime, endTime int64) (interval time.Duration, timeUnit string) {
	betweenTime := endTime - startTime // 单位为秒
	//经过分析，尽量保证点的数量小于等于60
	if betweenTime/pconst.TIME_ONE_SECOND <= 60 {
		interval = time.Second
		timeUnit = "s"
		return
	}
	if betweenTime/pconst.TIME_ONE_MINUTE <= 60 {
		interval = time.Minute
		timeUnit = "m"
		return
	}
	if betweenTime/pconst.TIME_ONE_HOUR <= 60 {
		interval = time.Hour
		timeUnit = "h"
		return
	}
	interval = time.Hour * 24
	timeUnit = "d"
	return
}

func GetInfluxDBSumInterval(startTime, endTime int64) (interval, timeUnit string) {
	//为方便统计，我们将单位全部换算成秒
	betweenTime := endTime - startTime // 单位为纳秒
	betweenTime = betweenTime / 1e9    // 转为秒
	//经过分析，尽量保证点的数量小于等于60
	if betweenTime/pconst.TIME_ONE_SECOND <= 60 {
		interval = "1s"
		timeUnit = "s"
		return
	}
	if betweenTime/pconst.TIME_ONE_MINUTE <= 60 {
		interval = "1m"
		timeUnit = "m"
		return
	}
	if betweenTime/pconst.TIME_ONE_HOUR <= 60 {
		interval = "1h"
		timeUnit = "h"
		return
	}
	interval = "1d"
	timeUnit = "d"
	return
}

func GetESRightInterval(startTime, endTime int64) string {
	//我们根据该比例计算出合适的interval
	betweenTime := endTime - startTime // 单位为秒
	interval := betweenTime / 100      //
	//最小为1秒，其余向5的整数靠齐
	interval = interval / 5
	if interval < 1 {
		interval = 1
	} else {
		interval = interval * 5
	}
	intervalStr := fmt.Sprintf("%ss", strconv.Itoa(int(interval)))
	return intervalStr
}

func GetESSumInterval(startTime, endTime int64) (interval, timeUnit string) {
	betweenTime := endTime - startTime // 单位为秒
	//经过分析，尽量保证点的数量小于等于60
	if betweenTime/pconst.TIME_ONE_SECOND <= 60 {
		interval = "1s"
		timeUnit = "s"
		return
	}
	if betweenTime/pconst.TIME_ONE_MINUTE <= 60 {
		interval = "1m"
		timeUnit = "m"
		return
	}
	if betweenTime/pconst.TIME_ONE_HOUR <= 60 {
		interval = "1h"
		timeUnit = "h"
		return
	}
	interval = "1d"
	timeUnit = "d"
	return
}

func FloatToString(Num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(Num, 'f', 2, 64)
}

func MD5Bytes(s []byte) string {
	ret := md5.Sum(s)
	return hex.EncodeToString(ret[:])
}

//计算字符串MD5值
func MD5(s string) string {
	return MD5Bytes([]byte(s))
}

//计算文件MD5值
func MD5File(file string) (string, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return MD5Bytes(data), nil
}

func ToInterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}

func StructToMap(item interface{}) (data map[string]interface{}, err error) {
	jsonByte, err := json.Marshal(item)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameDefault, "StructToMap err", err)
		return
	}
	err = json.Unmarshal(jsonByte, &data)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameDefault, "StructToMap err", err)
		return
	}
	return
}

func HandelParamValues(param interface{}) (params url.Values, err error) {
	paramMap, err := StructToMap(param)
	if err != nil {
		return
	}
	params = url.Values{}
	for key, value := range paramMap {
		if value == 0 || value == "" || value == nil {
			continue
		}
		typeStr := reflect.TypeOf(value).String()
		if typeStr == "float64" {
			valueStr := strconv.FormatFloat(value.(float64), 'f', -1, 64)
			params.Add(key, valueStr)
		} else if typeStr == "string" {
			params.Add(key, value.(string))
		} else if typeStr == "[]interface {}" {
			for _, em := range value.([]interface{}) {
				switch v := em.(type) {
				case string:
					params.Add(key, v)
				case float64:
					valueStr := strconv.FormatFloat(value.(float64), 'f', -1, 64)
					params.Add(key, valueStr)
				}
			}
		}
	}
	return
}

func RemoveRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

// Convert json string to map
func JsonToMap(jsonStr string) (map[string]string, error) {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
