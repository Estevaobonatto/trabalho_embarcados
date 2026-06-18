package model

type JogadorSnapshot struct {
	JogadorId string  `json:"jogadorId"`
	Nome      string  `json:"nome"`
	Vitorias  int     `json:"vitorias"`
	Mao       []Carta `json:"mao"`
	ChamouUno bool    `json:"chamouUno"`
}

type JogoSnapshot struct {
	GameId        string            `json:"gameId"`
	Status        string            `json:"status"`
	VersaoEstado  int               `json:"versaoEstado"`
	Jogadores     []JogadorSnapshot `json:"jogadores"`
	JogadorDaVez  string            `json:"jogadorDaVez"`
	Sentido       string            `json:"sentido"`
	CorAtual      string            `json:"corAtual"`
	CartaTopo     *Carta            `json:"cartaTopo"`
	MonteCompra   []Carta           `json:"monteCompra"`
	MonteDescarte []Carta           `json:"monteDescarte"`
	Eventos       []Evento          `json:"eventos"`
	Vencedor      *string           `json:"vencedor"`
}

func JogoParaSnapshot(j *Jogo) *JogoSnapshot {
	jogadores := make([]JogadorSnapshot, len(j.Jogadores))
	for i, jog := range j.Jogadores {
		mao := make([]Carta, len(jog.Mao))
		copy(mao, jog.Mao)
		jogadores[i] = JogadorSnapshot{
			JogadorId: jog.JogadorId,
			Nome:      jog.Nome,
			Vitorias:  jog.Vitorias,
			Mao:       mao,
			ChamouUno: jog.ChamouUno,
		}
	}
	return &JogoSnapshot{
		GameId:        j.GameId,
		Status:        string(j.Status),
		VersaoEstado:  j.VersaoEstado,
		Jogadores:     jogadores,
		JogadorDaVez:  j.JogadorDaVez,
		Sentido:       string(j.Sentido),
		CorAtual:      string(j.CorAtual),
		CartaTopo:     j.CartaTopo,
		MonteCompra:   j.MonteCompra,
		MonteDescarte: j.MonteDescarte,
		Eventos:       j.Eventos,
		Vencedor:      j.Vencedor,
	}
}
