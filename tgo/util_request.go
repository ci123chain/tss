package tgo

import (
	"github.com/gin-gonic/gin"
	"net/url"
)

func UtilRequestGetParam(c *gin.Context, key string) string {
	if c.Request.Method == "GET" || c.Request.Method == "DELETE" {
		return c.Query(key)
	}
	return c.PostForm(key)
}

func UtilRequestGetAllParams(c *gin.Context) (ret url.Values) {
	switch c.Request.Method {
	case "GET":
		fallthrough
	case "DELETE":
		ret = c.Request.URL.Query()
	case "POST":
		fallthrough
	case "PATCH":
		fallthrough
	case "PUT":
		c.Request.ParseForm()
		ret = c.Request.PostForm
	}
	return ret
}

func UtilRequestQueryDataString(c *gin.Context) string {
	var query url.Values
	query = UtilRequestGetAllParams(c)

	return query.Encode()
}
