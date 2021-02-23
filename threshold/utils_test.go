package threshold

import (
	"fmt"
	"testing"
)

func TestSplitPeer(t *testing.T) {
	persistentPeer := "4848492b9eaa0bd65b1e2b30711a8c00ebd99190@127.0.0.1:26656,223785f662dfc0e332e1662ed7e890afbf8ae826@127.0.0.1:36656"
	splitPeerFromPersistentPeer(persistentPeer)
}

func TestSplit(t *testing.T) {
	s := "ws-ib6tx0zzublt9-79c9964954-ps422"
	aa := s[:16]
	fmt.Println(aa)
}