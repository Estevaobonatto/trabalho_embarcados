package model

// Jogador representa um jogador do UNO conforme Seção 7.2 do contrato.
// A mão do jogador é privada e nunca deve ser exposta a outros jogadores.
type Jogador struct {
	JogadorId string  `json:"jogadorId"`
	Nome      string  `json:"nome"`
	Vitorias  int     `json:"vitorias"`
	Mao       []Carta `json:"-"`         // mão privada — nunca exposta a outros
	ChamouUno bool    `json:"chamouUno"` // true se chamou UNO, reset após jogar ou comprar
}

// JogadorPublico representa a visão pública de um jogador,
// sem expor a mão — apenas quantidade de cartas.
type JogadorPublico struct {
	JogadorId        string `json:"jogadorId"`
	Nome             string `json:"nome"`
	QuantidadeCartas int    `json:"quantidadeCartas"`
	ChamouUno        bool   `json:"chamouUno"`
}

// ParaJogadorPublico converte um Jogador interno para a visão pública.
func (j *Jogador) ParaJogadorPublico() JogadorPublico {
	return JogadorPublico{
		JogadorId:        j.JogadorId,
		Nome:             j.Nome,
		QuantidadeCartas: len(j.Mao),
		ChamouUno:        j.ChamouUno,
	}
}

// TemCarta verifica se o jogador possui uma carta específica na mão.
func (j *Jogador) TemCarta(cartaId string) bool {
	for _, c := range j.Mao {
		if c.ID == cartaId {
			return true
		}
	}
	return false
}

// RemoverCarta remove uma carta da mão do jogador pelo ID.
// Retorna a carta removida e true se encontrada, ou Carta{} e false.
func (j *Jogador) RemoverCarta(cartaId string) (Carta, bool) {
	for i, c := range j.Mao {
		if c.ID == cartaId {
			j.Mao = append(j.Mao[:i], j.Mao[i+1:]...)
			return c, true
		}
	}
	return Carta{}, false
}

// ObterCarta retorna a carta da mão pelo ID sem removê-la.
func (j *Jogador) ObterCarta(cartaId string) (Carta, bool) {
	for _, c := range j.Mao {
		if c.ID == cartaId {
			return c, true
		}
	}
	return Carta{}, false
}
