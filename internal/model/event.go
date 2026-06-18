package model

// TipoEvento representa os tipos de evento do jogo conforme Seção 2 do contrato.
type TipoEvento string

const (
	JOGADOR_ENTROU     TipoEvento = "JOGADOR_ENTROU"
	JOGO_INICIADO      TipoEvento = "JOGO_INICIADO"
	CARTA_JOGADA       TipoEvento = "CARTA_JOGADA"
	CARTA_COMPRADA     TipoEvento = "CARTA_COMPRADA"
	UNO_CHAMADO        TipoEvento = "UNO_CHAMADO"
	PENALIDADE_UNO     TipoEvento = "PENALIDADE_UNO"
	JOGADOR_BATEU      TipoEvento = "JOGADOR_BATEU"
	PARTIDA_FINALIZADA TipoEvento = "JOGO_FINALIZADO"
	FAILOVER           TipoEvento = "FAILOVER"
	LIDER_ALTERADO     TipoEvento = "LIDER_ALTERADO"
)

// Evento representa um evento de jogo conforme Seção 7.11 do contrato.
// Eventos são a fonte da verdade para replicação de estado.
type Evento struct {
	Sequencia    int        `json:"sequencia"`
	Tipo         TipoEvento `json:"tipo"`
	JogadorId    string     `json:"jogadorId"`
	Mensagem     string     `json:"mensagem"`
	VersaoEstado int        `json:"versaoEstado"`
}

// NovoEvento cria um novo evento com os dados fornecidos.
func NovoEvento(sequencia int, tipo TipoEvento, jogadorId, mensagem string, versaoEstado int) Evento {
	return Evento{
		Sequencia:    sequencia,
		Tipo:         tipo,
		JogadorId:    jogadorId,
		Mensagem:     mensagem,
		VersaoEstado: versaoEstado,
	}
}
