module CipherMachine

go 1.15

require (
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/ci123chain/ci123chain v1.3.3
	github.com/decred/dcrd/dcrec/edwards/v2 v2.0.1
	github.com/fortytw2/leaktest v1.3.0
	github.com/go-kit/kit v0.10.0
	github.com/golang/protobuf v1.4.3
	github.com/hashicorp/go-multierror v1.1.0
	github.com/ipfs/go-log v1.0.4
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/otiai10/mint v1.3.2 // indirect
	github.com/otiai10/primes v0.0.0-20180210170552-f6d2a1ba97c4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/rs/cors v1.7.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.32.3
	github.com/tendermint/tm-db v0.2.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
)

replace github.com/tendermint/tendermint => github.com/ci123chain/tendermint v0.32.7-rc6
