package game

import (
	"fmt"
	"math/rand"
	"strconv"

	"uno-api/internal/model"
)

// CriarBaralho gera as 108 cartas do baralho UNO padrão.
// Composição:
//   - 1× "0" de cada cor (4)
//   - 2× cada número de 1 a 9 de cada cor (72)
//   - 2× PULAR de cada cor (8)
//   - 2× INVERTER de cada cor (8)
//   - 2× MAIS_DOIS de cada cor (8)
//   - 4× CORINGA (PRETO) (4)
//   - 4× MAIS_QUATRO (PRETO) (4)
//
// Total: 108 cartas. Os IDs são gerados usando o contadorCarta do PartidaManager.
func (pm *PartidaManager) CriarBaralho() []model.Carta {
	cartas := make([]model.Carta, 0, 108)
	cores := []model.Cor{model.AMARELO, model.AZUL, model.VERDE, model.VERMELHO}

	for _, cor := range cores {
		valorZero := "0"
		cartas = append(cartas, pm.novaCarta(cor, model.NUMERICA, &valorZero))

		for v := 1; v <= 9; v++ {
			val := strconv.Itoa(v)
			cartas = append(cartas, pm.novaCarta(cor, model.NUMERICA, &val))
			cartas = append(cartas, pm.novaCarta(cor, model.NUMERICA, &val))
		}

		cartas = append(cartas, pm.novaCarta(cor, model.PULAR, nil))
		cartas = append(cartas, pm.novaCarta(cor, model.PULAR, nil))

		cartas = append(cartas, pm.novaCarta(cor, model.INVERTER, nil))
		cartas = append(cartas, pm.novaCarta(cor, model.INVERTER, nil))

		cartas = append(cartas, pm.novaCarta(cor, model.MAIS_DOIS, nil))
		cartas = append(cartas, pm.novaCarta(cor, model.MAIS_DOIS, nil))
	}

	for range 4 {
		cartas = append(cartas, pm.novaCarta(model.PRETO, model.CORINGA, nil))
	}

	for range 4 {
		cartas = append(cartas, pm.novaCarta(model.PRETO, model.MAIS_QUATRO, nil))
	}

	return cartas
}

// novaCarta cria uma carta com ID único gerado sequencialmente.
func (pm *PartidaManager) novaCarta(cor model.Cor, tipo model.TipoCarta, valor *string) model.Carta {
	pm.mu.Lock()
	pm.contadorCarta++
	id := fmt.Sprintf("carta-%03d", pm.contadorCarta)
	pm.mu.Unlock()

	return model.Carta{
		ID:    id,
		Cor:   cor,
		Tipo:  tipo,
		Valor: valor,
	}
}

// Embaralhar embaralha o slice de cartas usando Fisher-Yates.
func Embaralhar(cartas []model.Carta) {
	rand.Shuffle(len(cartas), func(i, j int) {
		cartas[i], cartas[j] = cartas[j], cartas[i]
	})
}

// ComprarCartaDoMonte compra a carta do topo do monteCompra.
// Se o monteCompra estiver vazio, recicla o monteDescarte:
// pega todas as cartas do monteDescarte exceto a cartaTopo,
// embaralha e as coloca como novo monteCompra.
func ComprarCartaDoMonte(jogo *model.Jogo) (model.Carta, error) {
	if len(jogo.MonteCompra) == 0 {
		if len(jogo.MonteDescarte) <= 1 {
			return model.Carta{}, fmt.Errorf("sem cartas disponíveis para compra")
		}

		recicladas := make([]model.Carta, len(jogo.MonteDescarte)-1)
		copy(recicladas, jogo.MonteDescarte[:len(jogo.MonteDescarte)-1])
		jogo.MonteDescarte = jogo.MonteDescarte[len(jogo.MonteDescarte)-1:]

		Embaralhar(recicladas)
		jogo.MonteCompra = recicladas
	}

	ultimo := len(jogo.MonteCompra) - 1
	carta := jogo.MonteCompra[ultimo]
	jogo.MonteCompra = jogo.MonteCompra[:ultimo]
	return carta, nil
}

// ComprarMultiplasCartas compra n cartas do monte e retorna o slice.
func ComprarMultiplasCartas(jogo *model.Jogo, n int) ([]model.Carta, error) {
	cartas := make([]model.Carta, 0, n)
	for range n {
		c, err := ComprarCartaDoMonte(jogo)
		if err != nil {
			return cartas, err
		}
		cartas = append(cartas, c)
	}
	return cartas, nil
}
