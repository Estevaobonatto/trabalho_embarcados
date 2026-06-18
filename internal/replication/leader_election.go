package replication

import (
	"uno-api/internal/model"
)

func (cs *ClusterState) realizarEleicao() {
	cs.mu.Lock()

	highestID := cs.ServidorID
	for _, peer := range cs.Peers {
		if peer.Active && peer.ServidorID > highestID {
			highestID = peer.ServidorID
		}
	}

	novoLider := (highestID == cs.ServidorID)
	liderAnterior := cs.LiderID

	if novoLider {
		cs.IsLider = true
		cs.LiderID = cs.ServidorID
		cs.EnderecoLider = cs.Endereco
	} else {
		cs.IsLider = false
		cs.LiderID = highestID
		for _, peer := range cs.Peers {
			if peer.ServidorID == highestID {
				cs.EnderecoLider = peer.URL
				break
			}
		}
	}

	cs.mu.Unlock()

	if novoLider && liderAnterior != cs.ServidorID {
		cs.registrarEventoCluster(model.LIDER_ALTERADO,
			"Novo lider eleito: "+cs.ServidorID)
	}
}
