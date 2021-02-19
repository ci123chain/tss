package route

import (
	"go-api-frame/controller"
	"go-api-frame/pconst"
	v1 "go-api-frame/route/v1"

	"github.com/gin-gonic/gin"
	"go-api-frame/tgo"
)

//主页
func RouteHome(parentRoute *gin.Engine) {
	parentRoute.GET("", controller.Welcome)
}

func RouteApi(parentRoute *gin.Engine) {
	prefix := tgo.ConfigAppGetString("UrlPrefix", "")
	RouteV1 := parentRoute.Group(prefix + pconst.APIAPIV1URL)
	{
		v1.ApiLogPlatform(RouteV1)
	}
}
