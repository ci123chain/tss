package threshold

import (
	"CipherMachine/tsslib/tss"
	"github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

func init() {
	RegisterTssMessages(cdc)
	//RegisterSaveDatas(cdc)
}

func RegisterSaveDatas(cdc *amino.Codec) {
	cdc.RegisterInterface((*tss.Round)(nil), nil)
	cdc.RegisterConcrete(&SaveData{}, "CipherMachine/savedata", nil)
}