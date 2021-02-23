package node

import (
	"CipherMachine/config"
	"CipherMachine/p2p"
	"CipherMachine/threshold"
	"CipherMachine/tsslib/ecdsa/keygen"
	"CipherMachine/tsslib/tss"
	"encoding/base64"
	"encoding/json"
	"fmt"
	create "github.com/ci123chain/ci123chain/sdk/init"
	"github.com/ci123chain/ci123chain/sdk/validator"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/log"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type ts struct {
	Bi *big.Int
}

func TestPrepare3peers(t *testing.T) {
	root1 := "../test1"
	root2 := "../test2"
	root3 := "../test3"
	err := os.MkdirAll(root1, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(root2, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(root3, os.ModePerm)
	if err != nil {
		panic(err)
	}

	os.Mkdir(filepath.Join(root1, "config"), os.ModePerm)
	os.Mkdir(filepath.Join(root1, "data"), os.ModePerm)

	os.Mkdir(filepath.Join(root2, "config"), os.ModePerm)
	os.Mkdir(filepath.Join(root2, "data"), os.ModePerm)

	os.Mkdir(filepath.Join(root3, "config"), os.ModePerm)
	os.Mkdir(filepath.Join(root3, "data"), os.ModePerm)

	privKey1, _, address1, _ := validator.NewValidatorKey()
	privKey2, _, address2, _ := validator.NewValidatorKey()
	privKey3, _, address3, _ := validator.NewValidatorKey()

	initFiles1, _ := create.NewInitFiles(privKey1, strings.ToLower(address1) + "@127.0.0.1:26656" + "," + strings.ToLower(address2) + "@127.0.0.1:36656" + "," + strings.ToLower(address3) + "@127.0.0.1:46656", false)

	configBytes1 := initFiles1.ConfigBytes
	nodeKeyBytes1 := initFiles1.NodeKeyBytes
	privKeyBytes1 := initFiles1.PrivValidatorKeyBytes
	privStateBytes1 := initFiles1.PrivValidatorStateBytes

	initFiles2, _ := create.NewInitFiles(privKey2, strings.ToLower(address1) + "@127.0.0.1:26656" + "," + strings.ToLower(address2) + "@127.0.0.1:36656" + "," + strings.ToLower(address3) + "@127.0.0.1:46656", false)

	configBytes2 := initFiles2.ConfigBytes
	nodeKeyBytes2 := initFiles2.NodeKeyBytes
	privKeyBytes2 := initFiles2.PrivValidatorKeyBytes
	privStateBytes2 := initFiles2.PrivValidatorStateBytes

	initFiles3, _ := create.NewInitFiles(privKey3, strings.ToLower(address1) + "@127.0.0.1:26656" + "," + strings.ToLower(address2) + "@127.0.0.1:36656" + "," + strings.ToLower(address3) + "@127.0.0.1:46656", false)

	configBytes3 := initFiles3.ConfigBytes
	nodeKeyBytes3 := initFiles3.NodeKeyBytes
	privKeyBytes3 := initFiles3.PrivValidatorKeyBytes
	privStateBytes3 := initFiles3.PrivValidatorStateBytes

	ioutil.WriteFile(filepath.Join(root1, "config/config.toml"), configBytes1, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root1, "config/node_key.json"), nodeKeyBytes1, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root1, "config/priv_validator_key.json"), privKeyBytes1, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root1, "data/priv_validator_state.json"), privStateBytes1, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root2, "config/config.toml"), configBytes2, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root2, "config/node_key.json"), nodeKeyBytes2, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root2, "config/priv_validator_key.json"), privKeyBytes2, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root2, "data/priv_validator_state.json"), privStateBytes2, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root3, "config/config.toml"), configBytes3, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root3, "config/node_key.json"), nodeKeyBytes3, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root3, "config/priv_validator_key.json"), privKeyBytes3, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root3, "data/priv_validator_state.json"), privStateBytes3, os.ModePerm)
}

func TestSdk(t *testing.T) {
	privKey1, _, address1, _ := validator.NewValidatorKey()
	initFiles1, err := create.NewInitFiles(privKey1, strings.ToLower(address1) + "@127.0.0.1:26656", true)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("./test.toml", initFiles1.ConfigBytes, os.ModePerm)
	viper.SetConfigFile("./test.toml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	c := new(config.Config)
	if err := viper.Unmarshal(&c); err != nil {
		panic(err)
	}
	k, err := p2p.LoadNodeKey("../test1/config/node_key.json")
	if err != nil {
		panic(err)
	}
	addr := k.ID()
	fmt.Println(addr)
	return
}

func Test2NodeStartStopWithConfig(t *testing.T) {
	root := "../test1"
	//defer os.RemoveAll(root)
	cfg, err := GetConfig(root)
	if err != nil {
		panic(err)
	}

	cfg.SetRoot(root)
	// create & start node
	n, err := DefaultNewNode(cfg, log.TestingLogger())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n.sw.NodeInfo())

	root2 := "../test2"
	//defer os.RemoveAll(root2)
	cfg2, err := GetConfig(root2)
	if err != nil {
		panic(err)
	}

	cfg2.SetRoot(root2)
	// create & start node
	n2, err := DefaultNewNode(cfg2, log.TestingLogger())
	require.NoError(t, err)
	err = n2.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n2.sw.NodeInfo())

	root3 := "../test3"
	//defer os.RemoveAll(root2)
	cfg3, err := GetConfig(root3)
	if err != nil {
		panic(err)
	}

	cfg3.SetRoot(root3)
	// create & start node
	n3, err := DefaultNewNode(cfg3, log.TestingLogger())
	require.NoError(t, err)
	err = n3.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n3.sw.NodeInfo())

	//// stop the node
	go func() {
		//n.Stop()
		select {
		case <-n.Quit():
		//case <-time.After(5 * time.Second):
		//	pid := os.Getpid()
		//	p, err := os.FindProcess(pid)
		//	if err != nil {
		//		panic(err)
		//	}
		//	err = p.Signal(syscall.SIGABRT)
		//	fmt.Println(err)
		//	t.Fatal("timed out waiting for shutdown")
		}
	}()

	//go func() {
	//	select {
	//	case <-n2.Quit():
	//		//case <-time.After(5 * time.Second):
	//		//	pid := os.Getpid()
	//		//	p, err := os.FindProcess(pid)
	//		//	if err != nil {
	//		//		panic(err)
	//		//	}
	//		//	err = p.Signal(syscall.SIGABRT)
	//		//	fmt.Println(err)
	//		//	t.Fatal("timed out waiting for shutdown")
	//	}
	//}()

	select {
	case <-n2.Quit():
		//case <-time.After(5 * time.Second):
		//	pid := os.Getpid()
		//	p, err := os.FindProcess(pid)
		//	if err != nil {
		//		panic(err)
		//	}
		//	err = p.Signal(syscall.SIGABRT)
		//	fmt.Println(err)
		//	t.Fatal("timed out waiting for shutdown")
	}
}

