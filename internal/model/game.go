package model

import "sync"

// StatusPartida representa o status de uma partida conforme Seção 2 do contrato.
type StatusPartida string

const (
	AGUARDANDO_JOGADORES StatusPartida = "AGUARDANDO_JOGADORES"
	EM_ANDAMENTO         StatusPartida = "EM_ANDAMENTO"
	FINALIZADO           StatusPartida = "FINALIZADO"
	CANCELADO            StatusPartida = "CANCELADO"
)

// Sentido representa o sentido da rodada conforme Seção 2 do contrato.
type Sentido string

const (
	HORARIO      Sentido = "HORARIO"
	ANTI_HORARIO Sentido = "ANTI_HORARIO"
)

// MaxJogadoresPorPartida é o número máximo de jogadores por partida.
const MaxJogadoresPorPartida = 4

// CartasIniciaisPorJogador é o número de cartas distribuídas no início da partida.
const CartasIniciaisPorJogador = 7

// Jogo representa o estado interno completo de uma partida,
// conforme Seção 9 do contrato.
// Nem todo este estado é exposto ao cliente.
type Jogo struct {
	mu sync.RWMutex `json:"-"` // mutex para acesso concorrente

	GameId        string        `json:"gameId"`
	Status        StatusPartida `json:"status"`
	VersaoEstado  int           `json:"versaoEstado"`
	Jogadores     []*Jogador    `json:"jogadores"`
	JogadorDaVez  string        `json:"jogadorDaVez"`
	Sentido       Sentido       `json:"sentido"`
	CorAtual      Cor           `json:"corAtual"`
	CartaTopo     *Carta        `json:"cartaTopo"`
	MonteCompra   []Carta       `json:"monteCompra"`
	MonteDescarte []Carta       `json:"monteDescarte"`
	Eventos       []Evento      `json:"eventos"`
	Vencedor      *string       `json:"vencedor"`
}

// NewJogo cria um novo Jogo com valores iniciais.
func NewJogo(gameId string) *Jogo {
	return &Jogo{
		GameId:        gameId,
		Status:        AGUARDANDO_JOGADORES,
		VersaoEstado:  0,
		Jogadores:     make([]*Jogador, 0, MaxJogadoresPorPartida),
		Sentido:       HORARIO,
		MonteCompra:   make([]Carta, 0),
		MonteDescarte: make([]Carta, 0),
		Eventos:       make([]Evento, 0),
	}
}

// Lock adquire lock de escrita no mutex do jogo.
func (j *Jogo) Lock() {
	j.mu.Lock()
}

// Unlock libera lock de escrita no mutex do jogo.
func (j *Jogo) Unlock() {
	j.mu.Unlock()
}

// RLock adquire lock de leitura no mutex do jogo.
func (j *Jogo) RLock() {
	j.mu.RLock()
}

// RUnlock libera lock de leitura no mutex do jogo.
func (j *Jogo) RUnlock() {
	j.mu.RUnlock()
}

// EstaCheio retorna true se o jogo atingiu o máximo de jogadores.
func (j *Jogo) EstaCheio() bool {
	return len(j.Jogadores) >= MaxJogadoresPorPartida
}

// EstaIniciado retorna true se o jogo não está mais aguardando jogadores.
// Inclui EM_ANDAMENTO, FINALIZADO e CANCELADO.
func (j *Jogo) EstaIniciado() bool {
	return j.Status != AGUARDANDO_JOGADORES
}

// EstaFinalizado retorna true se o jogo terminou.
func (j *Jogo) EstaFinalizado() bool {
	return j.Status == FINALIZADO || j.Status == CANCELADO
}

// BuscarJogador retorna o jogador pelo ID, ou nil se não encontrado.
func (j *Jogo) BuscarJogador(jogadorId string) *Jogador {
	for _, jog := range j.Jogadores {
		if jog.JogadorId == jogadorId {
			return jog
		}
	}
	return nil
}

// IndiceJogador retorna o índice do jogador na lista, ou -1 se não encontrado.
func (j *Jogo) IndiceJogador(jogadorId string) int {
	for i, jog := range j.Jogadores {
		if jog.JogadorId == jogadorId {
			return i
		}
	}
	return -1
}

// IncrementarVersao incrementa a versão do estado e retorna o novo valor.
func (j *Jogo) IncrementarVersao() int {
	j.VersaoEstado++
	return j.VersaoEstado
}

// QuantidadeJogadores retorna o número atual de jogadores.
func (j *Jogo) QuantidadeJogadores() int {
	return len(j.Jogadores)
}
