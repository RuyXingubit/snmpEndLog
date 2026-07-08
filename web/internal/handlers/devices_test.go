package handlers

import (
	"testing"
)

func TestBgpPeerStruct(t *testing.T) {
	// A simple test to ensure BgpPeer struct is correctly defined
	var peer BgpPeer
	
	peer.PeerAddr = "192.168.1.1"
	var as int64 = 65000
	peer.PeerAs = &as
	
	if peer.PeerAddr != "192.168.1.1" {
		t.Errorf("Expected PeerAddr 192.168.1.1, got %s", peer.PeerAddr)
	}
	
	if *peer.PeerAs != 65000 {
		t.Errorf("Expected PeerAs 65000, got %d", *peer.PeerAs)
	}
}
