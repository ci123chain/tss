package node

import (
	"CipherMachine/config"
	"CipherMachine/threshold"
	"CipherMachine/tsslib/ecdsa/keygen"
	"CipherMachine/tsslib/tss"
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

func writeConfigFile(root string, file *create.InitFiles) {
	ioutil.WriteFile(filepath.Join(root, "config/config.toml"), file.ConfigBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "config/node_key.json"), file.NodeKeyBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "config/priv_validator_key.json"), file.PrivValidatorKeyBytes, os.ModePerm)
	ioutil.WriteFile(filepath.Join(root, "data/priv_validator_state.json"), file.PrivValidatorStateBytes, os.ModePerm)
}

func Test3NodeStartWithConfig(t *testing.T) {
	n, _, _ := start3node(t)
	select {
	case <-n.Quit():
	}
}

func TestRunNode1(t *testing.T) {
	root1 := "../test1"
	//defer os.RemoveAll(root)
	n := getNewNode(root1)
	err := n.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n.sw.NodeInfo())

	select {
	case <-n.Quit():
	}
}

func TestRunNode2(t *testing.T) {
	root2 := "../test2"
	//defer os.RemoveAll(root2)
	n2 := getNewNode(root2)
	err := n2.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n2.sw.NodeInfo())

	select {
	case <-n2.Quit():
	}
}

func TestRunNode3(t *testing.T) {
	root3 := "../test3"
	//defer os.RemoveAll(root3)
	n3 := getNewNode(root3)
	err := n3.Start()
	require.NoError(t, err)
	t.Logf("Started node %v", n3.sw.NodeInfo())

	select {
	case <-n3.Quit():
	}
}

func TestKeygen(t *testing.T) {
	n, _, _ := start3node(t)
	time.Sleep(10 * time.Second)
	n.Keygen(2, "1")

	select {
	case <-n.Quit():
	}
}

func TestSigning(t *testing.T) {
	n, _, _ := start3node(t)

	time.Sleep(10 * time.Second)
	err := n.Signing(big.NewInt(42), "1")
	if err != nil {
		panic(err)
	}

	select {
	case <-n.Quit():
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
	time.Sleep(1)
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