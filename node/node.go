package node

import (
	"CipherMachine/tsslib/common"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/privval"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	grpccore "github.com/tendermint/tendermint/rpc/grpc"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
	"math/big"
	"net"
	"net/http"
	"os"
	cfg "CipherMachine/config"
	"CipherMachine/p2p"
	"CipherMachine/p2p/pex"
	"CipherMachine/p2p/tls"
	"CipherMachine/threshold"
	rpccore "CipherMachine/rpc/core"
	"strings"
	"time"
)

type Node struct {
	cmn.BaseService

	// config
	config        *cfg.Config

	// network
	transport   *p2p.MultiplexTransport
	sw          *p2p.Switch  // p2p connections
	addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	// services
	storeDB			 dbm.DB
	pexReactor       *pex.PEXReactor        // for exchanging peer addresses
	rpcListeners     []net.Listener         // rpc servers
}

// DBContext specifies config information for loading a new DB.
type DBContext struct {
	ID     string
	Config *cfg.Config
}

// DBProvider takes a DBContext and returns an instantiated DB.
type DBProvider func(*DBContext) (dbm.DB, error)

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.DBBackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir()), nil
}

func DefaultNewNode(config *cfg.Config, logger log.Logger) (*Node, error) {
	// Generate node PrivKey
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, err
	}

	// Convert old PrivValidator if it exists.
	oldPrivVal := config.OldPrivValidatorFile()
	newPrivValKey := config.PrivValidatorKeyFile()
	newPrivValState := config.PrivValidatorStateFile()
	if _, err := os.Stat(oldPrivVal); !os.IsNotExist(err) {
		oldPV, err := privval.LoadOldFilePV(oldPrivVal)
		if err != nil {
			return nil, fmt.Errorf("error reading OldPrivValidator from %v: %v\n", oldPrivVal, err)
		}
		logger.Info("Upgrading PrivValidator file",
			"old", oldPrivVal,
			"newKey", newPrivValKey,
			"newState", newPrivValState,
		)
		oldPV.Upgrade(newPrivValKey, newPrivValState)
	}

	return NewNode(config,
		privval.LoadOrGenFilePV(newPrivValKey, newPrivValState),
		nodeKey,
		DefaultDBProvider,
		logger,
	)
}

// NewNode returns a new, ready to go, Tendermint Node.
func NewNode(config *cfg.Config,
	privValidator types.PrivValidator,
	nodeKey *p2p.NodeKey,
	dbProvider DBProvider,
	logger log.Logger) (*Node, error) {

	storeDB, err := initDB(config, dbProvider)
	if err != nil {
		return nil, err
	}

	// If an address is provided, listen on the socket for a connection from an
	// external signing process.
	if config.PrivValidatorListenAddr != "" {
		// FIXME: we should start services inside OnStart
		privValidator, err = createAndStartPrivValidatorSocketClient(config.PrivValidatorListenAddr, logger)
		if err != nil {
			return nil, errors.Wrap(err, "error with private validator socket client")
		}
	}

	pubKey := privValidator.GetPubKey()
	if pubKey == nil {
		// TODO: GetPubKey should return errors - https://github.com/tendermint/tendermint/issues/3602
		return nil, errors.New("could not retrieve public key from private validator")
	}

	logNodeStartupInfo(pubKey, logger)

	nodeInfo, err := makeNodeInfo(config, nodeKey)
	if err != nil {
		return nil, err
	}

	// Setup Transport.
	transport, peerFilters := createTransport(config, nodeInfo, nodeKey)

	// Setup Switch.
	p2pLogger := logger.With("module", "p2p")
	sw := createSwitch(
		config, transport, peerFilters, nodeInfo, nodeKey, p2pLogger,
	)

	err = sw.AddPersistentPeers(splitAndTrimEmpty(config.P2P.PersistentPeers, ",", " "))
	if err != nil {
		return nil, errors.Wrap(err, "could not add peers from persistent_peers field")
	}

	addrBook, err := createAddrBookAndSetOnSwitch(config, sw, p2pLogger, nodeKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not create addrbook")
	}

	// Optionally, start the pex reactor
	//
	// TODO:
	//
	// We need to set Seeds and PersistentPeers on the switch,
	// since it needs to be able to use these (and their DNS names)
	// even if the PEX is off. We can include the DNS name in the NetAddress,
	// but it would still be nice to have a clear list of the current "PersistentPeers"
	// somewhere that we can return with net_info.
	//
	// If PEX is on, it should handle dialing the seeds. Otherwise the switch does it.
	// Note we currently use the addrBook regardless at least for AddOurAddress
	var pexReactor *pex.PEXReactor
	if config.P2P.PexReactor {
		pexReactor = createPEXReactorAndAddToSwitch(addrBook, config, sw, logger)
	}
	if config.ProfListenAddress != "" {
		go func() {
			logger.Error("Profile server", "err", http.ListenAndServe(config.ProfListenAddress, nil))
		}()
	}

	createTssReactorAndAdd(config, sw, logger, storeDB, string(nodeKey.ID()))

	node := &Node{
		config:        	config,
		transport: 		transport,
		sw:        		sw,
		addrBook:  		addrBook,
		nodeInfo:  		nodeInfo,
		nodeKey:   	    nodeKey,
		storeDB: 		storeDB,
		pexReactor:     pexReactor,
	}
	node.BaseService = *cmn.NewBaseService(logger, "Node", node)

	return node, nil
}