func GetConfig(root string) (*config.Config, error) {
	path := filepath.Join(root, "config/config.toml")
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	c := new(config.Config)
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}
	return c, nil
}

func TestConfig(t *testing.T) {
	configByte, _ := base64.StdEncoding.DecodeString("IyBUaGlzIGlzIGEgVE9NTCBjb25maWcgZmlsZS4KIyBGb3IgbW9yZSBpbmZvcm1hdGlvbiwgc2VlIGh0dHBzOi8vZ2l0aHViLmNvbS90b21sLWxhbmcvdG9tbAoKIyMjIyMgbWFpbiBiYXNlIGNvbmZpZyBvcHRpb25zICMjIyMjCgojIFRDUCBvciBVTklYIHNvY2tldCBhZGRyZXNzIG9mIHRoZSBBQkNJIGFwcGxpY2F0aW9uLAojIG9yIHRoZSBuYW1lIG9mIGFuIEFCQ0kgYXBwbGljYXRpb24gY29tcGlsZWQgaW4gd2l0aCB0aGUgVGVuZGVybWludCBiaW5hcnkKcHJveHlfYXBwID0gInRjcDovLzEyNy4wLjAuMToyNjY1OCIKCiMgQSBjdXN0b20gaHVtYW4gcmVhZGFibGUgbmFtZSBmb3IgdGhpcyBub2RlCm1vbmlrZXIgPSAiWWJxQ1ZtUloiCgojIElmIHRoaXMgbm9kZSBpcyBtYW55IGJsb2NrcyBiZWhpbmQgdGhlIHRpcCBvZiB0aGUgY2hhaW4sIEZhc3RTeW5jCiMgYWxsb3dzIHRoZW0gdG8gY2F0Y2h1cCBxdWlja2x5IGJ5IGRvd25sb2FkaW5nIGJsb2NrcyBpbiBwYXJhbGxlbAojIGFuZCB2ZXJpZnlpbmcgdGhlaXIgY29tbWl0cwpmYXN0X3N5bmMgPSB0cnVlCgojIERhdGFiYXNlIGJhY2tlbmQ6IGdvbGV2ZWxkYiB8IGNsZXZlbGRiIHwgYm9sdGRiCiMgKiBnb2xldmVsZGIgKGdpdGh1Yi5jb20vc3luZHRyL2dvbGV2ZWxkYiAtIG1vc3QgcG9wdWxhciBpbXBsZW1lbnRhdGlvbikKIyAgIC0gcHVyZSBnbwojICAgLSBzdGFibGUKIyAqIGNsZXZlbGRiICh1c2VzIGxldmlnbyB3cmFwcGVyKQojICAgLSBmYXN0CiMgICAtIHJlcXVpcmVzIGdjYwojICAgLSB1c2UgY2xldmVsZGIgYnVpbGQgdGFnIChnbyBidWlsZCAtdGFncyBjbGV2ZWxkYikKIyAqIGJvbHRkYiAodXNlcyBldGNkJ3MgZm9yayBvZiBib2x0IC0gZ2l0aHViLmNvbS9ldGNkLWlvL2Jib2x0KQojICAgLSBFWFBFUklNRU5UQUwKIyAgIC0gbWF5IGJlIGZhc3RlciBpcyBzb21lIHVzZS1jYXNlcyAocmFuZG9tIHJlYWRzIC0gaW5kZXhlcikKIyAgIC0gdXNlIGJvbHRkYiBidWlsZCB0YWcgKGdvIGJ1aWxkIC10YWdzIGJvbHRkYikKZGJfYmFja2VuZCA9ICJnb2xldmVsZGIiCgojIERhdGFiYXNlIGRpcmVjdG9yeQpkYl9kaXIgPSAiZGF0YSIKCiMgT3V0cHV0IGxldmVsIGZvciBsb2dnaW5nLCBpbmNsdWRpbmcgcGFja2FnZSBsZXZlbCBvcHRpb25zCmxvZ19sZXZlbCA9ICJtYWluOmluZm8sc3RhdGU6aW5mbywqOmVycm9yIgoKIyBPdXRwdXQgZm9ybWF0OiAncGxhaW4nIChjb2xvcmVkIHRleHQpIG9yICdqc29uJwpsb2dfZm9ybWF0ID0gInBsYWluIgoKIyMjIyMgYWRkaXRpb25hbCBiYXNlIGNvbmZpZyBvcHRpb25zICMjIyMjCgojIFBhdGggdG8gdGhlIEpTT04gZmlsZSBjb250YWluaW5nIHRoZSBpbml0aWFsIHZhbGlkYXRvciBzZXQgYW5kIG90aGVyIG1ldGEgZGF0YQpnZW5lc2lzX2ZpbGUgPSAiY29uZmlnL2dlbmVzaXMuanNvbiIKCiMgUGF0aCB0byB0aGUgSlNPTiBmaWxlIGNvbnRhaW5pbmcgdGhlIHByaXZhdGUga2V5IHRvIHVzZSBhcyBhIHZhbGlkYXRvciBpbiB0aGUgY29uc2Vuc3VzIHByb3RvY29sCnByaXZfdmFsaWRhdG9yX2tleV9maWxlID0gImNvbmZpZy9wcml2X3ZhbGlkYXRvcl9rZXkuanNvbiIKCiMgUGF0aCB0byB0aGUgSlNPTiBmaWxlIGNvbnRhaW5pbmcgdGhlIGxhc3Qgc2lnbiBzdGF0ZSBvZiBhIHZhbGlkYXRvcgpwcml2X3ZhbGlkYXRvcl9zdGF0ZV9maWxlID0gImRhdGEvcHJpdl92YWxpZGF0b3Jfc3RhdGUuanNvbiIKCiMgVENQIG9yIFVOSVggc29ja2V0IGFkZHJlc3MgZm9yIFRlbmRlcm1pbnQgdG8gbGlzdGVuIG9uIGZvcgojIGNvbm5lY3Rpb25zIGZyb20gYW4gZXh0ZXJuYWwgUHJpdlZhbGlkYXRvciBwcm9jZXNzCnByaXZfdmFsaWRhdG9yX2xhZGRyID0gIiIKCiMgUGF0aCB0byB0aGUgSlNPTiBmaWxlIGNvbnRhaW5pbmcgdGhlIHByaXZhdGUga2V5IHRvIHVzZSBmb3Igbm9kZSBhdXRoZW50aWNhdGlvbiBpbiB0aGUgcDJwIHByb3RvY29sCm5vZGVfa2V5X2ZpbGUgPSAiY29uZmlnL25vZGVfa2V5Lmpzb24iCgojIE1lY2hhbmlzbSB0byBjb25uZWN0IHRvIHRoZSBBQkNJIGFwcGxpY2F0aW9uOiBzb2NrZXQgfCBncnBjCmFiY2kgPSAic29ja2V0IgoKIyBUQ1Agb3IgVU5JWCBzb2NrZXQgYWRkcmVzcyBmb3IgdGhlIHByb2ZpbGluZyBzZXJ2ZXIgdG8gbGlzdGVuIG9uCnByb2ZfbGFkZHIgPSAibG9jYWxob3N0OjYwNjAiCgojIElmIHRydWUsIHF1ZXJ5IHRoZSBBQkNJIGFwcCBvbiBjb25uZWN0aW5nIHRvIGEgbmV3IHBlZXIKIyBzbyB0aGUgYXBwIGNhbiBkZWNpZGUgaWYgd2Ugc2hvdWxkIGtlZXAgdGhlIGNvbm5lY3Rpb24gb3Igbm90CmZpbHRlcl9wZWVycyA9IGZhbHNlCgojIyMjIyBhZHZhbmNlZCBjb25maWd1cmF0aW9uIG9wdGlvbnMgIyMjIyMKCiMjIyMjIHJwYyBzZXJ2ZXIgY29uZmlndXJhdGlvbiBvcHRpb25zICMjIyMjCltycGNdCgojIFRDUCBvciBVTklYIHNvY2tldCBhZGRyZXNzIGZvciB0aGUgUlBDIHNlcnZlciB0byBsaXN0ZW4gb24KbGFkZHIgPSAidGNwOi8vMC4wLjAuMDoyNjY1NyIKCiMgQSBsaXN0IG9mIG9yaWdpbnMgYSBjcm9zcy1kb21haW4gcmVxdWVzdCBjYW4gYmUgZXhlY3V0ZWQgZnJvbQojIERlZmF1bHQgdmFsdWUgJ1tdJyBkaXNhYmxlcyBjb3JzIHN1cHBvcnQKIyBVc2UgJ1siKiJdJyB0byBhbGxvdyBhbnkgb3JpZ2luCmNvcnNfYWxsb3dlZF9vcmlnaW5zID0gW10KCiMgQSBsaXN0IG9mIG1ldGhvZHMgdGhlIGNsaWVudCBpcyBhbGxvd2VkIHRvIHVzZSB3aXRoIGNyb3NzLWRvbWFpbiByZXF1ZXN0cwpjb3JzX2FsbG93ZWRfbWV0aG9kcyA9IFsiSEVBRCIsICJHRVQiLCAiUE9TVCIsIF0KCiMgQSBsaXN0IG9mIG5vbiBzaW1wbGUgaGVhZGVycyB0aGUgY2xpZW50IGlzIGFsbG93ZWQgdG8gdXNlIHdpdGggY3Jvc3MtZG9tYWluIHJlcXVlc3RzCmNvcnNfYWxsb3dlZF9oZWFkZXJzID0gWyJPcmlnaW4iLCAiQWNjZXB0IiwgIkNvbnRlbnQtVHlwZSIsICJYLVJlcXVlc3RlZC1XaXRoIiwgIlgtU2VydmVyLVRpbWUiLCBdCgojIFRDUCBvciBVTklYIHNvY2tldCBhZGRyZXNzIGZvciB0aGUgZ1JQQyBzZXJ2ZXIgdG8gbGlzdGVuIG9uCiMgTk9URTogVGhpcyBzZXJ2ZXIgb25seSBzdXBwb3J0cyAvYnJvYWRjYXN0X3R4X2NvbW1pdApncnBjX2xhZGRyID0gIiIKCiMgTWF4aW11bSBudW1iZXIgb2Ygc2ltdWx0YW5lb3VzIGNvbm5lY3Rpb25zLgojIERvZXMgbm90IGluY2x1ZGUgUlBDIChIVFRQJldlYlNvY2tldCkgY29ubmVjdGlvbnMuIFNlZSBtYXhfb3Blbl9jb25uZWN0aW9ucwojIElmIHlvdSB3YW50IHRvIGFjY2VwdCBhIGxhcmdlciBudW1iZXIgdGhhbiB0aGUgZGVmYXVsdCwgbWFrZSBzdXJlCiMgeW91IGluY3JlYXNlIHlvdXIgT1MgbGltaXRzLgojIDAgLSB1bmxpbWl0ZWQuCiMgU2hvdWxkIGJlIDwge3VsaW1pdCAtU259IC0ge01heE51bUluYm91bmRQZWVyc30gLSB7TWF4TnVtT3V0Ym91bmRQZWVyc30gLSB7TiBvZiB3YWwsIGRiIGFuZCBvdGhlciBvcGVuIGZpbGVzfQojIDEwMjQgLSA0MCAtIDEwIC0gNTAgPSA5MjQgPSB+OTAwCmdycGNfbWF4X29wZW5fY29ubmVjdGlvbnMgPSA5MDAKCiMgQWN0aXZhdGUgdW5zYWZlIFJQQyBjb21tYW5kcyBsaWtlIC9kaWFsX3NlZWRzIGFuZCAvdW5zYWZlX2ZsdXNoX21lbXBvb2wKdW5zYWZlID0gZmFsc2UKCiMgTWF4aW11bSBudW1iZXIgb2Ygc2ltdWx0YW5lb3VzIGNvbm5lY3Rpb25zIChpbmNsdWRpbmcgV2ViU29ja2V0KS4KIyBEb2VzIG5vdCBpbmNsdWRlIGdSUEMgY29ubmVjdGlvbnMuIFNlZSBncnBjX21heF9vcGVuX2Nvbm5lY3Rpb25zCiMgSWYgeW91IHdhbnQgdG8gYWNjZXB0IGEgbGFyZ2VyIG51bWJlciB0aGFuIHRoZSBkZWZhdWx0LCBtYWtlIHN1cmUKIyB5b3UgaW5jcmVhc2UgeW91ciBPUyBsaW1pdHMuCiMgMCAtIHVubGltaXRlZC4KIyBTaG91bGQgYmUgPCB7dWxpbWl0IC1Tbn0gLSB7TWF4TnVtSW5ib3VuZFBlZXJzfSAtIHtNYXhOdW1PdXRib3VuZFBlZXJzfSAtIHtOIG9mIHdhbCwgZGIgYW5kIG90aGVyIG9wZW4gZmlsZXN9CiMgMTAyNCAtIDQwIC0gMTAgLSA1MCA9IDkyNCA9IH45MDAKbWF4X29wZW5fY29ubmVjdGlvbnMgPSA5MDAKCiMgTWF4aW11bSBudW1iZXIgb2YgdW5pcXVlIGNsaWVudElEcyB0aGF0IGNhbiAvc3Vic2NyaWJlCiMgSWYgeW91J3JlIHVzaW5nIC9icm9hZGNhc3RfdHhfY29tbWl0LCBzZXQgdG8gdGhlIGVzdGltYXRlZCBtYXhpbXVtIG51bWJlcgojIG9mIGJyb2FkY2FzdF90eF9jb21taXQgY2FsbHMgcGVyIGJsb2NrLgptYXhfc3Vic2NyaXB0aW9uX2NsaWVudHMgPSAxMDAKCiMgTWF4aW11bSBudW1iZXIgb2YgdW5pcXVlIHF1ZXJpZXMgYSBnaXZlbiBjbGllbnQgY2FuIC9zdWJzY3JpYmUgdG8KIyBJZiB5b3UncmUgdXNpbmcgR1JQQyAob3IgTG9jYWwgUlBDIGNsaWVudCkgYW5kIC9icm9hZGNhc3RfdHhfY29tbWl0LCBzZXQgdG8KIyB0aGUgZXN0aW1hdGVkICMgbWF4aW11bSBudW1iZXIgb2YgYnJvYWRjYXN0X3R4X2NvbW1pdCBjYWxscyBwZXIgYmxvY2suCm1heF9zdWJzY3JpcHRpb25zX3Blcl9jbGllbnQgPSA1CgojIEhvdyBsb25nIHRvIHdhaXQgZm9yIGEgdHggdG8gYmUgY29tbWl0dGVkIGR1cmluZyAvYnJvYWRjYXN0X3R4X2NvbW1pdC4KIyBXQVJOSU5HOiBVc2luZyBhIHZhbHVlIGxhcmdlciB0aGFuIDEwcyB3aWxsIHJlc3VsdCBpbiBpbmNyZWFzaW5nIHRoZQojIGdsb2JhbCBIVFRQIHdyaXRlIHRpbWVvdXQsIHdoaWNoIGFwcGxpZXMgdG8gYWxsIGNvbm5lY3Rpb25zIGFuZCBlbmRwb2ludHMuCiMgU2VlIGh0dHBzOi8vZ2l0aHViLmNvbS90ZW5kZXJtaW50L3RlbmRlcm1pbnQvaXNzdWVzLzM0MzUKdGltZW91dF9icm9hZGNhc3RfdHhfY29tbWl0ID0gIjEwcyIKCiMgTWF4aW11bSBzaXplIG9mIHJlcXVlc3QgYm9keSwgaW4gYnl0ZXMKbWF4X2JvZHlfYnl0ZXMgPSAxMDAwMDAwCgojIE1heGltdW0gc2l6ZSBvZiByZXF1ZXN0IGhlYWRlciwgaW4gYnl0ZXMKbWF4X2hlYWRlcl9ieXRlcyA9IDEwNDg1NzYKCiMgVGhlIHBhdGggdG8gYSBmaWxlIGNvbnRhaW5pbmcgY2VydGlmaWNhdGUgdGhhdCBpcyB1c2VkIHRvIGNyZWF0ZSB0aGUgSFRUUFMgc2VydmVyLgojIE1pZ3RoIGJlIGVpdGhlciBhYnNvbHV0ZSBwYXRoIG9yIHBhdGggcmVsYXRlZCB0byB0ZW5kZXJtaW50J3MgY29uZmlnIGRpcmVjdG9yeS4KIyBJZiB0aGUgY2VydGlmaWNhdGUgaXMgc2lnbmVkIGJ5IGEgY2VydGlmaWNhdGUgYXV0aG9yaXR5LAojIHRoZSBjZXJ0RmlsZSBzaG91bGQgYmUgdGhlIGNvbmNhdGVuYXRpb24gb2YgdGhlIHNlcnZlcidzIGNlcnRpZmljYXRlLCBhbnkgaW50ZXJtZWRpYXRlcywKIyBhbmQgdGhlIENBJ3MgY2VydGlmaWNhdGUuCiMgTk9URTogYm90aCB0bHNfY2VydF9maWxlIGFuZCB0bHNfa2V5X2ZpbGUgbXVzdCBiZSBwcmVzZW50IGZvciBUZW5kZXJtaW50IHRvIGNyZWF0ZSBIVFRQUyBzZXJ2ZXIuIE90aGVyd2lzZSwgSFRUUCBzZXJ2ZXIgaXMgcnVuLgp0bHNfY2VydF9maWxlID0gIiIKCiMgVGhlIHBhdGggdG8gYSBmaWxlIGNvbnRhaW5pbmcgbWF0Y2hpbmcgcHJpdmF0ZSBrZXkgdGhhdCBpcyB1c2VkIHRvIGNyZWF0ZSB0aGUgSFRUUFMgc2VydmVyLgojIE1pZ3RoIGJlIGVpdGhlciBhYnNvbHV0ZSBwYXRoIG9yIHBhdGggcmVsYXRlZCB0byB0ZW5kZXJtaW50J3MgY29uZmlnIGRpcmVjdG9yeS4KIyBOT1RFOiBib3RoIHRsc19jZXJ0X2ZpbGUgYW5kIHRsc19rZXlfZmlsZSBtdXN0IGJlIHByZXNlbnQgZm9yIFRlbmRlcm1pbnQgdG8gY3JlYXRlIEhUVFBTIHNlcnZlci4gT3RoZXJ3aXNlLCBIVFRQIHNlcnZlciBpcyBydW4uCnRsc19rZXlfZmlsZSA9ICIiCgojIyMjIyBwZWVyIHRvIHBlZXIgY29uZmlndXJhdGlvbiBvcHRpb25zICMjIyMjCltwMnBdCgojIEFkZHJlc3MgdG8gbGlzdGVuIGZvciBpbmNvbWluZyBjb25uZWN0aW9ucwpsYWRkciA9ICJ0Y3A6Ly8wLjAuMC4wOjI2NjU2IgoKIyBBZGRyZXNzIHRvIGFkdmVydGlzZSB0byBwZWVycyBmb3IgdGhlbSB0byBkaWFsCiMgSWYgZW1wdHksIHdpbGwgdXNlIHRoZSBzYW1lIHBvcnQgYXMgdGhlIGxhZGRyLAojIGFuZCB3aWxsIGludHJvc3BlY3Qgb24gdGhlIGxpc3RlbmVyIG9yIHVzZSBVUG5QCiMgdG8gZmlndXJlIG91dCB0aGUgYWRkcmVzcy4KZXh0ZXJuYWxfYWRkcmVzcyA9ICIiCgojIENvbW1hIHNlcGFyYXRlZCBsaXN0IG9mIHNlZWQgbm9kZXMgdG8gY29ubmVjdCB0bwpzZWVkcyA9ICIiCgojIENvbW1hIHNlcGFyYXRlZCBsaXN0IG9mIG5vZGVzIHRvIGtlZXAgcGVyc2lzdGVudCBjb25uZWN0aW9ucyB0bwpwZXJzaXN0ZW50X3BlZXJzID0gImUyYTVkN2Q4MmE5MGUwYmJhYTY0MDYyNjllOGY0YTUxODIwN2QxYmJAbXlub2RlLnNoYXJkMS1lMmE1ZDdkODJhOTBlMGJiYWE2NDA2MjY5ZThmNGE1MTgyMDdkMWJiLmd3MDAyLm9uZWl0ZmFybS5jb206NzQ0M0B0bHMsZTgzM2JjM2U4OWI5MDY3ZTU4ZTgwMzdkY2E4NTZjNGE1ZTFkMTAyZEBteW5vZGUyLnNoYXJkMS1lODMzYmMzZTg5YjkwNjdlNThlODAzN2RjYTg1NmM0YTVlMWQxMDJkLmd3MDAyLm9uZWl0ZmFybS5jb206NzQ0M0B0bHMiCgojIFVQTlAgcG9ydCBmb3J3YXJkaW5nCnVwbnAgPSBmYWxzZQoKIyBQYXRoIHRvIGFkZHJlc3MgYm9vawphZGRyX2Jvb2tfZmlsZSA9ICJjb25maWcvYWRkcmJvb2suanNvbiIKCiMgU2V0IHRydWUgZm9yIHN0cmljdCBhZGRyZXNzIHJvdXRhYmlsaXR5IHJ1bGVzCiMgU2V0IGZhbHNlIGZvciBwcml2YXRlIG9yIGxvY2FsIG5ldHdvcmtzCmFkZHJfYm9va19zdHJpY3QgPSB0cnVlCgojIE1heGltdW0gbnVtYmVyIG9mIGluYm91bmQgcGVlcnMKbWF4X251bV9pbmJvdW5kX3BlZXJzID0gNDAKCiMgTWF4aW11bSBudW1iZXIgb2Ygb3V0Ym91bmQgcGVlcnMgdG8gY29ubmVjdCB0bywgZXhjbHVkaW5nIHBlcnNpc3RlbnQgcGVlcnMKbWF4X251bV9vdXRib3VuZF9wZWVycyA9IDEwCgojIFRpbWUgdG8gd2FpdCBiZWZvcmUgZmx1c2hpbmcgbWVzc2FnZXMgb3V0IG9uIHRoZSBjb25uZWN0aW9uCmZsdXNoX3Rocm90dGxlX3RpbWVvdXQgPSAiMTAwbXMiCgojIE1heGltdW0gc2l6ZSBvZiBhIG1lc3NhZ2UgcGFja2V0IHBheWxvYWQsIGluIGJ5dGVzCm1heF9wYWNrZXRfbXNnX3BheWxvYWRfc2l6ZSA9IDEwMjQKCiMgUmF0ZSBhdCB3aGljaCBwYWNrZXRzIGNhbiBiZSBzZW50LCBpbiBieXRlcy9zZWNvbmQKc2VuZF9yYXRlID0gNTEyMDAwMAoKIyBSYXRlIGF0IHdoaWNoIHBhY2tldHMgY2FuIGJlIHJlY2VpdmVkLCBpbiBieXRlcy9zZWNvbmQKcmVjdl9yYXRlID0gNTEyMDAwMAoKIyBTZXQgdHJ1ZSB0byBlbmFibGUgdGhlIHBlZXItZXhjaGFuZ2UgcmVhY3RvcgpwZXggPSB0cnVlCgojIFNlZWQgbW9kZSwgaW4gd2hpY2ggbm9kZSBjb25zdGFudGx5IGNyYXdscyB0aGUgbmV0d29yayBhbmQgbG9va3MgZm9yCiMgcGVlcnMuIElmIGFub3RoZXIgbm9kZSBhc2tzIGl0IGZvciBhZGRyZXNzZXMsIGl0IHJlc3BvbmRzIGFuZCBkaXNjb25uZWN0cy4KIwojIERvZXMgbm90IHdvcmsgaWYgdGhlIHBlZXItZXhjaGFuZ2UgcmVhY3RvciBpcyBkaXNhYmxlZC4Kc2VlZF9tb2RlID0gZmFsc2UKCiMgQ29tbWEgc2VwYXJhdGVkIGxpc3Qgb2YgcGVlciBJRHMgdG8ga2VlcCBwcml2YXRlICh3aWxsIG5vdCBiZSBnb3NzaXBlZCB0byBvdGhlciBwZWVycykKcHJpdmF0ZV9wZWVyX2lkcyA9ICIiCgojIFRvZ2dsZSB0byBkaXNhYmxlIGd1YXJkIGFnYWluc3QgcGVlcnMgY29ubmVjdGluZyBmcm9tIHRoZSBzYW1lIGlwLgphbGxvd19kdXBsaWNhdGVfaXAgPSBmYWxzZQoKIyBQZWVyIGNvbm5lY3Rpb24gY29uZmlndXJhdGlvbi4KaGFuZHNoYWtlX3RpbWVvdXQgPSAiMjBzIgpkaWFsX3RpbWVvdXQgPSAiM3MiCgojIFRMUyBvcHRpb24KdGxzX29wdGlvbiA9IHRydWUKClt0bHNfY29uZmlnXQojQmluZEFkZHJlc3NJUApiaW5kX2FkZHJlc3NfaXAgPSAiMTI3LjAuMC4xIgoKI0JpbmRBZGRyZXNzUG9ydApiaW5kX2FkZHJlc3NfcG9ydCA9IDkwMDEKCiNSZW1vdGVBZGRyZXNzSE9TVApyZW1vdGVfYWRkcmVzc19ob3N0ID0gIiIKCiNSZW1vdGVBZGRyZXNzUG9ydApyZW1vdGVfYWRkcmVzc19wb3J0ID0gNzQ0MwoKI1JlbW90ZVNlcnZlck5hbWUKcmVtb3RlX3NlcnZlcl9uYW1lID0gIiIKCiNSZW1vdGVUTFNDZXJ0VVJJCnJlbW90ZV90bHNfY2VydF91cmkgPSAiIgoKI1JlbW90ZVRMU0NlcnRLZXlVUkkKcmVtb3RlX3Rsc19jZXJ0X2tleV91cmkgPSAiIgoKI1JlbW90ZVRMU0RpYWxUaW1lb3V0CnJlbW90ZV90bHNfZGlhbF90aW1lb3V0ID0gNQoKI1JlbW90ZVRMU0luc2VjdXJlU2tpcFZlcmlmeQpyZW1vdGVfdGxzX2luc2VjdXJlX3NraXBfdmVyaWZ5ID0gdHJ1ZQoKIyMjIyMgbWVtcG9vbCBjb25maWd1cmF0aW9uIG9wdGlvbnMgIyMjIyMKW21lbXBvb2xdCgpyZWNoZWNrID0gdHJ1ZQpicm9hZGNhc3QgPSB0cnVlCndhbF9kaXIgPSAiIgoKIyBNYXhpbXVtIG51bWJlciBvZiB0cmFuc2FjdGlvbnMgaW4gdGhlIG1lbXBvb2wKc2l6ZSA9IDUwMDAKCiMgTGltaXQgdGhlIHRvdGFsIHNpemUgb2YgYWxsIHR4cyBpbiB0aGUgbWVtcG9vbC4KIyBUaGlzIG9ubHkgYWNjb3VudHMgZm9yIHJhdyB0cmFuc2FjdGlvbnMgKGUuZy4gZ2l2ZW4gMU1CIHRyYW5zYWN0aW9ucyBhbmQKIyBtYXhfdHhzX2J5dGVzPTVNQiwgbWVtcG9vbCB3aWxsIG9ubHkgYWNjZXB0IDUgdHJhbnNhY3Rpb25zKS4KbWF4X3R4c19ieXRlcyA9IDEwNzM3NDE4MjQKCiMgU2l6ZSBvZiB0aGUgY2FjaGUgKHVzZWQgdG8gZmlsdGVyIHRyYW5zYWN0aW9ucyB3ZSBzYXcgZWFybGllcikgaW4gdHJhbnNhY3Rpb25zCmNhY2hlX3NpemUgPSAxMDAwMAoKIyBNYXhpbXVtIHNpemUgb2YgYSBzaW5nbGUgdHJhbnNhY3Rpb24uCiMgTk9URTogdGhlIG1heCBzaXplIG9mIGEgdHggdHJhbnNtaXR0ZWQgb3ZlciB0aGUgbmV0d29yayBpcyB7bWF4X3R4X2J5dGVzfSArIHthbWlubyBvdmVyaGVhZH0uCm1heF90eF9ieXRlcyA9IDEwNDg1NzYKCiMjIyMjIGZhc3Qgc3luYyBjb25maWd1cmF0aW9uIG9wdGlvbnMgIyMjIyMKW2Zhc3RzeW5jXQoKIyBGYXN0IFN5bmMgdmVyc2lvbiB0byB1c2U6CiMgICAxKSAidjAiIChkZWZhdWx0KSAtIHRoZSBsZWdhY3kgZmFzdCBzeW5jIGltcGxlbWVudGF0aW9uCiMgICAyKSAidjEiIC0gcmVmYWN0b3Igb2YgdjAgdmVyc2lvbiBmb3IgYmV0dGVyIHRlc3RhYmlsaXR5CnZlcnNpb24gPSAidjAiCgojIyMjIyBjb25zZW5zdXMgY29uZmlndXJhdGlvbiBvcHRpb25zICMjIyMjCltjb25zZW5zdXNdCgp3YWxfZmlsZSA9ICJkYXRhL2NzLndhbC93YWwiCgp0aW1lb3V0X3Byb3Bvc2UgPSAiNXMiCnRpbWVvdXRfcHJvcG9zZV9kZWx0YSA9ICI1MDBtcyIKdGltZW91dF9wcmV2b3RlID0gIjFzIgp0aW1lb3V0X3ByZXZvdGVfZGVsdGEgPSAiNTAwbXMiCnRpbWVvdXRfcHJlY29tbWl0ID0gIjFzIgp0aW1lb3V0X3ByZWNvbW1pdF9kZWx0YSA9ICI1MDBtcyIKdGltZW91dF9jb21taXQgPSAiOHMiCgojIE1ha2UgcHJvZ3Jlc3MgYXMgc29vbiBhcyB3ZSBoYXZlIGFsbCB0aGUgcHJlY29tbWl0cyAoYXMgaWYgVGltZW91dENvbW1pdCA9IDApCnNraXBfdGltZW91dF9jb21taXQgPSBmYWxzZQoKIyBFbXB0eUJsb2NrcyBtb2RlIGFuZCBwb3NzaWJsZSBpbnRlcnZhbCBiZXR3ZWVuIGVtcHR5IGJsb2NrcwpjcmVhdGVfZW1wdHlfYmxvY2tzID0gdHJ1ZQpjcmVhdGVfZW1wdHlfYmxvY2tzX2ludGVydmFsID0gIjBzIgoKIyBSZWFjdG9yIHNsZWVwIGR1cmF0aW9uIHBhcmFtZXRlcnMKcGVlcl9nb3NzaXBfc2xlZXBfZHVyYXRpb24gPSAiMTAwbXMiCnBlZXJfcXVlcnlfbWFqMjNfc2xlZXBfZHVyYXRpb24gPSAiMnMiCgojIyMjIyB0cmFuc2FjdGlvbnMgaW5kZXhlciBjb25maWd1cmF0aW9uIG9wdGlvbnMgIyMjIyMKW3R4X2luZGV4XQoKIyBXaGF0IGluZGV4ZXIgdG8gdXNlIGZvciB0cmFuc2FjdGlvbnMKIwojIE9wdGlvbnM6CiMgICAxKSAibnVsbCIKIyAgIDIpICJrdiIgKGRlZmF1bHQpIC0gdGhlIHNpbXBsZXN0IHBvc3NpYmxlIGluZGV4ZXIsIGJhY2tlZCBieSBrZXktdmFsdWUgc3RvcmFnZSAoZGVmYXVsdHMgdG8gbGV2ZWxEQjsgc2VlIERCQmFja2VuZCkuCmluZGV4ZXIgPSAia3YiCgojIENvbW1hLXNlcGFyYXRlZCBsaXN0IG9mIHRhZ3MgdG8gaW5kZXggKGJ5IGRlZmF1bHQgdGhlIG9ubHkgdGFnIGlzICJ0eC5oYXNoIikKIwojIFlvdSBjYW4gYWxzbyBpbmRleCB0cmFuc2FjdGlvbnMgYnkgaGVpZ2h0IGJ5IGFkZGluZyAidHguaGVpZ2h0IiB0YWcgaGVyZS4KIwojIEl0J3MgcmVjb21tZW5kZWQgdG8gaW5kZXggb25seSBhIHN1YnNldCBvZiB0YWdzIGR1ZSB0byBwb3NzaWJsZSBtZW1vcnkKIyBibG9hdC4gVGhpcyBpcywgb2YgY291cnNlLCBkZXBlbmRzIG9uIHRoZSBpbmRleGVyJ3MgREIgYW5kIHRoZSB2b2x1bWUgb2YKIyB0cmFuc2FjdGlvbnMuCmluZGV4X3RhZ3MgPSAiY29udHJhY3QuYWRkcmVzcyxjb250cmFjdC5ldmVudC5kYXRhLGNvbnRyYWN0LmV2ZW50Lm5hbWUiCgojIFdoZW4gc2V0IHRvIHRydWUsIHRlbGxzIGluZGV4ZXIgdG8gaW5kZXggYWxsIHRhZ3MgKHByZWRlZmluZWQgdGFnczoKIyAidHguaGFzaCIsICJ0eC5oZWlnaHQiIGFuZCBhbGwgdGFncyBmcm9tIERlbGl2ZXJUeCByZXNwb25zZXMpLgojCiMgTm90ZSB0aGlzIG1heSBiZSBub3QgZGVzaXJhYmxlIChzZWUgdGhlIGNvbW1lbnQgYWJvdmUpLiBJbmRleFRhZ3MgaGFzIGEKIyBwcmVjZWRlbmNlIG92ZXIgSW5kZXhBbGxUYWdzIChpLmUuIHdoZW4gZ2l2ZW4gYm90aCwgSW5kZXhUYWdzIHdpbGwgYmUKIyBpbmRleGVkKS4KaW5kZXhfYWxsX3RhZ3MgPSBmYWxzZQoKIyMjIyMgaW5zdHJ1bWVudGF0aW9uIGNvbmZpZ3VyYXRpb24gb3B0aW9ucyAjIyMjIwpbaW5zdHJ1bWVudGF0aW9uXQoKIyBXaGVuIHRydWUsIFByb21ldGhldXMgbWV0cmljcyBhcmUgc2VydmVkIHVuZGVyIC9tZXRyaWNzIG9uCiMgUHJvbWV0aGV1c0xpc3RlbkFkZHIuCiMgQ2hlY2sgb3V0IHRoZSBkb2N1bWVudGF0aW9uIGZvciB0aGUgbGlzdCBvZiBhdmFpbGFibGUgbWV0cmljcy4KcHJvbWV0aGV1cyA9IHRydWUKCiMgQWRkcmVzcyB0byBsaXN0ZW4gZm9yIFByb21ldGhldXMgY29sbGVjdG9yKHMpIGNvbm5lY3Rpb25zCnByb21ldGhldXNfbGlzdGVuX2FkZHIgPSAiOjI2NjYwIgoKIyBNYXhpbXVtIG51bWJlciBvZiBzaW11bHRhbmVvdXMgY29ubmVjdGlvbnMuCiMgSWYgeW91IHdhbnQgdG8gYWNjZXB0IGEgbGFyZ2VyIG51bWJlciB0aGFuIHRoZSBkZWZhdWx0LCBtYWtlIHN1cmUKIyB5b3UgaW5jcmVhc2UgeW91ciBPUyBsaW1pdHMuCiMgMCAtIHVubGltaXRlZC4KbWF4X29wZW5fY29ubmVjdGlvbnMgPSAzCgojIEluc3RydW1lbnRhdGlvbiBuYW1lc3BhY2UKbmFtZXNwYWNlID0gInRlbmRlcm1pbnQiCg==")
	ioutil.WriteFile("./config.toml", configByte, os.ModePerm)
}

