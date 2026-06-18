package replication

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"uno-api/internal/game"
	"uno-api/internal/model"
)

type PeerInfo struct {
	URL        string
	ServidorID string
	Active     bool
	LastSeen   time.Time
}

type ClusterState struct {
	mu         sync.RWMutex
	ServidorID string
	Endereco   string
	Peers      map[string]*PeerInfo
	IsLider    bool
	LiderID    string
	EnderecoLider string

	PM         *game.PartidaManager
	httpClient *http.Client
	stopCh     chan struct{}
}

func NewClusterState(servidorID, endereco string, peerURLs []string, pm *game.PartidaManager) *ClusterState {
	cs := &ClusterState{
		ServidorID:    servidorID,
		Endereco:      endereco,
		Peers:         make(map[string]*PeerInfo),
		IsLider:       true,
		LiderID:       servidorID,
		EnderecoLider: endereco,
		PM:            pm,
		httpClient:    &http.Client{Timeout: 3 * time.Second},
		stopCh:        make(chan struct{}),
	}

	for _, url := range peerURLs {
		cs.Peers[url] = &PeerInfo{URL: url, Active: false}
	}

	if len(peerURLs) > 0 {
		go func() {
			time.Sleep(4 * time.Second)
			cs.elegerLiderInicial()
		}()
	}

	return cs
}

func (cs *ClusterState) elegerLiderInicial() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	highestID := cs.ServidorID
	for _, peer := range cs.Peers {
		id := cs.obterServidorId(peer.URL)
		if id != "" {
			peer.ServidorID = id
			peer.Active = true
			peer.LastSeen = time.Now()
			if id > highestID {
				highestID = id
			}
		}
	}

	cs.IsLider = (highestID == cs.ServidorID)
	if cs.IsLider {
		cs.LiderID = cs.ServidorID
		cs.EnderecoLider = cs.Endereco
	} else {
		cs.LiderID = highestID
		for _, peer := range cs.Peers {
			if peer.ServidorID == highestID {
				cs.EnderecoLider = peer.URL
				break
			}
		}
	}
}

func (cs *ClusterState) obterServidorId(peerURL string) string {
	resp, err := cs.httpClient.Get(peerURL + "/servidor")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Dados struct {
			ServidorId string `json:"servidorId"`
		} `json:"dados"`
	}
	if json.Unmarshal(body, &apiResp) == nil {
		return apiResp.Dados.ServidorId
	}
	return ""
}

func (cs *ClusterState) Start() {
	if len(cs.Peers) == 0 {
		return
	}
	go cs.loopHeartbeat()
	go cs.loopSincronizacao()
}

func (cs *ClusterState) Stop() {
	close(cs.stopCh)
}

func (cs *ClusterState) IsLeader() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.IsLider
}

func (cs *ClusterState) GetLiderEndereco() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.EnderecoLider
}

func (cs *ClusterState) GetLiderID() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.LiderID
}

func (cs *ClusterState) GetServidorID() string {
	return cs.ServidorID
}

func (cs *ClusterState) GetEndereco() string {
	return cs.Endereco
}

func (cs *ClusterState) GetVersaoEstadoAtual() int {
	return cs.PM.MaxVersaoEstado()
}

func (cs *ClusterState) GetServidorInfo() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return map[string]interface{}{
		"servidorId":         cs.ServidorID,
		"nome":               "Servidor UNO",
		"versaoContrato":     "1.1",
		"status":             "ATIVO",
		"lider":              cs.IsLider,
		"enderecoLider":      cs.EnderecoLider,
		"versaoEstadoAtual":  cs.PM.MaxVersaoEstado(),
	}
}

func (cs *ClusterState) HasPeers() bool {
	return len(cs.Peers) > 0
}

func (cs *ClusterState) registrarEventoCluster(tipo model.TipoEvento, mensagem string) {
	for _, id := range cs.PM.ListarGameIds() {
		jogo, err := cs.PM.ObterJogo(id)
		if err != nil {
			continue
		}
		jogo.Lock()
		versao := jogo.IncrementarVersao()
		evento := model.NovoEvento(len(jogo.Eventos)+1, tipo, cs.ServidorID, mensagem, versao)
		jogo.Eventos = append(jogo.Eventos, evento)
		jogo.Unlock()
	}
}

func (cs *ClusterState) doGetPeer(peerURL, path string, result interface{}) error {
	resp, err := cs.httpClient.Get(peerURL + path)
	if err != nil {
		return fmt.Errorf("erro de conexao com peer %s: %w", peerURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s retornou status %d", peerURL, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Dados json.RawMessage `json:"dados"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}
	if result != nil && apiResp.Dados != nil {
		return json.Unmarshal(apiResp.Dados, result)
	}
	return nil
}

func (cs *ClusterState) ElegerLiderInicial() {
	cs.elegerLiderInicial()
}

func (cs *ClusterState) SetIsLider(v bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.IsLider = v
}
