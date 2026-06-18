package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"uno-api/internal/game"
	"uno-api/internal/model"
	"uno-api/internal/replication"
)

type Handler struct {
	PM           *game.PartidaManager
	Cluster      *replication.ClusterState
	ServidorID   string
	Endereco     string
}

func NewHandler(pm *game.PartidaManager, cs *replication.ClusterState, servidorID, endereco string) *Handler {
	return &Handler{
		PM:         pm,
		Cluster:    cs,
		ServidorID: servidorID,
		Endereco:   endereco,
	}
}

func SetupRoutes(r *gin.Engine, h *Handler) {
	r.GET("/servidor", h.GetServidor)
	r.GET("/jogos", h.ListarJogos)
	r.GET("/jogos/:gameId/estado", h.GetEstado)
	r.GET("/jogos/:gameId/eventos", h.GetEventos)
	r.GET("/leaderboard", h.GetLeaderboard)

	write := r.Group("")
	write.Use(LiderMiddleware(h))
	write.POST("/jogadores", h.CriarJogador)
	write.POST("/jogos", h.CriarJogo)
	write.POST("/jogos/:gameId/entrar", h.EntrarNaPartida)
	write.POST("/jogos/:gameId/jogarCarta", h.JogarCarta)
	write.POST("/jogos/:gameId/comprar", h.ComprarCarta)
	write.POST("/jogos/:gameId/uno", h.ChamarUno)
	write.POST("/jogos/:gameId/bater", h.Bater)

	repl := r.Group("/_replicacao")
	repl.GET("/jogos", h.ReplicacaoListarJogos)
	repl.GET("/jogos/:gameId", h.ReplicacaoObterJogo)
}

func (h *Handler) ReplicacaoListarJogos(c *gin.Context) {
	ids := h.PM.ListarGameIds()
	type resumo struct {
		GameId       string `json:"gameId"`
		VersaoEstado int    `json:"versaoEstado"`
	}
	lista := make([]resumo, 0, len(ids))
	for _, id := range ids {
		snap, err := h.PM.ExportarSnapshot(id)
		if err != nil {
			continue
		}
		lista = append(lista, resumo{GameId: snap.GameId, VersaoEstado: snap.VersaoEstado})
	}
	resp := model.NovaRespostaSucesso("Lista de jogos para replicacao", lista)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReplicacaoObterJogo(c *gin.Context) {
	gameId := c.Param("gameId")
	snap, err := h.PM.ExportarSnapshot(gameId)
	if err != nil {
		je := converterErro(err)
		resp := model.NovaRespostaErro(je.Codigo, je.Msg)
		c.JSON(model.StatusHTTP(je.Codigo), resp)
		return
	}
	resp := model.NovaRespostaSucesso("Snapshot do jogo", snap)
	c.JSON(http.StatusOK, resp)
}

func converterErro(err error) game.ErroJogo {
	var je *game.ErroJogo
	if errors.As(err, &je) {
		return *je
	}
	return game.ErroJogo{Codigo: model.ERRO_INTERNO, Msg: err.Error()}
}
