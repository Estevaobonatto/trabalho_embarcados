package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type apiResponse struct {
	Sucesso  bool            `json:"sucesso"`
	Mensagem string          `json:"mensagem"`
	Dados    json.RawMessage `json:"dados"`
	Erro     *apiErro        `json:"erro"`
}

type apiErro struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

func (e *apiErro) Error() string {
	return fmt.Sprintf("[%s] %s", e.Codigo, e.Mensagem)
}

func (c *APIClient) doGet(path string, result interface{}) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("erro de conexao: %w", err)
	}
	defer resp.Body.Close()
	return c.parseResponse(resp, result)
}

func (c *APIClient) doPost(path string, body interface{}, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("erro ao serializar: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+path, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("erro de conexao: %w", err)
	}
	defer resp.Body.Close()
	return c.parseResponse(resp, result)
}

func (c *APIClient) parseResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler resposta: %w", err)
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
	ServidorId         string `json:"servidorId"`
	Nome               string `json:"nome"`
	VersaoContrato     string `json:"versaoContrato"`
	Status             string `json:"status"`
	Lider              bool   `json:"lider"`
	EnderecoLider      string `json:"enderecoLider"`
	VersaoEstadoAtual  int    `json:"versaoEstadoAtual"`
}

func (c *APIClient) GetServidor() (*ServidorInfo, error) {
	var result ServidorInfo
	err := c.doGet("/servidor", &result)
	return &result, err
}
