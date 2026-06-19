package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/model"
)

func (h *Handler) GetServidor(c *gin.Context) {
	var dados gin.H
	if h.Cluster != nil && h.Cluster.HasPeers() {
		dados = gin.H(h.Cluster.GetServidorInfo())
	} else {
		dados = gin.H{
			"servidorId":        h.ServidorID,
			"nome":              "Servidor UNO",
			"versaoContrato":    "2.0",
			"status":            "ATIVO",
			"lider":             true,
			"enderecoLider":     h.Endereco,
			"versaoEstadoAtual": h.PM.MaxVersaoEstado(),
		}
	}
	resp := model.NovaRespostaSucesso("Informacoes do servidor", dados)
	c.JSON(http.StatusOK, resp)
}
