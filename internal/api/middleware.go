package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/model"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				resp := model.NovaRespostaErro(model.ERRO_INTERNO, "Ocorreu um erro interno no servidor.")
				c.AbortWithStatusJSON(model.StatusHTTP(model.ERRO_INTERNO), resp)
			}
		}()
		c.Next()
	}
}

func LiderMiddleware(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Cluster != nil && h.Cluster.HasPeers() && !h.Cluster.IsLeader() {
			enderecoLider := h.Cluster.GetLiderEndereco()
			resp := model.NovaRespostaErro(model.SERVIDOR_NAO_E_LIDER,
				"Este servidor nao e o lider. Lider: "+enderecoLider)
			c.AbortWithStatusJSON(model.StatusHTTP(model.SERVIDOR_NAO_E_LIDER), resp)
			return
		}
		c.Next()
	}
}
