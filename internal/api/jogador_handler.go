package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/model"
)

func (h *Handler) CriarJogador(c *gin.Context) {
	var req struct {
		Nome string `json:"nome"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := model.NovaRespostaErro(model.NOME_INVALIDO, "")
		c.JSON(model.StatusHTTP(model.NOME_INVALIDO), resp)
		return
	}

	jogador, err := h.PM.CriarJogador(req.Nome)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}

	dados := gin.H{
		"jogadorId": jogador.JogadorId,
		"nome":      jogador.Nome,
	}
	resp := model.NovaRespostaSucesso("Jogador criado com sucesso", dados)
	c.JSON(http.StatusOK, resp)
}
