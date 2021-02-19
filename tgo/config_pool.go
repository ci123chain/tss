package tgo

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

var (
	poolConfigMux sync.Mutex
	poolConfigMap *ConfigPoolMap
)

type ConfigPoolMap struct {
	Configs map[string]ConfigPool
}

type ConfigPool struct {
	Address            []string
	Lifo               bool //list
	ClientPool         bool
	MaxTotal           int
	MaxIdle            int
	MinIdle            int
	TestOnCreate       bool
	TestOnBorrow       bool
	TestOnReturn       bool
	TestWhileIdle      bool
	BlockWhenExhausted bool
	MaxWaitMillis      int64
	TimeoutConn        int
	TimeoutRead        int
	TimeoutWrite       int
	TransportMaxIdel   int
}

//func GetAddressRandom get one address random
func (c *ConfigPool) GetAddressRandom() (server string, err error) {
	randomMax := len(c.Address)
	if randomMax == 0 {
		err = errors.New("addess is empty")
	} else {
		var randomValue int
		if randomMax > 1 {
			rand.Seed(time.Now().UnixNano())
			randomValue = rand.Intn(randomMax)
		} else {
			randomValue = 0
		}
		server = c.Address[randomValue]
	}
	return
}

func configPoolInit() {
	if poolConfigMap == nil {
		configFileName := "pool"
		if configPathExist(configFileName) {
			poolConfigMux.Lock()
			defer poolConfigMux.Unlock()
			if poolConfigMap == nil {
				poolConfigMap = &ConfigPoolMap{Configs: make(map[string]ConfigPool)}
			}
			defaultPoolConfig := configPoolGetDefault()
			err := configGet(configFileName, poolConfigMap, defaultPoolConfig)
			if err != nil {
				panic("configPoolInit error:" + err.Error())
			}
		}
	}
}

func configPoolGetDefault() *ConfigPoolMap {
	cps := &ConfigPoolMap{Configs: make(map[string]ConfigPool)}
	cps.Configs["test"] = ConfigPool{
		Address:          []string{"url"},
		ClientPool:       true,
		MaxTotal:         20,
		MinIdle:          2,
		MaxIdle:          20,
		TimeoutConn:      3000,
		TransportMaxIdel: 10}
	return cps
}

func configPoolGet(poolName string) *ConfigPool {
	poolConfig, ok := poolConfigMap.Configs[poolName]
	if !ok {
		return nil
	}
	return &poolConfig
}