func (n *Node) OnStart() error {
	n.StartTLS()
	// Add private IDs to addrbook to block those peers being added
	n.addrBook.AddPrivateIDs(splitAndTrimEmpty(n.config.P2P.PrivatePeerIDs, ",", " "))

	// Start the RPC server before the P2P server
	// so we can eg. receive txs for the first block
	if n.config.RPC.ListenAddress != "" {
		listeners, err := n.startRPC()
		if err != nil {
			return err
		}
		n.rpcListeners = listeners
	}

	// Start the transport.
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(n.nodeKey.ID(), n.config.P2P.ListenAddress))
	if err != nil {
		return err
	}
	if err := n.transport.Listen(*addr); err != nil {
		return err
	}

	n.isListening = true

	// Start the switch (the P2P server).
	err = n.sw.Start()
	if err != nil {
		return err
	}

	// Always connect to persistent peers
	err = n.sw.DialPeersAsync(splitAndTrimEmpty(n.config.P2P.PersistentPeers, ",", " "))
	if err != nil {
		return errors.Wrap(err, "could not dial peers from persistent_peers field")
	}

	return nil
}

func(n *Node) StartTLS() {
	var newCfg = n.Config()
	p2p.SetP2PConfig(newCfg)
	p2p.SetLogger(n.Logger)
	tls.SetTLSConfig(newCfg.TLSConfig)
	tls.SetLogger(n.Logger)
	if newCfg.P2P.TLSOption {
		tls.NewTLS()
		n.Logger.Info("Started TLS")
	}
}

// Config returns the Node's config.
func (n *Node) Config() *cfg.Config {
	return n.config
}

