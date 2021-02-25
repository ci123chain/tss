package threshold

import (
	"CipherMachine/config"
	"CipherMachine/p2p"
	"CipherMachine/p2p/conn"
	"CipherMachine/store"
	"CipherMachine/tsslib/common"
	"CipherMachine/tsslib/ecdsa/keygen"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	//"CipherMachine/tsslib/ecdsa/resharing"
	"CipherMachine/tsslib/ecdsa/signing"
	"CipherMachine/tsslib/tss"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	dbm "github.com/tendermint/tm-db"
	"math/big"
	"reflect"
	"sync"
)

const (
	storeKey   = "tss"
	TssChannel = byte(0x50)
	SaveDataKey = "SaveData"
	KeygenMsg = "keygenMsg"
	SigningMsg = "signingMsg"
)

// secret id
type SessionID string

// TssReactor handles tss sign and verify by broadcasting amongst peers.
type TssReactor struct {
	p2p.BaseReactor
	tssStore store.Store

	mtx    sync.Mutex

	//localConfig
	localAddr string
	peers []string

	keygenCh map[SessionID]struct {
		errCh chan *tss.Error
		outCh chan tss.Message
		endCh chan keygen.LocalPartySaveData
	}

	signingCh map[SessionID]struct {
		errCh chan *tss.Error
		outCh chan tss.Message
		endCh chan common.SignatureData
	}

	//control keygen/signing routine
	keygenRSwitch map[SessionID]bool
	signingRSwitch map[SessionID]bool

	localParty map[SessionID]tss.Party
	saveDatas map[SessionID]SaveData
}

type SaveData struct {
	PartySaveData *PartySaveData
	ConfigSaveData *ConfigSaveData
	//KeygenParty *keygen.LocalParty
	//SigningParty *signing.LocalParty
	//ResharingParty *resharing.LocalParty
}

//func(sd *SaveData) MarshalJSON() ([]byte, error) {
//	type jsonSaveData struct {
//
//	}
//}
//
//func(sd *SaveData) UnmarshalJSON([]byte) error {
//
//}

type PartySaveData struct {
	LocalPartySaveData keygen.LocalPartySaveData
	SortedPartyIDs tss.SortedPartyIDs
}

type ConfigSaveData struct {
	LocalAddr string
	Peers []string
	Thresold int
}

// NewTssReactor returns a new TssReactor with the given config.
func NewTssReactor(config *config.Config, storeDB dbm.DB, addr string) *TssReactor {
	peers := splitPeerFromPersistentPeer(config.P2P.PersistentPeers)
	tssStore := store.NewStore(storeDB, []byte(storeKey))
	tsR := &TssReactor{
		localAddr: addr,
		tssStore: tssStore,
		peers: peers,
		keygenCh: make(map[SessionID]struct {
			errCh chan *tss.Error
			outCh chan tss.Message
			endCh chan keygen.LocalPartySaveData
		}),
		signingCh: make(map[SessionID]struct {
			errCh chan *tss.Error
			outCh chan tss.Message
			endCh chan common.SignatureData
		}),
		localParty: make(map[SessionID]tss.Party),
		saveDatas: make(map[SessionID]SaveData),
		keygenRSwitch: make(map[SessionID]bool),
		signingRSwitch: make(map[SessionID]bool),
	}
	tsR.BaseReactor = *p2p.NewBaseReactor("TssReactor", tsR)
	return tsR
}

func (tsr *TssReactor) SetSwitch(sw *p2p.Switch) {
	tsr.Switch = sw
}

func (tsr *TssReactor) GetChannels() []*conn.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:       TssChannel,
			Priority: 5,
		},
	}
}

//todo: resharing?
func (tsr *TssReactor) AddPeer(peer p2p.Peer) {}

//todo: resharing?
func (tsr *TssReactor) RemovePeer(peer p2p.Peer, reason interface{}) {}

