package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-api-frame/tgo"
	"net/http"
	"strings"
)

//跨域
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		var headerKeys []string
		for k, _ := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if tgo.ConfigEnvIsDev() || tgo.ConfigEnvIsBeta() {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Headers", headerStr)
				c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Set("content-type", "application/json")
			}
		}
		if m == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
