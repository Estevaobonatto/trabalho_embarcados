package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"uno-api/internal/game"
	"uno-api/internal/replication"
)

func setupTestServer() (*gin.Engine, *Handler, *game.PartidaManager) {
	gin.SetMode(gin.TestMode)
	pm := game.NewPartidaManager()
	cs := replication.NewClusterState("srv-test", "http://localhost:0", nil, pm)
	h := NewHandler(pm, cs, "srv-test", "http://localhost:0")

	r := gin.New()
	r.Use(RecoveryMiddleware())
	SetupRoutes(r, h)
	return r, h, pm
}

func doRequest(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao parsear resposta: %v", err)
	}
	return resp
}

func TestGetServidor(t *testing.T) {
	r, _, _ := setupTestServer()
	w := doRequest(t, r, "GET", "/servidor", nil)

	if w.Code != 200 {
		t.Errorf("status esperado 200, veio %d", w.Code)
	}
	resp := parseResp(t, w)
	if resp["sucesso"] != true {
		t.Error("sucesso devia ser true")
	}
	dados := resp["dados"].(map[string]interface{})
	if dados["servidorId"] != "srv-test" {
		t.Errorf("servidorId errado: %v", dados["servidorId"])
	}
	if dados["versaoContrato"] != "1.1" {
		t.Errorf("versaoContrato errada: %v", dados["versaoContrato"])
	}
}

func TestCriarJogador(t *testing.T) {
	r, _, _ := setupTestServer()
	w := doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})

	if w.Code != 200 {
		t.Errorf("status esperado 200, veio %d", w.Code)
	}
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["jogadorId"] != "jogador-001" {
		t.Errorf("jogadorId errado: %v", dados["jogadorId"])
	}
	if dados["nome"] != "Ana" {
		t.Errorf("nome errado: %v", dados["nome"])
	}
}

func TestCriarJogadorNomeVazio(t *testing.T) {
	r, _, _ := setupTestServer()
	w := doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": ""})

	if w.Code != 400 {
		t.Errorf("status esperado 400, veio %d", w.Code)
	}
	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	if erro["codigo"] != "NOME_INVALIDO" {
		t.Errorf("codigo erro errado: %v", erro["codigo"])
	}
}

func TestCriarJogo(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	w := doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	if w.Code != 200 {
		t.Errorf("status esperado 200, veio %d", w.Code)
	}
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["gameId"] != "jogo-001" {
		t.Errorf("gameId errado: %v", dados["gameId"])
	}
	if dados["status"] != "AGUARDANDO_JOGADORES" {
		t.Errorf("status errado: %v", dados["status"])
	}
}

func TestCriarJogoJogadorInexistente(t *testing.T) {
	r, _, _ := setupTestServer()
	w := doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-999"})

	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	if erro["codigo"] != "JOGADOR_NAO_ENCONTRADO" {
		t.Errorf("codigo erro errado: %v", erro["codigo"])
	}
}

func TestListarJogos(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	w := doRequest(t, r, "GET", "/jogos", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].([]interface{})
	if len(dados) != 1 {
		t.Errorf("esperado 1 jogo, tem %d", len(dados))
	}
}

func TestEntrarNaPartida(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Bruno"})
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	w := doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["quantidadeJogadores"].(float64) != 2 {
		t.Errorf("quantidadeJogadores errado: %v", dados["quantidadeJogadores"])
	}
}

func TestEntrarPartidaInexistente(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	w := doRequest(t, r, "POST", "/jogos/jogo-999/entrar", map[string]string{"jogadorId": "jogador-001"})

	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	if erro["codigo"] != "JOGO_NAO_ENCONTRADO" {
		t.Errorf("codigo erro errado: %v", erro["codigo"])
	}
}

func TestGetEstado(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	w := doRequest(t, r, "GET", "/jogos/jogo-001/estado?jogadorId=jogador-001", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["gameId"] != "jogo-001" {
		t.Errorf("gameId errado: %v", dados["gameId"])
	}
}

func TestGetEstadoSemJogadorId(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	w := doRequest(t, r, "GET", "/jogos/jogo-001/estado", nil)
	if w.Code != 404 {
		t.Errorf("status esperado 404, veio %d", w.Code)
	}
}

func TestJogarCartaFluxoCompleto(t *testing.T) {
	r, _, _ := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	w := doRequest(t, r, "GET", "/jogos/jogo-001/estado?jogadorId=jogador-001", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})

	if dados["status"] != "EM_ANDAMENTO" {
		t.Fatalf("jogo nao iniciou: status=%v", dados["status"])
	}

	mao := dados["minhaMao"].([]interface{})
	if len(mao) != 7 {
		t.Errorf("jogador devia ter 7 cartas, tem %d", len(mao))
	}

	if dados["jogadorDaVez"] != "jogador-001" {
		t.Errorf("jogadorDaVez errado: %v", dados["jogadorDaVez"])
	}
}