func (tsr *TssReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	msg, err := decodeMsg(msgBytes)
	if err != nil {
		tsr.Logger.Error("Error decoding message", "src", src, "chId", chID, "msg", msg, "err", err, "bytes", msgBytes)
		return
	}

	pMsg, err := tss.ParseWireMessage(msg.Pmsg, msg.From, msg.Isbroadcast)
	if err != nil {
		tsr.Logger.Error("Error parseWire message", "err", err)
		return
	}
	tsr.Logger.Debug("receive msg", "from", msg.From.Id, "me", tsr.localAddr)
	if msg.PmsgType == KeygenMsg {
		//no keygen
		if !tsr.keygenRSwitch[msg.Sid] {
			tsr.Keygen(msg.Threshold, msg.Sid)
		}
		_, err := tsr.localParty[msg.Sid].Update(pMsg)
		if err != nil {
			tsr.keygenCh[msg.Sid].errCh <- err
		}
		return
	} else if msg.PmsgType == SigningMsg {
		//keygen done, do signing
		if !tsr.signingRSwitch[msg.Sid] {
			x := new(big.Int)
			x.SetBytes(msg.Msg)
			if err := tsr.Signing(x, msg.Sid); err != nil {
				tsr.Logger.Error("signing err", err)
				return
			}
		}
		//} else {
		//	if err := tsr.getSaveData(msg.Sid); err != nil {
		//		tsr.Logger.Error("get saveData failed", "err", err)
		//		return
		//	}
		//
		//	if err := tsr.validateSaveData(msg.Sid); err != nil{
		//		tsr.Logger.Error("validate saveData failed", "err", err)
		//		return
		//	}
		//}
		ok, err := tsr.localParty[msg.Sid].Update(pMsg)
		if err != nil {
			tsr.signingCh[msg.Sid].errCh <- err
		}
		time.Sleep(2 * time.Second)
		if ok {
			tsr.Logger.Debug("receive success", "local round", tsr.localParty[msg.Sid].(*signing.LocalParty).BaseParty.String(), "me", tsr.localAddr)
		} else {
			tsr.Logger.Debug("update response not ok")
		}
		return
	}
}

func (tsr *TssReactor) InitPeer(peer p2p.Peer) p2p.Peer {
	return peer
}

//keygen initiator function
func (tsr *TssReactor) Keygen(threshold int, sid SessionID) {
	if 	tsr.localParty[sid] != nil {
		return
	}

	//prepare keygen params
	pIDs := generatePIDs(tsr.peers)
	p2pCtx := tss.NewPeerContext(pIDs)

	var partyIndex int
	for i := range pIDs {
		if pIDs[i].Id == tsr.localAddr {
			partyIndex = i
			break
		}
	}
	keygenCh := struct {
		errCh chan *tss.Error
		outCh chan tss.Message
		endCh chan keygen.LocalPartySaveData
	}{
		errCh: make(chan *tss.Error),
		outCh: make(chan tss.Message),
		endCh: make(chan keygen.LocalPartySaveData),
	}

	tsr.keygenCh[sid] = keygenCh

	params := tss.NewParameters(p2pCtx, pIDs[partyIndex], len(pIDs), threshold)
	localParty := keygen.NewLocalParty(params, tsr.keygenCh[sid].outCh, tsr.keygenCh[sid].endCh).(*keygen.LocalParty)
	tsr.localParty[sid] = localParty

	go func(P *keygen.LocalParty) {
		if err := P.Start(); err != nil {
			tsr.keygenCh[sid].errCh <- err
		}
	}(localParty)

	go tsr.keygenRoutine(partyIndex, localParty, pIDs, sid, threshold)
	return
}