func (n *Node) startRPC() ([]net.Listener, error) {
	n.ConfigureRPC()
	listenAddrs := splitAndTrimEmpty(n.config.RPC.ListenAddress, ",", " ")
	coreCodec := amino.NewCodec()
	ctypes.RegisterAmino(coreCodec)

	if n.config.RPC.Unsafe {
		rpccore.AddUnsafeRoutes()
	}

	config := rpcserver.DefaultConfig()
	config.MaxBodyBytes = n.config.RPC.MaxBodyBytes
	config.MaxHeaderBytes = n.config.RPC.MaxHeaderBytes
	config.MaxOpenConnections = n.config.RPC.MaxOpenConnections
	// If necessary adjust global WriteTimeout to ensure it's greater than
	// TimeoutBroadcastTxCommit.
	// See https://github.com/tendermint/tendermint/issues/3435
	if config.WriteTimeout <= n.config.RPC.TimeoutBroadcastTxCommit {
		config.WriteTimeout = n.config.RPC.TimeoutBroadcastTxCommit + 1*time.Second
	}

	// we may expose the rpc over both a unix and tcp socket
	listeners := make([]net.Listener, len(listenAddrs))
	for i, listenAddr := range listenAddrs {
		mux := http.NewServeMux()
		rpcLogger := n.Logger.With("module", "rpc-server")
		wmLogger := rpcLogger.With("protocol", "websocket")
		wm := rpcserver.NewWebsocketManager(rpccore.Routes, coreCodec,
			rpcserver.OnDisconnect(func(remoteAddr string) {
			}),
			rpcserver.ReadLimit(config.MaxBodyBytes),
		)
		wm.SetLogger(wmLogger)
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		rpcserver.RegisterRPCFuncs(mux, rpccore.Routes, coreCodec, rpcLogger)
		listener, err := rpcserver.Listen(
			listenAddr,
			config,
		)
		if err != nil {
			return nil, err
		}

		var rootHandler http.Handler = mux
		if n.config.RPC.IsCorsEnabled() {
			corsMiddleware := cors.New(cors.Options{
				AllowedOrigins: n.config.RPC.CORSAllowedOrigins,
				AllowedMethods: n.config.RPC.CORSAllowedMethods,
				AllowedHeaders: n.config.RPC.CORSAllowedHeaders,
			})
			rootHandler = corsMiddleware.Handler(mux)
		}
		if n.config.RPC.IsTLSEnabled() {
			go rpcserver.StartHTTPAndTLSServer(
				listener,
				rootHandler,
				n.config.RPC.CertFile(),
				n.config.RPC.KeyFile(),
				rpcLogger,
				config,
			)
		} else {
			go rpcserver.StartHTTPServer(
				listener,
				rootHandler,
				rpcLogger,
				config,
			)
		}

		listeners[i] = listener
	}

	// we expose a simplified api over grpc for convenience to app devs
	grpcListenAddr := n.config.RPC.GRPCListenAddress
	if grpcListenAddr != "" {
		config := rpcserver.DefaultConfig()
		config.MaxOpenConnections = n.config.RPC.MaxOpenConnections
		listener, err := rpcserver.Listen(grpcListenAddr, config)
		if err != nil {
			return nil, err
		}
		go grpccore.StartGRPCServer(listener)
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// ConfigureRPC sets all variables in rpccore so they will serve
// rpc calls from this node
func (n *Node) ConfigureRPC() {
	rpccore.SetP2PPeers(n.sw)
	rpccore.SetP2PTransport(n)
	rpccore.SetAddrBook(n.addrBook)
	rpccore.SetLogger(n.Logger.With("module", "rpc"))
	rpccore.SetConfig(*n.config.RPC)
}

func (n *Node) Listeners() []string {
	return []string{
		fmt.Sprintf("Listener(@%v)", n.config.P2P.ExternalAddress),
	}
}

func (n *Node) IsListening() bool {
	return n.isListening
}

// NodeInfo returns the Node's Info from the Switch.
func (n *Node) NodeInfo() p2p.NodeInfo {
	return n.nodeInfo
}

// Keygen exported, used in client
func (n *Node) Keygen(shares int, sid threshold.SessionID) chan struct{}{
	return n.sw.Reactor("tss").(*threshold.TssReactor).Keygen(shares, sid)
}

// Signing exported, used in client
func (n *Node) Signing(msg *big.Int, sid threshold.SessionID) (chan common.SignatureData, error) {
	resCh, err := n.sw.Reactor("tss").(*threshold.TssReactor).Signing(msg, sid)
	if err != nil {
		return nil, err
	}
	return resCh, nil
}

// Verify exported, used in client
func (n *Node) Verify(msg *big.Int, sid threshold.SessionID, signature common.SignatureData) error {
	if err := n.sw.Reactor("tss").(*threshold.TssReactor).Verify(msg, sid, signature); err != nil {
		return err
	}
	return nil
}

// Resharing exported, used in client
func (n *Node) Resharing() {
	return
}

func initDB(config *cfg.Config, dbProvider DBProvider) (storeDB dbm.DB, err error) {
	storeDB, err = dbProvider(&DBContext{"store", config})
	if err != nil {
		return
	}

	return storeDB, nil
}

func createAndStartPrivValidatorSocketClient(
	listenAddr string,
	logger log.Logger,
) (types.PrivValidator, error) {
	pve, err := privval.NewSignerListener(listenAddr, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start private validator")
	}

	pvsc, err := privval.NewSignerClient(pve)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start private validator")
	}

	return pvsc, nil
}

func createPEXReactorAndAddToSwitch(addrBook pex.AddrBook, config *cfg.Config,
	sw *p2p.Switch, logger log.Logger) *pex.PEXReactor {

	// TODO persistent peers ? so we can have their DNS addrs saved
	pexReactor := pex.NewPEXReactor(addrBook,
		&pex.PEXReactorConfig{
			Seeds:    splitAndTrimEmpty(config.P2P.Seeds, ",", " "),
			SeedMode: config.P2P.SeedMode,
			// See consensus/reactor.go: blocksToContributeToBecomeGoodPeer 10000
			// blocks assuming 10s blocks ~ 28 hours.
			// TODO (melekes): make it dynamic based on the actual block latencies
			// from the live network.
			// https://github.com/tendermint/tendermint/issues/3523
			SeedDisconnectWaitPeriod: 28 * time.Hour,
		})
	pexReactor.SetLogger(logger.With("module", "pex"))
	sw.AddReactor("PEX", pexReactor)
	return pexReactor
}

func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))
	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}
	return nonEmptyStrings
}

func logNodeStartupInfo(pubKey crypto.PubKey, logger log.Logger) {
	// Log the version info.
	logger.Info("Version info",
		"software", version.TMCoreSemVer,
		"address", pubKey.Address().String(),
	)
}

