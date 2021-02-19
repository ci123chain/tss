package server

import (
	"fmt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "go-api-frame/docs"
	"go-api-frame/middleware"
	"go-api-frame/route"
	"go-api-frame/tgo"
	"go-api-frame/util/ip"
	"strconv"
)

var httpPort int

func RunHttp() {
	r := tgo.NewGin()
	r.Use(middleware.Cors())
	httpPort = tgo.ConfigAppGetInt("port", 80)
	portStr := ":" + strconv.Itoa(httpPort)
	if tgo.ConfigEnvIsDev() {
		ipAddress := "127.0.0.1"
		//获取本机IP地址
		ipArr, err := ip.LocalIPv4s()
		if err == nil && len(ipArr) > 0 {
			ipAddress = ipArr[0]
		}
		url := ginSwagger.URL("http://" + ipAddress + "" + portStr + "/swagger/doc.json") // The url pointing to API definition
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	}
	route.RouteHome(r)
	route.RouteApi(r)
	fmt.Println("start", httpPort)
	tgo.ListenHttp(portStr, r, 10)
}