func TestKeygen(t *testing.T) {
	root := "../test1"
	//defer os.RemoveAll(root)
	cfg, err := GetConfig(root)
	if err != nil {
		panic(err)
	}

	cfg.SetRoot(root)
	// create & start node
	n, err := DefaultNewNode(cfg, log.TestingLogger())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n.sw.NodeInfo())

	root2 := "../test2"
	//defer os.RemoveAll(root2)
	cfg2, err := GetConfig(root2)
	if err != nil {
		panic(err)
	}

	cfg2.SetRoot(root2)
	// create & start node
	n2, err := DefaultNewNode(cfg2, log.TestingLogger())
	require.NoError(t, err)
	err = n2.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n2.sw.NodeInfo())

	//root3 := "../test3"
	////defer os.RemoveAll(root)
	//cfg3, err := GetConfig(root3)
	//if err != nil {
	//	panic(err)
	//}
	//
	//cfg3.SetRoot(root3)
	//// create & start node
	//n3, err := DefaultNewNode(cfg3, log.TestingLogger())
	//require.NoError(t, err)
	//err = n3.Start()
	//require.NoError(t, err)
	//
	//t.Logf("Started node %v", n3.sw.NodeInfo())

	time.Sleep(10 * time.Second)
	n.Keygen(1, "1")

	go func() {
		//n.Stop()
		select {
		case <-n.Quit():
			//case <-time.After(5 * time.Second):
			//	pid := os.Getpid()
			//	p, err := os.FindProcess(pid)
			//	if err != nil {
			//		panic(err)
			//	}
			//	err = p.Signal(syscall.SIGABRT)
			//	fmt.Println(err)
			//	t.Fatal("timed out waiting for shutdown")
		}
	}()

	select {
	case <-n2.Quit():
		//case <-time.After(5 * time.Second):
		//	pid := os.Getpid()
		//	p, err := os.FindProcess(pid)
		//	if err != nil {
		//		panic(err)
		//	}
		//	err = p.Signal(syscall.SIGABRT)
		//	fmt.Println(err)
		//	t.Fatal("timed out waiting for shutdown")
	}
}

