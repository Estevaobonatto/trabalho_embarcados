package game

import (
	"uno-api/internal/model"
)

// ValidarJogada verifica se uma carta pode ser jogada sobre a carta do topo,
// conforme as regras da Seção 8 do contrato:
//
//   - NUMERICA: mesma cor OU mesmo valor
//   - PULAR, INVERTER, MAIS_DOIS: mesma cor OU mesmo tipo (símbolo)
//   - CORINGA, MAIS_QUATRO: SEMPRE válido
//
// Retorna nil se a jogada é válida, ou um erro com o código apropriado.
func ValidarJogada(carta, cartaTopo *model.Carta, corAtual model.Cor) error {
	switch carta.Tipo {
	case model.NUMERICA:
		if carta.Cor != corAtual && (cartaTopo == nil || carta.Cor != cartaTopo.Cor) {
			if carta.Valor == nil || cartaTopo == nil || cartaTopo.Valor == nil || *carta.Valor != *cartaTopo.Valor {
				return NovoErro(model.JOGADA_INVALIDA)
			}
		}
		return nil

	case model.PULAR, model.INVERTER, model.MAIS_DOIS:
		if carta.Cor != corAtual && (cartaTopo == nil || (carta.Cor != cartaTopo.Cor && carta.Tipo != cartaTopo.Tipo)) {
			return NovoErro(model.JOGADA_INVALIDA)
		}
		return nil

	case model.CORINGA, model.MAIS_QUATRO:
		return nil

	default:
		return NovoErro(model.JOGADA_INVALIDA)
	}
}

// ValidarCorEscolhida verifica se corEscolhida foi fornecida quando obrigatória
// (CORINGA ou MAIS_QUATRO), e se o valor é uma cor jogável (não PRETO).
func ValidarCorEscolhida(carta *model.Carta, corEscolhida *model.Cor) error {
	if carta.PrecisaCorEscolhida() {
		if corEscolhida == nil {
			return NovoErro(model.COR_OBRIGATORIA)
		}
		if *corEscolhida == model.PRETO {
			return NovoErro(model.COR_OBRIGATORIA)
		}
		if !model.CorValida(string(*corEscolhida)) {
			return NovoErro(model.COR_OBRIGATORIA)
		}
	}
	return nil
}
