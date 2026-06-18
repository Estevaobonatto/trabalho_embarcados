package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"uno-api/internal/game"
	"uno-api/internal/model"
)

func (h *Handler) GetEstado(c *gin.Context) {
	gameId := c.Param("gameId")
	jogadorId := c.Query("jogadorId")

	if jogadorId == "" {
		resp := model.NovaRespostaErro(model.JOGADOR_NAO_ENCONTRADO, "")
		c.JSON(model.StatusHTTP(model.JOGADOR_NAO_ENCONTRADO), resp)
		return
	}

	estado, err := h.PM.ObterEstadoPublico(gameId, jogadorId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	resp := model.NovaRespostaSucesso("Estado da partida", estado)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetEventos(c *gin.Context) {
	gameId := c.Param("gameId")
	desdeStr := c.DefaultQuery("desde", "0")

	desde, err := strconv.Atoi(desdeStr)
	if err != nil {
		desde = 0
	}

	eventos, err := h.PM.ObterEventos(gameId, desde)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	if eventos == nil {
		eventos = make([]model.Evento, 0)
	}

	resp := model.NovaRespostaSucesso("Eventos da partida", eventos)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetLeaderboard(c *gin.Context) {
	ranking := h.PM.ObterLeaderboard()
	if ranking == nil {
		ranking = make([]game.JogadorRanking, 0)
	}
	resp := model.NovaRespostaSucesso("Leaderboard", ranking)
	c.JSON(http.StatusOK, resp)
}
