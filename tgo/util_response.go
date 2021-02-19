package tgo

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UtilResponseReturnJsonNoP(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(msg...), model, false, true)
}

func UtilResponseReturnJson(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(msg...), model, true, true)
}

func UtilResponseReturnJsonNoPReal(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(msg...), model, false, false)
}

func UtilResponseReturnJsonReal(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(msg...), model, true, false)
}

func getResponseMsg(msg ...string) (message string) {
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	return
}

func UtilResponseReturnJsonWithMsg(c *gin.Context, code int, msg string, model interface{},
	callbackFlag bool, unifyCode bool) {
	if unifyCode && code == 0 {
		code = 1001
	}
	if msg == "" {
		msg = ConfigCodeGetMessage(code)
	}
	var rj interface{}
	//添加结果
	if _, ok := codeFalseMap[code]; !ok {
		c.Set("result", true)
	} else {
		c.Set("result", false)
	}
	rj = gin.H{
		"code":    code,
		"message": msg,
		"data":    model,
	}

	var callback string
	if callbackFlag {
		callback = c.Query("callback")
	}

	if UtilIsEmpty(callback) {
		c.JSON(http.StatusOK, rj)
	} else {
		r, err := json.Marshal(rj)
		if err != nil {
			LogErrorw(LogNameLogic, "UtilResponseReturnJsonWithMsg json Marshal error", err)
		} else {
			c.String(http.StatusOK, "%s(%s)", callback, r)
		}
	}
}

func UtilResponseReturnJsonFailed(c *gin.Context, code int) {
	UtilResponseReturnJson(c, code, nil)
}

func UtilResponseReturnJsonSuccess(c *gin.Context, data interface{}) {
	UtilResponseReturnJson(c, 0, data)
}

func UtilResponseRedirect(c *gin.Context, url string) {
	c.Redirect(http.StatusMovedPermanently, url)
}

func utilResponseJSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
