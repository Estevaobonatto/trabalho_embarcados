package replication

import (
	"time"
)

func (cs *ClusterState) loopHeartbeat() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopCh:
			return
		case <-ticker.C:
			cs.verificarPeers()
		}
	}
}

func (cs *ClusterState) verificarPeers() {
	liderAtivo := true
	existeSuperior := false

	cs.mu.RLock()
	currentLiderURL := cs.EnderecoLider
	currentLiderID := cs.LiderID
	myID := cs.ServidorID
	cs.mu.RUnlock()

	for peerURL := range cs.Peers {
		id := cs.obterServidorId(peerURL)
		cs.mu.Lock()
		if id != "" {
			if peer, ok := cs.Peers[peerURL]; ok {
				peer.ServidorID = id
				peer.Active = true
				peer.LastSeen = time.Now()
				if id > currentLiderID {
					existeSuperior = true
				}
			}
		} else {
			if peer, ok := cs.Peers[peerURL]; ok {
				if peer.Active && time.Since(peer.LastSeen) > 6*time.Second {
					peer.Active = false
				}
			}
		}
		if peer, ok := cs.Peers[peerURL]; ok && peer.URL == currentLiderURL && !peer.Active {
			liderAtivo = false
		}
		cs.mu.Unlock()
	}

	if (existeSuperior && currentLiderID != myID) || (!liderAtivo && currentLiderURL != cs.Endereco) {
		cs.realizarEleicao()
	}

	if existeSuperior && currentLiderID == myID {
		cs.realizarEleicao()
	}
}

func (cs *ClusterState) IsPeerActive(peerURL string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if peer, ok := cs.Peers[peerURL]; ok {
		return peer.Active
	}
	return false
}
