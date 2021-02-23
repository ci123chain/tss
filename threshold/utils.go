package threshold

import (
	"encoding/hex"
	"fmt"
	"CipherMachine/tsslib/tss"
	"strings"
)

func splitPeerFromPersistentPeer(persistentPeer string) (peersIDs []string) {
	peers := strings.SplitN(persistentPeer, ",", -1)
	for _, v := range peers {
		peeri := strings.SplitN(v, "@", 2)
		peersIDs = append(peersIDs, peeri[0])
	}
	return
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