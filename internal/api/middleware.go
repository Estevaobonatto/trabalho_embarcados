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

// LiderMiddleware bloqueia escritas em servidores que não são o líder do cluster.
// Conforme Estrutura V2, o cliente recebe o endereço do líder no campo
// "enderecoLider" da resposta de erro para poder redirecionar.
func LiderMiddleware(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Cluster != nil && h.Cluster.HasPeers() && !h.Cluster.IsLeader() {
			enderecoLider := h.Cluster.GetLiderEndereco()
			resp := gin.H{
				"sucesso": false,
				"erro": gin.H{
					"codigo":        model.SERVIDOR_NAO_E_LIDER,
					"mensagem":      "Este servidor nao e o lider. Redirecione para: " + enderecoLider,
					"enderecoLider": enderecoLider,
				},
			}
			c.AbortWithStatusJSON(model.StatusHTTP(model.SERVIDOR_NAO_E_LIDER), resp)
			return
		}
		c.Next()
	}
}
