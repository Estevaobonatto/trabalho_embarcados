package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APIClient implementa um cliente HTTP com failover automático para o cluster UNO.
// Conforme Estrutura V2:
//   - O jogador sempre tenta falar com o líder primeiro.
//   - Se o líder não responder (timeout/erro de conexão), tenta a réplica.
//   - Se o servidor responder SERVIDOR_NAO_E_LIDER, troca para o endereço indicado.
type APIClient struct {
	Servers    []string   // lista de URLs de servidores (líder primeiro)
	HTTPClient *http.Client
	mu         sync.Mutex // protege o índice do servidor ativo
	activeIdx  int        // índice do servidor atualmente sendo usado
}

// NewAPIClient cria um cliente com failover a partir de uma lista de URLs.
// Se apenas uma URL for fornecida, o comportamento é o mesmo de antes.
func NewAPIClient(baseURLs ...string) *APIClient {
	urls := make([]string, 0, len(baseURLs))
	for _, u := range baseURLs {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		urls = []string{"http://localhost:8080"}
	}
	return &APIClient{
		Servers: urls,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		activeIdx: 0,
	}
}

// ActiveURL retorna a URL do servidor atualmente em uso pelo cliente.
func (c *APIClient) ActiveURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Servers[c.activeIdx]
}

// switchTo troca o servidor ativo para o índice dado.
func (c *APIClient) switchTo(idx int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx >= 0 && idx < len(c.Servers) {
		c.activeIdx = idx
	}
}

// switchToURL troca o servidor ativo para a URL indicada, se ela existir na lista.
func (c *APIClient) switchToURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, s := range c.Servers {
		if s == url {
			c.activeIdx = i
			return
		}
	}
	// Se a URL não está na lista, adiciona ao final (servidor descoberto dinamicamente).
	c.Servers = append(c.Servers, url)
	c.activeIdx = len(c.Servers) - 1
}

// tryNext tenta o próximo servidor da lista. Retorna false se não há mais.
func (c *APIClient) tryNext() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeIdx+1 >= len(c.Servers) {
		return false
	}
	c.activeIdx++
	return true
}

type apiResponse struct {
	Sucesso  bool            `json:"sucesso"`
	Mensagem string          `json:"mensagem"`
	Dados    json.RawMessage `json:"dados"`
	Erro     *apiErro        `json:"erro"`
}

// apiErro inclui o campo enderecoLider que pode vir em respostas de erro
// do tipo SERVIDOR_NAO_E_LIDER para permitir failover do cliente.
type apiErro struct {
	Codigo        string `json:"codigo"`
	Mensagem      string `json:"mensagem"`
	EnderecoLider string `json:"enderecoLider"`
}

func (e *apiErro) Error() string {
	if e.EnderecoLider != "" {
		return fmt.Sprintf("[%s] %s (lider em %s)", e.Codigo, e.Mensagem, e.EnderecoLider)
	}
	return fmt.Sprintf("[%s] %s", e.Codigo, e.Mensagem)
}

// doGet executa GET com failover automático.
func (c *APIClient) doGet(path string, result interface{}) error {
	return c.doRequest("GET", path, nil, result)
}

// doPost executa POST com failover automático.
func (c *APIClient) doPost(path string, body interface{}, result interface{}) error {
	return c.doRequest("POST", path, body, result)
}

// doRequest executa a requisição com failover. Em caso de erro de conexão,
// tenta o próximo servidor da lista. Em caso de SERVIDOR_NAO_E_LIDER,
// troca para o endereço do líder e tenta novamente.
func (c *APIClient) doRequest(method, path string, body interface{}, result interface{}) error {
	maxTentativas := len(c.Servers) + 1
	for i := 0; i < maxTentativas; i++ {
		err := c.tryOnce(method, path, body, result)
		if err == nil {
			return nil
		}

		// Se recebemos SERVIDOR_NAO_E_LIDER com enderecoLider, redireciona.
		var apiErr *apiErro
		if errorsAs(err, &apiErr) && apiErr.Codigo == "SERVIDOR_NAO_E_LIDER" && apiErr.EnderecoLider != "" {
			c.switchToURL(apiErr.EnderecoLider)
			continue
		}

		// Erro de conexão: tenta o próximo servidor.
		if isConnectionError(err) {
			if c.tryNext() {
				continue
			}
		}

		// Outros erros: retorna imediatamente.
		return err
	}
	return fmt.Errorf("falha apos %d tentativas em todos os servidores", maxTentativas)
}

