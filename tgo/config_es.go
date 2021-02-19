package tgo

import (
	"sync"
)

var (
	esConfigMux sync.Mutex
	esConfig    *ConfigES
)

type ConfigES struct {
	Address             []string
	Timeout             int
	TransportMaxIdel    int
	HealthCheckEnabled  bool
	HealthCheckTimeout  int
	HealthCheckInterval int
	SnifferEnabled      bool
}

func configESInit() {
	if esConfig == nil || len(esConfig.Address) == 0 {
		configFileName := "es"
		if configPathExist(configFileName) {
			esConfigMux.Lock()
			defer esConfigMux.Unlock()
			if esConfig == nil || len(esConfig.Address) == 0 {
				esConfig = &ConfigES{}
				defaultESConfig := configESGetDefault()
				err := configGet(configFileName, esConfig, defaultESConfig)
				if err != nil {
					panic("configESInit error:" + err.Error())
				}
				if esConfig.HealthCheckTimeout == 0 {
					esConfig.HealthCheckTimeout = 1
				}
				if esConfig.HealthCheckInterval == 0 {
					esConfig.HealthCheckInterval = 60
				}
			}
		}
	}
}

func configESClear() {
	esConfigMux.Lock()
	defer esConfigMux.Unlock()
	esConfig = nil
}

func configESGetDefault() *ConfigES {
	return &ConfigES{Address: []string{"url"},
		HealthCheckTimeout:  1,
		HealthCheckInterval: 60,
		Timeout:             3000,
		TransportMaxIdel:    10}
}

func configESGetAddress() []string {
	return esConfig.Address
}

func configESGet() *ConfigES {
	return esConfig
}
