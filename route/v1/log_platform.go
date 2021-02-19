package v1

import (
	"github.com/gin-gonic/gin"
	v1 "go-api-frame/controller/v1"
)

func ApiLogPlatform(parentRoute gin.IRouter) {
	r := parentRoute.Group("log_platform")
	{
		sc := r.Group("sidecar")
		{
			sc.GET("/log_key_list", v1.GetLogPlatformForSideCar)
		}
		r.GET("/info/:id", v1.GetLogPlatformDataInfo)
		r.POST("", v1.AddLogPlatformData)
		r.PUT("", v1.EditLogPlatformData)
		r.DELETE("/:id", v1.DeleteLogPlatformData)
		r.GET("/count", v1.GetLogKeyCount)
	}
}
