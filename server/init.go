package server

import (
	"fmt"
	"github.com/rubenv/sql-migrate"
	"github.com/urfave/cli"
	"go-api-frame/tgo"
	"runtime"
	"time"
)

func InitService(c *cli.Context) error {
	//环境初始化
	configRuntime()
	//初始化配置文件及内部服务
	tgo.InitConfigAndBase(c.String("c"))
	//初始化外部依赖服务
	tgo.InitOutSideResource()
	//针对线上环境，sql初始化
	if !tgo.ConfigEnvIsDev() {
		sqlMigrate()
	}
	return nil
}

func configRuntime() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	now := time.Now().String()
	fmt.Printf("Running time is %s\n", now)
	fmt.Printf("Running with %d CPUs\n", numCPU)
}

func sqlMigrate() {
	//docker 环境下，根据docker file 配置，sql文件在统计db目录下
	migrations := &migrate.FileMigrationSource{
		Dir: "/root/db",
	}
	Orm, err := tgo.NewDaoMysql().GetOrm()
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "sqlMigrate tgo.NewDaoMysql().GetOrm() err ", err)
		return
	}
	code, err := migrate.Exec(Orm.DB.DB(), "mysql", migrations, migrate.Up)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "sqlMigrate err ", err)
	}
	tgo.LogInfof(tgo.LogNameMysql, "sqlMigrate code is : %d", code)
}