// tryOnce executa uma única requisição sem failover.
func (c *APIClient) tryOnce(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("erro ao serializar: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.ActiveURL() + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("erro ao criar requisicao: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &connectionError{err: err}
	}
	defer resp.Body.Close()
	return c.parseResponse(resp, result)
}

// connectionError marca um erro como sendo de conexão (passível de failover).
type connectionError struct {
	err error
}

func (e *connectionError) Error() string { return e.err.Error() }
func (e *connectionError) Unwrap() error { return e.err }

// isConnectionError retorna true se o erro for de conexão (rede).
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*connectionError)
	return ok
}

// errorsAs é um shim para errors.As (evita importar "errors" no topo).
func errorsAs(err error, target interface{}) bool {
	type wrapper interface{ Unwrap() error }
	for err != nil {
		if apiErr, ok := err.(*apiErro); ok {
			if t, ok := target.(**apiErro); ok {
				*t = apiErr
				return true
			}
		}
		if w, ok := err.(wrapper); ok {
			err = w.Unwrap()
		} else {
			return false
		}
	}
	return false
}

func (c *APIClient) parseResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectionError{err: fmt.Errorf("erro ao ler resposta: %w", err)}
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	if !apiResp.Sucesso && apiResp.Erro != nil {
		return apiResp.Erro
	}

	if result != nil && apiResp.Dados != nil {
		if err := json.Unmarshal(apiResp.Dados, result); err != nil {
			return fmt.Errorf("erro ao decodificar dados: %w", err)
		}
	}

	return nil
}

type JogadorInfo struct {
	JogadorId string `json:"jogadorId"`
	Nome      string `json:"nome"`
}

func (c *APIClient) CriarJogador(nome string) (*JogadorInfo, error) {
	var result JogadorInfo
	err := c.doPost("/jogadores", map[string]string{"nome": nome}, &result)
	return &result, err
}

type JogoInfo struct {
	GameId string `json:"gameId"`
	Status string `json:"status"`
}

func (c *APIClient) CriarJogo(jogadorId string) (*JogoInfo, error) {
	var result JogoInfo
	err := c.doPost("/jogos", map[string]string{"jogadorId": jogadorId}, &result)
	return &result, err
}

type ResumoJogo struct {
	GameId              string `json:"gameId"`
	Status              string `json:"status"`
	QuantidadeJogadores int    `json:"quantidadeJogadores"`
	MaxJogadores        int    `json:"maxJogadores"`
}

func (c *APIClient) ListarJogos() ([]ResumoJogo, error) {
	var result []ResumoJogo
	err := c.doGet("/jogos", &result)
	return result, err
}

type EntrarInfo struct {
	GameId              string `json:"gameId"`
	Status              string `json:"status"`
	QuantidadeJogadores int    `json:"quantidadeJogadores"`
}

func (c *APIClient) EntrarNaPartida(gameId, jogadorId string) (*EntrarInfo, error) {
	var result EntrarInfo
	err := c.doPost("/jogos/"+gameId+"/entrar", map[string]string{"jogadorId": jogadorId}, &result)
	return &result, err
}

type CartaInfo struct {
	ID    string  `json:"id"`
	Cor   string  `json:"cor"`
	Tipo  string  `json:"tipo"`
	Valor *string `json:"valor"`
}

type JogadorPublicoInfo struct {
	JogadorId        string `json:"jogadorId"`
	Nome             string `json:"nome"`
	QuantidadeCartas int    `json:"quantidadeCartas"`
	ChamouUno        bool   `json:"chamouUno"`
}

