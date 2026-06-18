package replication

import (
	"time"

	"uno-api/internal/model"
)

func (cs *ClusterState) loopSincronizacao() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopCh:
			return
		case <-ticker.C:
			if !cs.IsLeader() {
				cs.sincronizarComLider()
			}
		}
	}
}

func (cs *ClusterState) sincronizarComLider() {
	liderURL := cs.GetLiderEndereco()
	if liderURL == "" || liderURL == cs.Endereco {
		return
	}

	cs.sincronizarJogos(liderURL)
	cs.sincronizarLeaderboard(liderURL)
}

type resumoJogoRemoto struct {
	GameId       string `json:"gameId"`
	VersaoEstado int    `json:"versaoEstado"`
}

func (cs *ClusterState) sincronizarJogos(liderURL string) {
	var remotos []resumoJogoRemoto
	if err := cs.doGetPeer(liderURL, "/_replicacao/jogos", &remotos); err != nil {
		return
	}

	for _, r := range remotos {
		jogo, err := cs.PM.ObterJogo(r.GameId)
		if err != nil || jogo.VersaoEstado < r.VersaoEstado {
			var snap model.JogoSnapshot
			if err := cs.doGetPeer(liderURL, "/_replicacao/jogos/"+r.GameId, &snap); err != nil {
				continue
			}
			cs.PM.ImportarSnapshot(&snap)
		}
	}
}

type jogadorRemoto struct {
	JogadorId string `json:"jogadorId"`
	Nome      string `json:"nome"`
	Vitorias  int    `json:"vitorias"`
}

func (cs *ClusterState) sincronizarLeaderboard(liderURL string) {
	var remotos []jogadorRemoto
	if err := cs.doGetPeer(liderURL, "/leaderboard", &remotos); err != nil {
		return
	}
	for _, r := range remotos {
		cs.PM.SincronizarJogador(r.JogadorId, r.Nome, r.Vitorias)
	}
}