func TestComprarCartaEndpoint(t *testing.T) {
	r, _, _ := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	w := doRequest(t, r, "POST", "/jogos/jogo-001/comprar", map[string]string{"jogadorId": "jogador-001"})
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})

	if dados["passouAVez"] != true {
		t.Error("passouAVez devia ser true")
	}
	if dados["proximoJogador"] == "jogador-001" {
		t.Error("proximoJogador nao devia ser o mesmo")
	}
	cartaComprada := dados["cartaComprada"].(map[string]interface{})
	if cartaComprada["id"] == nil {
		t.Error("cartaComprada sem id")
	}
}

func TestUnoEndpoint(t *testing.T) {
	r, _, pm := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	jogo, _ := pm.ObterJogo("jogo-001")
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogador := jogo.BuscarJogador(vez)
	jogo.RUnlock()

	jogo.Lock()
	for len(jogador.Mao) > 1 {
		jogador.Mao = jogador.Mao[:len(jogador.Mao)-1]
	}
	jogo.Unlock()

	w := doRequest(t, r, "POST", "/jogos/jogo-001/uno", map[string]string{"jogadorId": vez})
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["quantidadeCartas"].(float64) != 1 {
		t.Errorf("quantidadeCartas devia ser 1, veio %v", dados["quantidadeCartas"])
	}
}

func TestBaterEndpoint(t *testing.T) {
	r, _, pm := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	jogo, _ := pm.ObterJogo("jogo-001")
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogador := jogo.BuscarJogador(vez)
	jogo.RUnlock()

	jogo.Lock()
	jogador.Mao = nil
	jogo.Unlock()

	w := doRequest(t, r, "POST", "/jogos/jogo-001/bater", map[string]string{"jogadorId": vez})
	resp := parseResp(t, w)
	dados := resp["dados"].(map[string]interface{})
	if dados["status"] != "FINALIZADO" {
		t.Errorf("status devia ser FINALIZADO, veio %v", dados["status"])
	}
	if dados["vencedor"] != vez {
		t.Errorf("vencedor errado: %v", dados["vencedor"])
	}
}

func TestEventosEndpoint(t *testing.T) {
	r, _, _ := setupTestServer()

	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})

	w := doRequest(t, r, "GET", "/jogos/jogo-001/eventos", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].([]interface{})
	if len(dados) < 1 {
		t.Error("devia ter pelo menos 1 evento (JOGADOR_ENTROU)")
	}
}

func TestEventosComDesde(t *testing.T) {
	r, _, _ := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	w := doRequest(t, r, "GET", "/jogos/jogo-001/eventos?desde=2", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].([]interface{})
	if len(dados) == 0 {
		t.Error("devia ter eventos apos sequencia 2")
	}
}

func TestLeaderboardEndpoint(t *testing.T) {
	r, _, _ := setupTestServer()
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Ana"})
	doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Bruno"})

	w := doRequest(t, r, "GET", "/leaderboard", nil)
	resp := parseResp(t, w)
	dados := resp["dados"].([]interface{})
	if len(dados) != 2 {
		t.Errorf("esperado 2 jogadores, tem %d", len(dados))
	}
}

func TestJogarCartaSemJogador(t *testing.T) {
	r, _, _ := setupTestServer()
	w := doRequest(t, r, "POST", "/jogos/jogo-001/jogarCarta", map[string]interface{}{
		"jogadorId": "jogador-999",
		"cartaId":   "carta-001",
	})
	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	if erro["codigo"] != "JOGO_NAO_ENCONTRADO" {
		t.Errorf("codigo erro errado: %v", erro["codigo"])
	}
}

func TestEntrarPartidaCheia(t *testing.T) {
	r, _, _ := setupTestServer()

	nomes := []string{"Ana", "Bruno", "Carla", "Daniel", "Extra"}
	for _, nome := range nomes {
		doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": nome})
	}
	doRequest(t, r, "POST", "/jogos", map[string]string{"jogadorId": "jogador-001"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-002"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-003"})
	doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-004"})

	w := doRequest(t, r, "POST", "/jogos/jogo-001/entrar", map[string]string{"jogadorId": "jogador-005"})
	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	codigo := erro["codigo"].(string)
	if codigo != "JOGO_JA_INICIADO" && codigo != "JOGO_CHEIO" {
		t.Errorf("codigo erro esperado JOGO_JA_INICIADO ou JOGO_CHEIO, veio %s", codigo)
	}
}

func TestLiderRedirect(t *testing.T) {
	r, h, _ := setupTestServer()
	cs := replication.NewClusterState("srv-low", "http://localhost:0", []string{"http://localhost:9999"}, h.PM)
	h.Cluster = cs
	cs.ElegerLiderInicial()
	cs.SetIsLider(false)

	w := doRequest(t, r, "POST", "/jogadores", map[string]string{"nome": "Teste"})
	resp := parseResp(t, w)
	erro := resp["erro"].(map[string]interface{})
	if erro["codigo"] != "SERVIDOR_NAO_E_LIDER" {
		t.Errorf("esperado SERVIDOR_NAO_E_LIDER, veio %v", erro["codigo"])
	}
}