func makeNodeInfo(
	config *cfg.Config,
	nodeKey *p2p.NodeKey,
) (p2p.NodeInfo, error) {
	nodeInfo := p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.NewProtocolVersion(
			version.P2PProtocol, // global
		),
		ID_:     nodeKey.ID(),
		Network: config.ChainID(),
		Version: version.TMCoreSemVer,
		Moniker: config.Moniker,
		Other: p2p.DefaultNodeInfoOther{
			RPCAddress: config.RPC.ListenAddress,
		},
	}

	if config.P2P.PexReactor {
		nodeInfo.Channels = append(nodeInfo.Channels, pex.PexChannel, threshold.TssChannel)
	}

	lAddr := config.P2P.ExternalAddress

	if lAddr == "" {
		lAddr = config.P2P.ListenAddress
	}

	nodeInfo.ListenAddr = lAddr

	err := nodeInfo.Validate()
	return nodeInfo, err
}

func createTransport(config *cfg.Config, nodeInfo p2p.NodeInfo, nodeKey *p2p.NodeKey) (*p2p.MultiplexTransport, []p2p.PeerFilterFunc) {
	var (
		mConnConfig = p2p.MConnConfig(config.P2P)
		transport   = p2p.NewMultiplexTransport(nodeInfo, *nodeKey, mConnConfig)
		connFilters = []p2p.ConnFilterFunc{}
		peerFilters = []p2p.PeerFilterFunc{}
	)

	if !config.P2P.AllowDuplicateIP {
		connFilters = append(connFilters, p2p.ConnDuplicateIPFilter())
	}

	// Filter peers by addr or pubkey with an ABCI query.
	// If the query return code is OK, add peer.
	if config.FilterPeers {
		connFilters = append(
			connFilters,
			// ABCI query for address filtering.
			func(_ p2p.ConnSet, c net.Conn, _ []net.IP) error {
				return nil
			},
		)

		peerFilters = append(
			peerFilters,
			// ABCI query for ID filtering.
			func(_ p2p.IPeerSet, p p2p.Peer) error {
				return nil
			},
		)
	}

	p2p.MultiplexTransportConnFilters(connFilters...)(transport)
	return transport, peerFilters
}

func createSwitch(config *cfg.Config,
	transport *p2p.MultiplexTransport,
	peerFilters []p2p.PeerFilterFunc,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	p2pLogger log.Logger) *p2p.Switch {

	sw := p2p.NewSwitch(
		config.P2P,
		transport,
		p2p.SwitchPeerFilters(peerFilters...),
	)
	sw.SetLogger(p2pLogger)
	//sw.AddReactor("MEMPOOL", mempoolReactor)
	//sw.AddReactor("BLOCKCHAIN", bcReactor)
	//sw.AddReactor("CONSENSUS", consensusReactor)
	//sw.AddReactor("EVIDENCE", evidenceReactor)

	sw.SetNodeInfo(nodeInfo)
	sw.SetNodeKey(nodeKey)

	p2pLogger.Info("P2P Node ID", "ID", nodeKey.ID(), "file", config.NodeKeyFile())
	return sw
}

func createAddrBookAndSetOnSwitch(config *cfg.Config, sw *p2p.Switch,
	p2pLogger log.Logger, nodeKey *p2p.NodeKey) (pex.AddrBook, error) {

	addrBook := pex.NewAddrBook(config.P2P.AddrBookFile(), config.P2P.AddrBookStrict)
	addrBook.SetLogger(p2pLogger.With("book", config.P2P.AddrBookFile()))

	// Add ourselves to addrbook to prevent dialing ourselves
	if config.P2P.ExternalAddress != "" {
		addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeKey.ID(), config.P2P.ExternalAddress))
		if err != nil {
			return nil, errors.Wrap(err, "p2p.external_address is incorrect")
		}
		addrBook.AddOurAddress(addr)
	}
	if config.P2P.ListenAddress != "" {
		addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeKey.ID(), config.P2P.ListenAddress))
		if err != nil {
			return nil, errors.Wrap(err, "p2p.laddr is incorrect")
		}
		addrBook.AddOurAddress(addr)
	}

	sw.SetAddrBook(addrBook)

	return addrBook, nil
}

func createTssReactorAndAdd(config *cfg.Config, sw *p2p.Switch, logger log.Logger, storeDB dbm.DB, addr string)  {
	tssReactor := threshold.NewTssReactor(config, storeDB, addr)
	tssReactor.SetLogger(logger.With("module", "tss"))
	sw.AddReactor("tss", tssReactor)
	return
}