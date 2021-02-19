package tgo

import (
	"strconv"
	"sync"
)

var (
	codeConfig sync.Map
)

type ConfigCodeList struct {
	Codes map[string]string
}

func configCodeInit() {
	//defaultData := &ConfigCodeList{Codes: map[string]string{"0": "success"}}
	//configFileName := "code"
	//if configPathExist(configFileName) {
	//	fileCodeConfig := new(ConfigCodeList)
	//	err := configGet(configFileName, fileCodeConfig, defaultData)
	//	if err != nil {
	//		panic("configCodeInit error:" + err.Error())
	//	}
	//	for k, v := range fileCodeConfig.Codes {
	//		codeConfig.Store(k, v)
	//	}
	//}
	for k, v := range globalConfig.Code {
		codeConfig.Store(k, v)
	}
}

func configCodeClear() {
	codeConfig = sync.Map{}
}

func ConfigCodeGetMessage(code int) string {
	msg, exists := codeConfig.Load(strconv.Itoa(code))
	if !exists {
		return "system error"
	}
	return msg.(string)
}

var (
	//监控false的code map
	codeFalseMap = map[int]struct{}{
		PCONST_CODE_SERVER_BUSY: {},
	}
)