type EstadoJogo struct {
	GameId       string               `json:"gameId"`
	Status       string               `json:"status"`
	VersaoEstado int                  `json:"versaoEstado"`
	JogadorDaVez string               `json:"jogadorDaVez"`
	Sentido      string               `json:"sentido"`
	CorAtual     string               `json:"corAtual"`
	CartaTopo    *CartaInfo           `json:"cartaTopo"`
	MinhaMao     []CartaInfo          `json:"minhaMao"`
	Jogadores    []JogadorPublicoInfo `json:"jogadores"`
	Vencedor     *string              `json:"vencedor"`
}

func (c *APIClient) ObterEstado(gameId, jogadorId string) (*EstadoJogo, error) {
	var result EstadoJogo
	path := fmt.Sprintf("/jogos/%s/estado?jogadorId=%s", gameId, jogadorId)
	err := c.doGet(path, &result)
	return &result, err
}

type ResultadoJogada struct {
	GameId         string `json:"gameId"`
	VersaoEstado   int    `json:"versaoEstado"`
	ProximoJogador string `json:"proximoJogador"`
	CorAtual       string `json:"corAtual"`
}

func (c *APIClient) JogarCarta(gameId, jogadorId, cartaId string, corEscolhida *string) (*ResultadoJogada, error) {
	body := map[string]interface{}{
		"jogadorId":    jogadorId,
		"cartaId":      cartaId,
		"corEscolhida": corEscolhida,
	}
	var result ResultadoJogada
	err := c.doPost("/jogos/"+gameId+"/jogarCarta", body, &result)
	return &result, err
}

type ResultadoCompra struct {
	CartaComprada  CartaInfo `json:"cartaComprada"`
	PassouAVez     bool      `json:"passouAVez"`
	ProximoJogador string    `json:"proximoJogador"`
}

func (c *APIClient) ComprarCarta(gameId, jogadorId string) (*ResultadoCompra, error) {
	var result ResultadoCompra
	err := c.doPost("/jogos/"+gameId+"/comprar", map[string]string{"jogadorId": jogadorId}, &result)
	return &result, err
}

type UnoResult struct {
	JogadorId        string `json:"jogadorId"`
	QuantidadeCartas int    `json:"quantidadeCartas"`
}

func (c *APIClient) ChamarUno(gameId, jogadorId string) (*UnoResult, error) {
	var result UnoResult
	err := c.doPost("/jogos/"+gameId+"/uno", map[string]string{"jogadorId": jogadorId}, &result)
	return &result, err
}

type BaterResult struct {
	Vencedor string `json:"vencedor"`
	Status   string `json:"status"`
}

func (c *APIClient) Bater(gameId, jogadorId string) (*BaterResult, error) {
	var result BaterResult
	err := c.doPost("/jogos/"+gameId+"/bater", map[string]string{"jogadorId": jogadorId}, &result)
	return &result, err
}

type EventoInfo struct {
	Sequencia    int    `json:"sequencia"`
	Tipo         string `json:"tipo"`
	JogadorId    string `json:"jogadorId"`
	Mensagem     string `json:"mensagem"`
	VersaoEstado int    `json:"versaoEstado"`
}

func (c *APIClient) ObterEventos(gameId string, desde int) ([]EventoInfo, error) {
	var result []EventoInfo
	path := fmt.Sprintf("/jogos/%s/eventos?desde=%d", gameId, desde)
	err := c.doGet(path, &result)
	return result, err
}

type RankingInfo struct {
	JogadorId string `json:"jogadorId"`
	Nome      string `json:"nome"`
	Vitorias  int    `json:"vitorias"`
}

func (c *APIClient) ObterLeaderboard() ([]RankingInfo, error) {
	var result []RankingInfo
	err := c.doGet("/leaderboard", &result)
	return result, err
}

type ServidorInfo struct {
	ServidorId        string `json:"servidorId"`
	Nome              string `json:"nome"`
	VersaoContrato    string `json:"versaoContrato"`
	Status            string `json:"status"`
	Lider             bool   `json:"lider"`
	EnderecoLider     string `json:"enderecoLider"`
	VersaoEstadoAtual int    `json:"versaoEstadoAtual"`
}

func (c *APIClient) GetServidor() (*ServidorInfo, error) {
	var result ServidorInfo
	err := c.doGet("/servidor", &result)
	return &result, err
}
