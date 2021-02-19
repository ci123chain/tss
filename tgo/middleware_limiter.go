package tgo

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gin-gonic/gin"
)

// 按照请求ip限流
func LimitHandlerByIP(lmt *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
		if httpError != nil {
			UtilResponseReturnJson(c, httpError.StatusCode, nil, httpError.Message)
			c.Abort()
		} else {
			c.Next()
		}
	}
}

// 按照key限流
func LimitHandlerByKey(lmt *limiter.Limiter, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var keys []string
		keys = append(keys, key)
		httpError := tollbooth.LimitByKeys(lmt, keys)
		if httpError != nil {
			UtilResponseReturnJson(c, httpError.StatusCode, nil, httpError.Message)
			c.Abort()
		} else {
			c.Next()
		}
	}
}
