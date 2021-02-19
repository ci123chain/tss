package v1

import (
	"github.com/gin-gonic/gin"
	"go-api-frame/controller/v1"
)

func Test(parentRoute *gin.RouterGroup) {
	router := parentRoute.Group("/test")
	router.GET("/test", v1.TestTest)
}
