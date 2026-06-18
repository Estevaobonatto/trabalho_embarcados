package client

import (
	"fmt"
	"strings"
)

var corParaCodigo = map[string]string{
	"AMARELO":  "Y",
	"AZUL":     "B",
	"VERDE":    "G",
	"VERMELHO": "R",
	"PRETO":    "",
}

var codigoParaCor = map[string]string{
	"Y": "AMARELO",
	"B": "AZUL",
	"G": "VERDE",
	"R": "VERMELHO",
}

var codigoAnsi = map[string]string{
	"AMARELO":  "\033[33m",
	"AZUL":     "\033[34m",
	"VERDE":    "\033[32m",
	"VERMELHO": "\033[31m",
	"PRETO":    "\033[37m",
}

var tipoParaCodigo = map[string]string{
	"NUMERICA":    "",
	"PULAR":       "S",
	"INVERTER":    "R",
	"MAIS_DOIS":   "Z",
	"CORINGA":     "X",
	"MAIS_QUATRO": "Y",
}

type CartaCurta struct {
	Cor   string
	Tipo  string
	Valor *string
}

func (cc *CartaCurta) String() string {
	if cc.Tipo == "CORINGA" || cc.Tipo == "MAIS_QUATRO" {
		return cc.Tipo
	}
	corCod := corParaCodigo[cc.Cor]
	tipoCod := tipoParaCodigo[cc.Tipo]
	if cc.Tipo == "NUMERICA" && cc.Valor != nil {
		return corCod + *cc.Valor
	}
	return corCod + tipoCod
}

func (cc *CartaCurta) StringColorido() string {
	ansi := codigoAnsi[cc.Cor]
	reset := "\033[0m"
	return ansi + cc.String() + reset
}

func ParsearCodigoCurto(codigo string) *CartaCurta {
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	if len(codigo) < 2 {
		return nil
	}

	corCod := string(codigo[0])
	tipoCod := string(codigo[1])

	cor, ok := codigoParaCor[corCod]
	if !ok {
		return nil
	}

	cc := &CartaCurta{Cor: cor}

	switch tipoCod {
	case "S":
		cc.Tipo = "PULAR"
	case "R":
		cc.Tipo = "INVERTER"
	case "Z":
		cc.Tipo = "MAIS_DOIS"
	case "X":
		cc.Tipo = "CORINGA"
	case "Y":
		cc.Tipo = "MAIS_QUATRO"
	default:
		if tipoCod >= "0" && tipoCod <= "9" {
			cc.Tipo = "NUMERICA"
			val := tipoCod
			cc.Valor = &val
		} else {
			return nil
		}
	}

	return cc
}

func FormatarCartaCurta(carta CartaInfo) string {
	cc := CartaCurta{
		Cor:   carta.Cor,
		Tipo:  carta.Tipo,
		Valor: carta.Valor,
	}
	return cc.String()
}

func FormatarCartaColorida(carta CartaInfo) string {
	cc := CartaCurta{
		Cor:   carta.Cor,
		Tipo:  carta.Tipo,
		Valor: carta.Valor,
	}
	return cc.StringColorido()
}

func EncontrarCartaPorCodigo(mao []CartaInfo, codigo string) (CartaInfo, *string, error) {
	cc := ParsearCodigoCurto(codigo)
	if cc == nil {
		return CartaInfo{}, nil, fmt.Errorf("codigo invalido: %s", codigo)
	}

	if cc.Tipo == "CORINGA" || cc.Tipo == "MAIS_QUATRO" {
		for _, carta := range mao {
			if carta.Tipo == cc.Tipo {
				corEsc := cc.Cor
				return carta, &corEsc, nil
			}
		}
		return CartaInfo{}, nil, fmt.Errorf("voce nao possui %s na mao", cc.Tipo)
	}

	for _, carta := range mao {
		if carta.Cor == cc.Cor && carta.Tipo == cc.Tipo {
			if cc.Tipo == "NUMERICA" {
				if carta.Valor != nil && cc.Valor != nil && *carta.Valor == *cc.Valor {
					return carta, nil, nil
				}
			} else {
				return carta, nil, nil
			}
		}
	}

	return CartaInfo{}, nil, fmt.Errorf("voce nao possui a carta %s na mao", codigo)
}
