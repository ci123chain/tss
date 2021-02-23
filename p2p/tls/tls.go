package tls

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"
)
const (
	bufferSize           = 1024
	connSpanTimeOut      = 1200 //max 20 mintue tcp alive tunnel handle
)

func NewTLS() {
	host := config.BindAddressIP
	port := config.BindAddressPort

	localSockURI := fmt.Sprintf("%s:%s", host, strconv.FormatUint(uint64(port), 10))
	loger.Info("local socket uri", "uri", localSockURI)
	local, err := net.Listen("tcp", localSockURI)
	if err != nil {
		loger.Error("listen error", "listen uri", localSockURI)
		panic(err)
	}
	//defer local.Close()

	go func() {
		for {
			client, err := local.Accept()
			if err != nil {
				continue
			}else {
				var remote = TrafficRemoteS{
					RemoteAddressHOST:           config.RemoteAddressHOST,
					RemoteAddressPort:           config.RemoteAddressPort,
					RemoteServerName:            config.RemoteServerName,
					RemoteTLSCertURI:            config.RemoteTLSCertURI,
					RemoteTLSCertKeyURI:         config.RemoteTLSCertKeyURI,
					RemoteTLSDialTimeout:        config.RemoteTLSDialTimeout,
					RemoteTLSInsecureSkipVerify: config.RemoteTLSInsecureSkipVerify,
				}
				go handleLocalClient(client, &remote)
			}
		}
	}()

	//for {
	//
	//	client, err := local.Accept()
	//	if err != nil {
	//		loger.Error("accept connect failed", "err", err.Error())
	//		continue
	//	}
	//
	//	go handleLocalClient(client, &config.TrafficRemote)
	//}
}

func handleLocalClient(client net.Conn, remoteOpts *TrafficRemoteS) {
	//defer func() {
	//	_ = client.Close()
	//}()

	handleRemote(client, remoteOpts)
}

func handleRemote(client net.Conn, remoteOpts *TrafficRemoteS) {
	defer client.Close()

	var cert tls.Certificate
	var err error
	//对于从ssl供应商获取的证书，可以不提供秘钥，前提是系统支持ROOT CA
	if len(remoteOpts.RemoteTLSCertURI) != 0 && len(remoteOpts.RemoteTLSCertKeyURI) != 0 {
		cert, err = tls.LoadX509KeyPair(remoteOpts.RemoteTLSCertURI, remoteOpts.RemoteTLSCertKeyURI)
		if err != nil {
			//log.Printf("Initialize the certificate, the key failed %s", err.Error())
			loger.Error("Initialize the certificate, the key failed", "err", err.Error())
			return
		}
	}

	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         remoteOpts.RemoteServerName,
		InsecureSkipVerify: remoteOpts.RemoteTLSInsecureSkipVerify,
	}

	remoteServer := fmt.Sprintf("%s:%s", remoteOpts.RemoteAddressHOST, strconv.FormatUint(uint64(remoteOpts.RemoteAddressPort), 10))
	//remoteServer := fmt.Sprintf("%s:%s", "test-chain-tls-server.gw001.oneitfarm.com", strconv.FormatUint(uint64(remoteOpts.RemoteAddressPort), 10))
	//remoteServer := fmt.Sprintf("%s:%s", remoteOpts.RemoteAddressHOST, strconv.FormatUint(uint64(9000), 10))
	//loger.Info("connect remote", "remote uri", remoteServer)

	remote, dial_err := tls.DialWithDialer(&net.Dialer{
		Timeout: time.Second * time.Duration(remoteOpts.RemoteTLSDialTimeout),
	}, "tcp", remoteServer, &config)

	if dial_err != nil {
		loger.Error("connect remote failed, error", "err", dial_err.Error())
		return
	}

	defer remote.Close()

	remoteReader := bufio.NewReader(remote)
	remoteWriter := bufio.NewWriter(remote)

	chClient := make(chan struct{}) //client 信号
	chRemote := make(chan struct{}) //remote 信号

	clientReader := bufio.NewReader(client)
	clientWriter := bufio.NewWriter(client)

	//从client端读取数据
	go readDataFromClientBufIO(chClient, clientReader, remoteWriter)

	//从remote端读取数据
	go readDataFromRemoteBufIO(chRemote, remoteReader, clientWriter)

	for {
		select {
		case _, ok := <-chRemote:
			if !ok {
				return
			}
		case _, ok := <-chClient:
			if !ok {
				return
			}
		case <-time.After(time.Second * connSpanTimeOut): //连接最大保活时间
			//log.Printf("close conn by timer")
			loger.Info("close conn by timer")
			return
		}
	}
}

func readDataFromClientBufIO(ch chan struct{}, localReader *bufio.Reader, remoteWriter *bufio.Writer) {

	//bufferData := make([]byte, bufferSize)

	for {
		bufferData := make([]byte, bufferSize)
		n, err := localReader.Read(bufferData)
		if err != nil {
			break
		}

		_, err = remoteWriter.Write(bufferData[:n])

		if err != nil {
			loger.Error("remote write failed", "err", err.Error())
			break
		}

		//性能与安全优化：当读取缓冲区已经为空时，进行flush 避免flush造成的性能损耗和网络通信特征
		if localReader.Buffered() <= 0 {
			err = remoteWriter.Flush()

			if err != nil {
				break
			}
		}
	}

	close(ch)
	return
}

func readDataFromRemoteBufIO(ch chan struct{}, remoteReader *bufio.Reader, localWriter *bufio.Writer) {
	//提前开辟内存 在通道维持期间 不需要频繁开辟，避免性能损耗 并 复用内存
	//bufferData := make([]byte, bufferSize)

	for {
		bufferData := make([]byte, bufferSize)
		n, err := remoteReader.Read(bufferData)
		if err != nil {
			break
		}

		n, err = localWriter.Write(bufferData[:n])

		if err != nil {
			loger.Error("local write failed", "err", err.Error())
			break
		}

		//性能与安全优化：当读取缓冲区已经为空时，进行flush 避免flush造成的性能损耗和网络通信特征
		if remoteReader.Buffered() <= 0 {
			err = localWriter.Flush()

			if err != nil {
				break
			}
		}
	}

	close(ch)
	return

}
