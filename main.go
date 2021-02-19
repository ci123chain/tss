package main

import (
	"github.com/urfave/cli"
	"go-api-frame/script"
	"go-api-frame/server"
	"os"
)

// @title go-api-frame API
// @version 1.0.0
// @description This is go-api-frame api list.
// @host localhost
// @BasePath /api/v1
func main() {
	//ws-wsged1x8c8sp0.dev-sh-001.oneitfarm.com
	app := cli.NewApp()
	app.Name = "go-api-frame"
	app.Usage = "run scripts!"
	app.Version = "1.0.0"
	app.Author = "anonymous"
	app.Commands = script.Commands()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "server",
			Value: "http",
			Usage: "run server type:  http",
		},
		cli.StringFlag{
			Name:  "c",
			Value: "config.yaml",
			Usage: "config file url",
		},
	}
	app.Before = server.InitService
	app.Action = func(c *cli.Context) error {
		println("RunHttp Server.")
		serverType := c.String("server")
		switch serverType {
		case "http":
			server.RunHttp()
		default:
			server.RunHttp()
		}
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		panic("app run error:" + err.Error())
	}
}
