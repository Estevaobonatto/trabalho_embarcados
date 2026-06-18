package model

// Cor representa as cores do jogo UNO conforme Seção 2 do contrato.
// PRETO é usado exclusivamente para cartas CORINGA e MAIS_QUATRO.
type Cor string

const (
	AMARELO  Cor = "AMARELO"
	AZUL     Cor = "AZUL"
	VERDE    Cor = "VERDE"
	VERMELHO Cor = "VERMELHO"
	PRETO    Cor = "PRETO"
)

// CoresValidas contém todas as cores válidas para validação.
var CoresValidas = []Cor{AMARELO, AZUL, VERDE, VERMELHO, PRETO}

// CoresJogaveis são as cores que um jogador pode escolher ao jogar CORINGA/MAIS_QUATRO.
var CoresJogaveis = []Cor{AMARELO, AZUL, VERDE, VERMELHO}

// CorValida verifica se uma string representa uma cor válida.
func CorValida(c string) bool {
	switch Cor(c) {
	case AMARELO, AZUL, VERDE, VERMELHO, PRETO:
		return true
	}
	return false
}

// TipoCarta representa os tipos de carta do UNO conforme Seção 2 do contrato.
type TipoCarta string

const (
	NUMERICA    TipoCarta = "NUMERICA"
	PULAR       TipoCarta = "PULAR"
	INVERTER    TipoCarta = "INVERTER"
	MAIS_DOIS   TipoCarta = "MAIS_DOIS"
	CORINGA     TipoCarta = "CORINGA"
	MAIS_QUATRO TipoCarta = "MAIS_QUATRO"
)

// TipoCartaValido verifica se uma string representa um tipo de carta válido.
func TipoCartaValido(t string) bool {
	switch TipoCarta(t) {
	case NUMERICA, PULAR, INVERTER, MAIS_DOIS, CORINGA, MAIS_QUATRO:
		return true
	}
	return false
}

// Carta representa uma carta do baralho UNO conforme Seção 3 do contrato.
// Para cartas especiais (PULAR, INVERTER, etc.), o campo valor é null.
// Para cartas CORINGA e MAIS_QUATRO, a cor é sempre PRETO.
type Carta struct {
	ID    string    `json:"id"`
	Cor   Cor       `json:"cor"`
	Tipo  TipoCarta `json:"tipo"`
	Valor *string   `json:"valor"`
}

// PrecisaCorEscolhida retorna true se a carta exige corEscolhida (CORINGA ou MAIS_QUATRO).
func (c *Carta) PrecisaCorEscolhida() bool {
	return c.Tipo == CORINGA || c.Tipo == MAIS_QUATRO
}

// EhNumerica retorna true se a carta é do tipo NUMERICA.
func (c *Carta) EhNumerica() bool {
	return c.Tipo == NUMERICA
}

// EhEspecial retorna true se a carta tem efeito especial (não numérica).
func (c *Carta) EhEspecial() bool {
	return c.Tipo != NUMERICA
}
