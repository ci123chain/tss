// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package signing

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ipfs/go-log"
	"github.com/stretchr/testify/assert"

	"CipherMachine/tsslib/common"
	"CipherMachine/tsslib/ecdsa/keygen"
	"CipherMachine/tsslib/test"
	"CipherMachine/tsslib/tss"
)

const (
	testParticipants = test.TestParticipants
	testThreshold    = test.TestThreshold
)

func setUp(level string) {
	if err := log.SetLogLevel("tss-lib", level); err != nil {
		panic(err)
	}
}

func TestE2EConcurrent(t *testing.T) {
	setUp("info")

	threshold := 1
	root := "../../../threshold/test"
	file1 := "keygen_data_0.json"
	file2 := "keygen_data_1.json"
	s1 := filepath.Join(root, file1)
	s2 := filepath.Join(root, file2)
	bz1, _ := ioutil.ReadFile(s1)
	bz2, _ := ioutil.ReadFile(s2)
	var sdt1, sdt2 keygen.LocalPartySaveData
	json.Unmarshal(bz1, &sdt1)
	json.Unmarshal(bz2, &sdt2)
	keys := []keygen.LocalPartySaveData{sdt1, sdt2}
	signPIDs := generatePIDs(splitPeerFromPersistentPeer("bd6d6460d07980f17ec433d3bb401ecbe2ae2472@127.0.0.1:26656,8cf7506978b5f74ba44c50f2c41b6ac6ac982149@127.0.0.1:36656"))

	//threshold := testThreshold
	// PHASE: load keygen fixtures

	//keys, signPIDs, err := keygen.LoadKeygenTestFixturesRandomSet(testThreshold+1, testParticipants)
	//assert.NoError(t, err, "should load keygen fixtures")
	//assert.Equal(t, testThreshold+1, len(keys))
	//assert.Equal(t, testThreshold+1, len(signPIDs))

	// PHASE: signing
	// use a shuffled selection of the list of parties for this test
	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*LocalParty, 0, len(signPIDs))

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs))
	endCh := make(chan common.SignatureData, len(signPIDs))

	updater := test.SharedPartyUpdater

	// init the parties
	for i := 0; i < len(signPIDs); i++ {
		params := tss.NewParameters(p2pCtx, signPIDs[i], len(signPIDs), threshold)

		P := NewLocalParty(big.NewInt(42), params, keys[i], outCh, endCh).(*LocalParty)
		parties = append(parties, P)
		go func(P *LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	var ended int32
signing:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			common.Logger.Errorf("Error: %s", err)
			assert.FailNow(t, err.Error())
			break signing

		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh)
				}
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(parties[dest[0].Index], msg, errCh)
			}

		case <-endCh:
			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
				t.Logf("Done. Received signature data from %d participants", ended)
				R := parties[0].temp.bigR
				r := parties[0].temp.rx
				fmt.Printf("sign result: R(%s, %s), r=%s\n", R.X().String(), R.Y().String(), r.String())

				modN := common.ModInt(tss.EC().Params().N)

				// BEGIN check s correctness
				sumS := big.NewInt(0)
				for _, p := range parties {
					sumS = modN.Add(sumS, p.temp.si)
				}
				fmt.Printf("S: %s\n", sumS.String())
				// END check s correctness

				// BEGIN ECDSA verify
				pkX, pkY := keys[0].ECDSAPub.X(), keys[0].ECDSAPub.Y()
				pk := ecdsa.PublicKey{
					Curve: tss.EC(),
					X:     pkX,
					Y:     pkY,
				}
				ok := ecdsa.Verify(&pk, big.NewInt(42).Bytes(), R.X(), sumS)
				assert.True(t, ok, "ecdsa verify must pass")
				t.Log("ECDSA signing test done.")
				// END ECDSA verify

				break signing
			}
		}
	}
}

func generatePIDs(peersIDs []string) tss.SortedPartyIDs{
	ids := make(tss.UnSortedPartyIDs, 0, )
	for i := range peersIDs{
		key, err := hex.DecodeString(peersIDs[i])
		if err != nil {
			panic(err)
		}
		ids = append(ids, &tss.PartyID{
			MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
				Id:      peersIDs[i],
				Moniker: fmt.Sprintf("tss-peer[%d]", i+1),
				Key:     key,
			},
			Index: i,
		})
	}
	return tss.SortPartyIDs(ids)
}

func splitPeerFromPersistentPeer(persistentPeer string) (peersIDs []string) {
	peers := strings.SplitN(persistentPeer, ",", -1)
	for _, v := range peers {
		peeri := strings.SplitN(v, "@", 2)
		peersIDs = append(peersIDs, peeri[0])
	}
	return
}