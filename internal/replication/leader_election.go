package replication

import (
	"uno-api/internal/model"
)

// realizarEleicao executa o algoritmo Bully simplificado:
// o servidor com maior servidorId ativo vence.
//
// Conforme Estrutura V2, ao assumir o lugar do líder anterior (failover),
// registra-se um evento FAILOVER e depois um LIDER_ALTERADO.
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
	enderecoAnterior := cs.EnderecoLider

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
		// Se o líder anterior estava em outro endereço, registramos o FAILOVER
		// (o antigo líder caiu e a réplica assumiu).
		if enderecoAnterior != "" && enderecoAnterior != cs.Endereco {
			cs.registrarEventoCluster(model.FAILOVER,
				"Antigo lider "+liderAnterior+" em "+enderecoAnterior+" caiu. Replica "+cs.ServidorID+" assumiu como novo lider.")
		}
		cs.registrarEventoCluster(model.LIDER_ALTERADO,
			"Novo lider eleito: "+cs.ServidorID)
	}
}