func (tsr *TssReactor) keygenRoutine(partyIndex int, party *keygen.LocalParty, pIDs tss.SortedPartyIDs, sid SessionID, threshold int) {
	if tsr.keygenRSwitch[sid] {
		return
	}
	tsr.keygenRSwitch[sid] = true
	defer func() {
		tsr.keygenRSwitch[sid] = false
	}()

	for {
		select {
		case err := <- tsr.keygenCh[sid].errCh:
			tsr.Logger.Error("keygen error", err)
			return

		case msg := <- tsr.keygenCh[sid].outCh:
			dest := msg.GetTo()
			if dest == nil { // broadcast
				for _, id := range tsr.peers {
					if id == msg.GetFrom().Id {
						continue
					}
					tsr.TrySendByPeerID(party, sid, id, threshold, msg, KeygenMsg, nil, tsr.keygenCh[sid].errCh)
				}
			} else { // point-to-point
				if dest[0].Id == msg.GetFrom().Id {
					tsr.Logger.Error("msg error", "Error: %s", errors.New(fmt.Sprintf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)))
					return
				}
				tsr.TrySendByPeerID(party, sid, dest[0].Id, threshold, msg, KeygenMsg, nil, tsr.keygenCh[sid].errCh)
			}

		case save := <- tsr.keygenCh[sid].endCh:
			tryWriteTestFixtureFile(partyIndex, save)
			saveData := SaveData{
				PartySaveData:  &PartySaveData{
					LocalPartySaveData: save,
					SortedPartyIDs:     pIDs,
				},
				ConfigSaveData: &ConfigSaveData{
					LocalAddr: tsr.localAddr,
					Peers:     tsr.peers,
					Thresold:  threshold,
				},
				//KeygenParty:    party,
				//SigningParty:   nil,
				//ResharingParty: nil,
			}
			tsr.saveDatas[sid] = saveData
			//psd := cdc.MustMarshalBinaryBare(saveData)
			psd, _ := json.Marshal(saveData)
			tsr.tssStore.Set(tsr.newPrefixKey(SaveDataKey, sid), psd)
			//psdd := tsr.tssStore.Get(tsr.newPrefixKey(SaveDataKey, sid))
			//var saveDatad SaveData
			//if err := cdc.UnmarshalBinaryBare(psdd, &saveDatad); err != nil {
			//	panic(err)
			//}
			var saveDatad SaveData
			if err := json.Unmarshal(psd, &saveDatad); err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(saveData, saveDatad) {
				panic("unmarshal failed")
			}
			tsr.Logger.Info(fmt.Sprintf("sid: %s:keygen done", sid))
			return
		}
	}
}

//signing initiator function
func (tsr *TssReactor) Signing(msg *big.Int, sid SessionID) error {
	if err := tsr.getSaveData(sid); err != nil {
		return err
	}
	if err := tsr.validateSaveData(sid); err != nil {
		return err
	}

	signPIDs := tsr.saveDatas[sid].PartySaveData.SortedPartyIDs
	p2pCtx := tss.NewPeerContext(signPIDs)

	signingCh := struct {
		errCh chan *tss.Error
		outCh chan tss.Message
		endCh chan common.SignatureData
	}{
		errCh: make(chan *tss.Error),
		outCh: make(chan tss.Message),
		endCh: make(chan common.SignatureData),
	}

	tsr.signingCh[sid] = signingCh

	var partyIndex int
	for i := range signPIDs {
		if signPIDs[i].Id == tsr.localAddr {
			partyIndex = i
			break
		}
	}

	params := tss.NewParameters(p2pCtx, signPIDs[partyIndex], len(signPIDs), tsr.saveDatas[sid].ConfigSaveData.Thresold)
	localParty := signing.NewLocalParty(msg, params, tsr.saveDatas[sid].PartySaveData.LocalPartySaveData, tsr.signingCh[sid].outCh, tsr.signingCh[sid].endCh).(*signing.LocalParty)
	tsr.localParty[sid] = localParty

	go func(P *signing.LocalParty) {
		if err := P.Start(); err != nil {
			tsr.signingCh[sid].errCh <- err
		}
	}(localParty)

	go tsr.signingRoutine(msg, localParty, sid)
	return nil
}

