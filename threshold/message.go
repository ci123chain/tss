package threshold

import (
	"CipherMachine/tsslib/tss"
	"github.com/tendermint/go-amino"
)

type msgType string

type TssMessage struct {
	Sid SessionID
	Threshold int
	From *tss.PartyID
	Isbroadcast bool
	Msg []byte
	Pmsg []byte
	PmsgType msgType
}

func RegisterTssMessages(cdc *amino.Codec) {
	cdc.RegisterConcrete(&TssMessage{}, "CipherMachine/tssMessage", nil)
}