package node

import (
	"CipherMachine/config"
	"CipherMachine/threshold"
	"CipherMachine/tsslib/ecdsa/keygen"
	"encoding/json"
	create "github.com/ci123chain/ci123chain/sdk/init"
	"github.com/ci123chain/ci123chain/sdk/validator"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//生成3个本地测试节点，需手动修改生成后的config.toml中相应port，已生成好的模版在test1、test2、test3文件夹中。
func TestPrepare3peersConfig(t *testing.T) {
	root1 := "../test1"
	root2 := "../test2"
	root3 := "../test3"

	os.MkdirAll(root1, os.ModePerm)
	os.MkdirAll(root2, os.ModePerm)
	os.MkdirAll(root3, os.ModePerm)

	os.MkdirAll(filepath.Join(root1, "config"), os.ModePerm)
	os.MkdirAll(filepath.Join(root1, "data"), os.ModePerm)

	os.MkdirAll(filepath.Join(root2, "config"), os.ModePerm)
	os.MkdirAll(filepath.Join(root2, "data"), os.ModePerm)

	os.MkdirAll(filepath.Join(root3, "config"), os.ModePerm)
	os.MkdirAll(filepath.Join(root3, "data"), os.ModePerm)

	privKey1, _, address1, _ := validator.NewValidatorKey()
	privKey2, _, address2, _ := validator.NewValidatorKey()
	privKey3, _, address3, _ := validator.NewValidatorKey()

	persistentPeer := strings.ToLower(address1) + "@127.0.0.1:26656" + "," + strings.ToLower(address2) + "@127.0.0.1:36656" + "," + strings.ToLower(address3) + "@127.0.0.1:46656"

	initFiles1, _ := create.NewInitFiles(privKey1, persistentPeer, false)
	writeConfigFile(root1, initFiles1)

	initFiles2, _ := create.NewInitFiles(privKey2, persistentPeer, false)
	writeConfigFile(root2, initFiles2)

	initFiles3, _ := create.NewInitFiles(privKey3, persistentPeer, false)
	writeConfigFile(root3, initFiles3)
}

//本地启动节点1
func TestRunNode1(t *testing.T) {
	root1 := "../test1"
	//defer os.RemoveAll(root)
	n := getNewNode(root1)
	err := n.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n.sw.NodeInfo())

	go n.Stop()

	select {
	case <-n.Quit():
	case <-time.After(5 * time.Second):
	}
}

//本地启动节点2
func TestRunNode2(t *testing.T) {
	root2 := "../test2"
	//defer os.RemoveAll(root2)
	n2 := getNewNode(root2)
	err := n2.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n2.sw.NodeInfo())

	go n2.Stop()

	select {
	case <-n2.Quit():
	case <-time.After(5 * time.Second):
	}
}

//本地启动节点3
func TestRunNode3(t *testing.T) {
	root3 := "../test3"
	//defer os.RemoveAll(root3)
	n3 := getNewNode(root3)
	err := n3.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n3.sw.NodeInfo())

	go n3.Stop()

	select {
	case <-n3.Quit():
	case <-time.After(5 * time.Second):
	}
}

//本地启动3个节点，并进行门限签名初始化
func TestKeygen(t *testing.T) {
	n, _, _ := start3node(t)
	time.Sleep(10 * time.Second)
	sessionID := threshold.SessionID("session-1")

	//传入门限数和sessionID
	resCh := n.Keygen(2, sessionID)
	if resCh == nil {
		return
	}

	select {
	case <-resCh:
		//wait for n2, n3 done, just for test
		time.Sleep(2 * time.Second)
	}
}

//本地启动3个节点，并进行门限签名，需提前调用keygen
func TestSigning(t *testing.T) {
	n, _, _ := start3node(t)
	time.Sleep(10 * time.Second)
	sessionID := threshold.SessionID("session-1")
	msg := big.NewInt(42)
	resCh, err := n.Signing(msg, sessionID)
	require.NoError(t, err)

	select {
	case signature := <-resCh:
		err := n.Verify(msg, sessionID, signature)
		require.NoError(t, err)
	}
}

//测试从文件中读取saveData
func TestReadKeygenData(t *testing.T) {
	root := "../threshold/test"
	file1 := "keygen_data_0.json"
	file2 := "keygen_data_1.json"
	s1 := filepath.Join(root, file1)
	s2 := filepath.Join(root, file2)
	bz1, _ := ioutil.ReadFile(s1)
	bz2, _ := ioutil.ReadFile(s2)
	var sdt1, sdt2 keygen.LocalPartySaveData
	json.Unmarshal(bz1, &sdt1)
	json.Unmarshal(bz2, &sdt2)
}

func writeConfigFile(root string, file *create.InitFiles) {
	ioutil.WriteFile(filepath.Join(root, "config/config.toml"), file.ConfigBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "config/node_key.json"), file.NodeKeyBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "config/priv_validator_key.json"), file.PrivValidatorKeyBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "data/priv_validator_state.json"), file.PrivValidatorStateBytes, os.ModePerm)
}

func start3node(t *testing.T) (n, n2, n3 *Node){
	root := "../test1"
	//defer os.RemoveAll(root)
	n = getNewNode(root)
	err := n.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n.sw.NodeInfo())

	root2 := "../test2"
	//defer os.RemoveAll(root2)
	n2 = getNewNode(root2)
	err = n2.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n2.sw.NodeInfo())

	root3 := "../test3"
	//defer os.RemoveAll(root3)
	n3 = getNewNode(root3)
	err = n3.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n3.sw.NodeInfo())
	return
}

func getNewNode(root string) *Node {
	cfg, err := getConfig(root)
	if err != nil {
		panic(err)
	}

	cfg.SetRoot(root)
	// create node
	n, err := DefaultNewNode(cfg, log.TestingLogger())
	if err != nil {
		panic(err)
	}
	return n
}

func getConfig(root string) (*config.Config, error) {
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