package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/model"
)

type entrarRequest struct {
	JogadorId string `json:"jogadorId"`
}

func (h *Handler) EntrarNaPartida(c *gin.Context) {
	gameId := c.Param("gameId")
	var req entrarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	jogo, err := h.PM.EntrarNaPartida(gameId, req.JogadorId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"gameId":              jogo.GameId,
		"status":              string(jogo.Status),
		"quantidadeJogadores": len(jogo.Jogadores),
	}
	resp := model.NovaRespostaSucesso("Jogador entrou na partida", dados)
	c.JSON(http.StatusOK, resp)
}

type jogarCartaRequest struct {
	JogadorId    string  `json:"jogadorId"`
	CartaId      string  `json:"cartaId"`
	CorEscolhida *string `json:"corEscolhida"`
}

func (h *Handler) JogarCarta(c *gin.Context) {
	gameId := c.Param("gameId")
	var req jogarCartaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	var corEscolhida *model.Cor
	if req.CorEscolhida != nil {
		c := model.Cor(*req.CorEscolhida)
		corEscolhida = &c
	}

	resultado, err := h.PM.JogarCarta(gameId, req.JogadorId, req.CartaId, corEscolhida)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"gameId":         resultado.GameId,
		"versaoEstado":   resultado.VersaoEstado,
		"proximoJogador": resultado.ProximoJogador,
		"corAtual":       resultado.CorAtual,
	}
	resp := model.NovaRespostaSucesso("Carta jogada com sucesso", dados)
	c.JSON(http.StatusOK, resp)
}

type comprarRequest struct {
	JogadorId string `json:"jogadorId"`
}

func (h *Handler) ComprarCarta(c *gin.Context) {
	gameId := c.Param("gameId")
	var req comprarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	resultado, err := h.PM.ComprarCarta(gameId, req.JogadorId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"cartaComprada":  resultado.CartaComprada,
		"passouAVez":     resultado.PassouAVez,
		"proximoJogador": resultado.ProximoJogador,
	}
	resp := model.NovaRespostaSucesso("Carta comprada com sucesso", dados)
	c.JSON(http.StatusOK, resp)
}

type unoRequest struct {
	JogadorId string `json:"jogadorId"`
}

func (h *Handler) ChamarUno(c *gin.Context) {
	gameId := c.Param("gameId")
	var req unoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	if err := h.PM.ChamarUno(gameId, req.JogadorId); err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"jogadorId":        req.JogadorId,
		"quantidadeCartas": 1,
	}
	resp := model.NovaRespostaSucesso("UNO chamado com sucesso", dados)
	c.JSON(http.StatusOK, resp)
}

type baterRequest struct {
	JogadorId string `json:"jogadorId"`
}

func (h *Handler) Bater(c *gin.Context) {
	gameId := c.Param("gameId")
	var req baterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	resultado, err := h.PM.Bater(gameId, req.JogadorId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"vencedor": resultado.Vencedor,
		"status":   resultado.Status,
	}
	resp := model.NovaRespostaSucesso("Jogador bateu", dados)
	c.JSON(http.StatusOK, resp)
}