func TestSigning(t *testing.T) {
	root := "../test1"
	//defer os.RemoveAll(root)
	cfg, err := GetConfig(root)
	if err != nil {
		panic(err)
	}

	cfg.SetRoot(root)
	// create & start node
	n, err := DefaultNewNode(cfg, log.TestingLogger())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n.sw.NodeInfo())

	root2 := "../test2"
	//defer os.RemoveAll(root2)
	cfg2, err := GetConfig(root2)
	if err != nil {
		panic(err)
	}

	cfg2.SetRoot(root2)
	// create & start node
	n2, err := DefaultNewNode(cfg2, log.TestingLogger())
	require.NoError(t, err)
	err = n2.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n2.sw.NodeInfo())

	time.Sleep(10 * time.Second)
	n.Signing(big.NewInt(42), "1")

	go func() {
		//n.Stop()
		select {
		case <-n.Quit():
			//case <-time.After(5 * time.Second):
			//	pid := os.Getpid()
			//	p, err := os.FindProcess(pid)
			//	if err != nil {
			//		panic(err)
			//	}
			//	err = p.Signal(syscall.SIGABRT)
			//	fmt.Println(err)
			//	t.Fatal("timed out waiting for shutdown")
		}
	}()

	select {
	case <-n2.Quit():
		//case <-time.After(5 * time.Second):
		//	pid := os.Getpid()
		//	p, err := os.FindProcess(pid)
		//	if err != nil {
		//		panic(err)
		//	}
		//	err = p.Signal(syscall.SIGABRT)
		//	fmt.Println(err)
		//	t.Fatal("timed out waiting for shutdown")
	}
}

