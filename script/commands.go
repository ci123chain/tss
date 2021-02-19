package script

import (
	"github.com/urfave/cli"
	"go-api-frame/script/logic"
)

func Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "sync_data",
			Usage: "同步注册中心数据、接收注册中心资源信息、接收熔断记录",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "role",
					Value: "consumer",
					Usage: "生产还是消费",
				},
			},
			Action: func(c *cli.Context) {
				logic.CatchCmdSignals()
				return
			},
		},
	}
}