func (tsr *TssReactor) signingRoutine(msg *big.Int, party *signing.LocalParty, sid SessionID) {
	if tsr.signingRSwitch[sid] {
		return
	}
	tsr.signingRSwitch[sid] = true
	defer func() {
		tsr.signingRSwitch[sid] = false
	}()

	for {
		select {
		case err := <-tsr.signingCh[sid].errCh:
			tsr.Logger.Error("signing err", err)
			return

		case pmsg := <-tsr.signingCh[sid].outCh:
			dest := pmsg.GetTo()
			if dest == nil {
				for _, id := range tsr.peers {
					if id == pmsg.GetFrom().Id {
						continue
					}
					tsr.TrySendByPeerID(party, sid, id, tsr.saveDatas[sid].ConfigSaveData.Thresold, pmsg, SigningMsg, msg, tsr.signingCh[sid].errCh)
				}
			} else {
				if dest[0].Id == pmsg.GetFrom().Id {
					tsr.Logger.Error("msg error", "Error: %s", errors.New(fmt.Sprintf("party %d tried to send a message to itself (%d)", dest[0].Index, pmsg.GetFrom().Index)))
					return
				}
				tsr.TrySendByPeerID(party, sid, dest[0].Id, tsr.saveDatas[sid].ConfigSaveData.Thresold, pmsg, SigningMsg, msg, tsr.signingCh[sid].errCh)
			}

		case signature := <-tsr.signingCh[sid].endCh:
			tsr.Logger.Info(fmt.Sprintf("Done. Received signature data"))
			sigr := signature.R
			sigR := big.NewInt(0)
			sigR.SetBytes(sigr)

			sums := signature.S
			sumS := big.NewInt(0)
			sumS.SetBytes(sums)

			// BEGIN ECDSA verify
			pkX, pkY := tsr.saveDatas[sid].PartySaveData.LocalPartySaveData.ECDSAPub.X(), tsr.saveDatas[sid].PartySaveData.LocalPartySaveData.ECDSAPub.Y()
			pk := ecdsa.PublicKey{
				Curve: tss.EC(),
				X:     pkX,
				Y:     pkY,
			}
			ok := ecdsa.Verify(&pk, msg.Bytes(), sigR, sumS)
			if !ok {
				tsr.Logger.Info("ECDSA verify failed.")
			}
			tsr.Logger.Info("ECDSA signing done.")
			// END ECDSA verify
			return
		}
	}
}

//todo: resharing initiator function
func (tsr *TssReactor) Resharing() error {
	return nil
}

func (tsr *TssReactor) TrySendByPeerID(party tss.Party, sid SessionID, pid string, threshold int, pmsg tss.Message, pmsgType msgType, msg *big.Int, errCh chan<- *tss.Error)  {
	bz, _, err := pmsg.WireBytes()
	if err != nil {
		errCh <- party.WrapError(err)
		return
	}
	from := pmsg.GetFrom()
	isbroadcast := pmsg.IsBroadcast()
	tssmsg := &TssMessage{
		Sid: sid,
		Isbroadcast: isbroadcast,
		Threshold: threshold,
		From: from,
		Pmsg:  bz,
		PmsgType: pmsgType,
	}
	if pmsgType == SigningMsg{ // signing msg
		tssmsg.Msg = msg.Bytes()
	}
	tssbz, err := cdc.MarshalBinaryBare(tssmsg)
	if err != nil {
		errCh <- party.WrapError(err)
		return
	}
	src := tsr.Switch.Peers().Get(p2p.ID(pid))
	if src == nil {
		panic(fmt.Sprintf("cannot find this pid: %s in switch", pid))
	}
 	queued := src.TrySend(TssChannel, tssbz)
	for {
		if !queued {
			tsr.Logger.Debug("Send queue is full, try again", "peer", src.ID())
			time.Sleep(1 * time.Second)
			queued = src.TrySend(TssChannel, tssbz)
		} else {
			return
		}
	}
}

func (tsr *TssReactor) getSaveData(sid SessionID) error {
	if psd := tsr.tssStore.Get(tsr.newPrefixKey(SaveDataKey, sid)); psd != nil {
		var saveData SaveData
		//if err := cdc.UnmarshalBinaryBare(psd, &saveData); err != nil {
		//	return err
		//}
		if err := json.Unmarshal(psd, &saveData); err != nil {
			return err
		}
		tsr.saveDatas[sid] = saveData
		tryWriteTestFixtureFile(rand.Int()+2, saveData.PartySaveData.LocalPartySaveData)
		return nil
	}
	return errors.New("No SaveData")
}