func TestCdcMarshalUnmarshal(t *testing.T) {
	cdc := amino.NewCodec()
	cdc.RegisterConcrete(&threshold.TssMessage{}, "", nil)
	tmsg := &threshold.TssMessage{
		Sid: "1",
	}
	bz, err := cdc.MarshalBinaryBare(tmsg)
	if err != nil {
		panic(err)
	}
	var ttmsg threshold.TssMessage
	err = cdc.UnmarshalBinaryBare(bz, &ttmsg)
	if err != nil {
		panic(err)
	}

	cdc.RegisterConcrete(&threshold.SaveData{}, "123", nil)
	cdc.RegisterInterface((*tss.Round)(nil),nil)

	sd := &threshold.SaveData{
		PartySaveData: &threshold.PartySaveData{
			LocalPartySaveData: keygen.LocalPartySaveData{
				Ks: []*big.Int{big.NewInt(10), big.NewInt(100)},
			},
			SortedPartyIDs:     nil,
		},
		ConfigSaveData: &threshold.ConfigSaveData{
			LocalAddr: "1",
			Peers:     nil,
			Thresold:  3,
		},
	}
	sbz, err := cdc.MarshalBinaryBare(sd)
	if err != nil {
		panic(err)
	}
	var ssd threshold.SaveData
	err = cdc.UnmarshalBinaryBare(sbz, &ssd)
	if err != nil {
		panic(err)
	}
}

func TestTs(t *testing.T) {
	cdc := amino.NewCodec()
	cdc.RegisterConcrete(&ts{}, "123", nil)
	te := &ts{Bi: big.NewInt(10)}
	bz := cdc.MustMarshalBinaryBare(te)
	var tes ts
	cdc.MustUnmarshalBinaryBare(bz, &tes)
	fmt.Println(tes)
	bz, _ = json.Marshal(te)
	json.Unmarshal(bz, &tes)
	fmt.Println(tes)
}

func TestKeygenVerify(t *testing.T) {
	root := "../threshold/test"
	file1 := "keygen_data_0.json"
	file2 := "keygen_data_0.json"
	s1 := filepath.Join(root, file1)
	s2 := filepath.Join(root, file2)
	bz1, _ := ioutil.ReadFile(s1)
	bz2, _ := ioutil.ReadFile(s2)
	var sdt1, sdt2 keygen.LocalPartySaveData
	json.Unmarshal(bz1, &sdt1)
	json.Unmarshal(bz2, &sdt2)
	time.Sleep(1)
}