package tls

import (
	"github.com/tendermint/tendermint/libs/log"
	"sync"

	//"sync"
)

var (
	config *TLSConfig
	loger log.Logger
)

//TrafficLocalS is configure of local network options
type TrafficLocalS struct {
	BindAddressIP   string `mapstructure:"bind_address_ip"`
	BindAddressPort int    `mapstructure:"bind_address_port"`
}

//TrafficRemoteS is configure of remote tls options
type TrafficRemoteS struct {
	RemoteAddressHOST           string `mapstructure:"remote_address_host"` ///remote node host.
	RemoteAddressPort           int    `mapstructure:"remote_address_port"` ///gateway port.
	RemoteServerName            string `mapstructure:"remote_server_name"`  ///remote node name.
	RemoteTLSCertURI            string `mapstructure:"remote_tls_cert"`
	RemoteTLSCertKeyURI         string `mapstructure:"remote_tls_cert_key"`
	RemoteTLSDialTimeout        int    `mapstructure:"remote_dial_timeout"`
	RemoteTLSInsecureSkipVerify bool   `mapstructure:"remote_insecure_skip_verify"` ///true means close verify.
}

type TLSConfig struct {
	Mutex           sync.Mutex
	BindAddressIP   string `mapstructure:"bind_address_ip"`
	BindAddressPort int    `mapstructure:"bind_address_port"`
	RemoteAddressHOST           string `mapstructure:"remote_address_host"` ///remote node host.
	RemoteAddressPort           int    `mapstructure:"remote_address_port"` ///gateway port.
	RemoteServerName            string `mapstructure:"remote_server_name"`  ///remote node name.
	RemoteTLSCertURI            string `mapstructure:"remote_tls_cert"`
	RemoteTLSCertKeyURI         string `mapstructure:"remote_tls_cert_key"`
	RemoteTLSDialTimeout        int    `mapstructure:"remote_tls_dial_timeout"`
	RemoteTLSInsecureSkipVerify bool   `mapstructure:"remote_tls_insecure_skip_verify"` ///true means close verify.
}

func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		BindAddressIP:   "127.0.0.1",
		BindAddressPort: 9001,
		RemoteAddressHOST:           "",
		RemoteAddressPort:           7443,
		RemoteServerName:            "",
		RemoteTLSCertURI:            "",
		RemoteTLSCertKeyURI:         "",
		RemoteTLSDialTimeout:        5,
		RemoteTLSInsecureSkipVerify: true,
	}
}

func SetLogger(log log.Logger) {
	loger = log
}

func SetTLSConfig(conf *TLSConfig) {
	config = conf
}

func SetRemoteNOdeAddress(remoteHost string, port int) {
	config.RemoteAddressHOST = remoteHost
	config.RemoteAddressPort = port
}
