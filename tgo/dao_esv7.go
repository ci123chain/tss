package tgo

import (
	"context"
	"github.com/olivere/elastic/v7"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type DaoESV7 struct {
	IndexName string
}

var (
	esv7Client    *elastic.Client
	esv7ClientMux sync.Mutex
)

func (dao *DaoESV7) GetIndex() string {
	return globalConfig.Elastic.Index
}

func (dao *DaoESV7) GetConnect() (*elastic.Client, error) {
	if esv7Client == nil {
		esv7ClientMux.Lock()
		defer esv7ClientMux.Unlock()
		clientHttp := &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: globalConfig.Elastic.TransportMaxIdel,
			},
			Timeout: time.Duration(globalConfig.Elastic.Timeout) * time.Millisecond,
		}
		options := []elastic.ClientOptionFunc{
			elastic.SetHttpClient(clientHttp),
			elastic.SetURL(strings.Split(globalConfig.Elastic.Address, ",")...),
			elastic.SetHealthcheckInterval(time.Duration(globalConfig.Elastic.HealthCheckInterval) * time.Second),
			elastic.SetHealthcheckTimeout(time.Duration(globalConfig.Elastic.HealthCheckTimeout) * time.Second),
			elastic.SetSniff(globalConfig.Elastic.SnifferEnabled),
			elastic.SetHealthcheck(globalConfig.Elastic.HealthCheckEnabled),
			elastic.SetBasicAuth(globalConfig.Elastic.Username, globalConfig.Elastic.Password),
		}
		if ConfigEnvIsDev() {
			// 开发环境显示es查询日志
			options = append(options, elastic.SetTraceLog(log.New(os.Stderr, "[[ELASTIC]]", 0)))
		}
		client, err := elastic.NewClient(options...)
		if err != nil {
			LogErrorw(LogNameEs, "es connect error", err)
			return nil, err
		}
		esv7Client = client
	}
	return esv7Client, nil
}

func (dao *DaoESV7) CloseConnect(client *elastic.Client) {
}

func (dao *DaoESV7) Insert(id string, data interface{}) error {
	client, err := dao.GetConnect()
	if err != nil {
		return err
	}
	defer dao.CloseConnect(client)
	ctx := context.Background()
	_, errRes := client.Index().Index(dao.IndexName).Id(id).BodyJson(data).Do(ctx)
	if errRes != nil {
		LogErrorw(LogNameEs, "insert error",
			errRes,
		)
		return errRes
	}
	return nil
}

func (dao *DaoESV7) Update(id string, doc interface{}) error {
	client, err := dao.GetConnect()
	if err != nil {
		return err
	}
	defer dao.CloseConnect(client)
	ctx := context.Background()
	_, errRes := client.Update().Index(dao.IndexName).Id(id).
		Doc(doc).
		Do(ctx)
	if errRes != nil {
		LogErrorw(LogNameEs, "Update DaoESV7 Update error", errRes)
		return errRes
	}
	return nil
}

func (dao *DaoESV7) UpdateAppend(id string, name string, value interface{}) error {
	client, err := dao.GetConnect()
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, errRes := client.Update().Index(dao.IndexName).Id(id).
		Script(elastic.NewScriptStored("append-reply").Param("reply", value)).
		Do(ctx)
	if errRes != nil {
		LogErrorw(LogNameEs, "UpdateAppend DaoESV7 Update error", errRes)
		return err
	}
	return nil
}