func (tsr *TssReactor) validateSaveData(sid SessionID) error {
	if !reflect.DeepEqual(tsr.peers, tsr.saveDatas[sid].ConfigSaveData.Peers) {
		return errors.New("Peers is not the same as the previous peers")
	}
	if tsr.localAddr != tsr.saveDatas[sid].ConfigSaveData.LocalAddr {
		return errors.New("LocalAddress is not the same as the previous localAddress")
	}
	return nil
}

func (tsr *TssReactor) newPrefixKey(prefix string, sid SessionID) []byte {
	return []byte(prefix + string(sid))
}

func decodeMsg(bz []byte) (msg TssMessage, err error) {
	err = cdc.UnmarshalBinaryBare(bz, &msg)
	return
}

//// RegisterTssMessages registers the tss messages for amino encoding.
//func RegisterTssMessages(cdc *amino.Codec) {
//	cdc.RegisterInterface((*tss.Message)(nil), nil)
//
//	cdc.RegisterConcrete(&keygen.KGRound1Message{}, "tss/keygen/round1message", nil)
//	cdc.RegisterConcrete(&keygen.KGRound2Message1{}, "tss/keygen/round2message1", nil)
//	cdc.RegisterConcrete(&keygen.KGRound2Message2{}, "tss/keygen/round2message2", nil)
//	cdc.RegisterConcrete(&keygen.KGRound3Message{}, "tss/keygen/round3message", nil)
//
//	cdc.RegisterConcrete(&signing.SignRound1Message1{}, "tss/signing/round1message1", nil)
//	cdc.RegisterConcrete(&signing.SignRound1Message2{}, "tss/signing/round1message2", nil)
//	cdc.RegisterConcrete(&signing.SignRound2Message{}, "tss/signing/round2message", nil)
//	cdc.RegisterConcrete(&signing.SignRound3Message{}, "tss/signing/round3message", nil)
//	cdc.RegisterConcrete(&signing.SignRound4Message{}, "tss/signing/round4message", nil)
//	cdc.RegisterConcrete(&signing.SignRound5Message{}, "tss/signing/round5message", nil)
//	cdc.RegisterConcrete(&signing.SignRound6Message{}, "tss/signing/round6message", nil)
//	cdc.RegisterConcrete(&signing.SignRound7Message{}, "tss/signing/round7message", nil)
//	cdc.RegisterConcrete(&signing.SignRound8Message{}, "tss/signing/round8message", nil)
//	cdc.RegisterConcrete(&signing.SignRound9Message{}, "tss/signing/round9message", nil)
//
//	cdc.RegisterConcrete(&resharing.DGRound1Message{}, "tss/resharing/round1message", nil)
//	cdc.RegisterConcrete(&resharing.DGRound2Message1{}, "tss/resharing/round2message1", nil)
//	cdc.RegisterConcrete(&resharing.DGRound2Message2{}, "tss/resharing/round2message2", nil)
//	cdc.RegisterConcrete(&resharing.DGRound3Message1{}, "tss/resharing/round3message1", nil)
//	cdc.RegisterConcrete(&resharing.DGRound3Message2{}, "tss/resharing/round3message2", nil)
//	cdc.RegisterConcrete(&resharing.DGRound4Message{}, "tss/resharing/round1message", nil)
//}
//

func tryWriteTestFixtureFile(index int, data keygen.LocalPartySaveData) {
	fixtureFileName := makeTestFixtureFilePath(index)

	// fixture file does not already exist?
	// if it does, we won't re-create it here
	fi, err := os.Stat(fixtureFileName)
	if !(err == nil && fi != nil && !fi.IsDir()) {
		fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
		}
		bz, err := json.Marshal(&data)
		if err != nil {
		}
		_, err = fd.Write(bz)
		if err != nil {
		}
	} else {
	}
	//
}

const (
	testFixtureDirFormat  = "%s/test"
	testFixtureFileFormat = "keygen_data_%d.json"
)

func makeTestFixtureFilePath(partyIndex int) string {
	_, callerFileName, _, _ := runtime.Caller(0)
	srcDirName := filepath.Dir(callerFileName)
	fixtureDirName := fmt.Sprintf(testFixtureDirFormat, srcDirName)
	return fmt.Sprintf("%s/"+testFixtureFileFormat, fixtureDirName, partyIndex)
}
