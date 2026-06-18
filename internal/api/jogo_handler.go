package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/game"
	"uno-api/internal/model"
)

func (h *Handler) CriarJogo(c *gin.Context) {
	var req struct {
		JogadorId string `json:"jogadorId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	jogo, err := h.PM.CriarPartida(req.JogadorId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"gameId": jogo.GameId,
		"status": jogo.Status,
	}
	resp := model.NovaRespostaSucesso("Jogo criado com sucesso", dados)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListarJogos(c *gin.Context) {
	jogos := h.PM.ListarJogos()
	if jogos == nil {
		jogos = make([]game.ResumoJogo, 0)
	}
	resp := model.NovaRespostaSucesso("Lista de jogos", jogos)
	c.JSON(http.StatusOK, resp)
}
