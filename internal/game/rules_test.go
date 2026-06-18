package game

import (
	"testing"

	"uno-api/internal/model"
)

func cartaNum(t *testing.T, cor model.Cor, valor string) model.Carta {
	return model.Carta{ID: "t-" + string(cor) + "-" + valor, Cor: cor, Tipo: model.NUMERICA, Valor: &valor}
}

func cartaEsp(t *testing.T, cor model.Cor, tipo model.TipoCarta) model.Carta {
	return model.Carta{ID: "t-" + string(cor) + "-" + string(tipo), Cor: cor, Tipo: tipo, Valor: nil}
}

func TestValidarJogadaNumericaMesmaCor(t *testing.T) {
	topo := cartaNum(t, model.VERMELHO, "5")
	jogada := cartaNum(t, model.VERMELHO, "3")
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("numerica mesma cor devia ser valida: %v", err)
	}
}

func TestValidarJogadaNumericaMesmoValor(t *testing.T) {
	topo := cartaNum(t, model.VERMELHO, "5")
	jogada := cartaNum(t, model.AZUL, "5")
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("numerica mesmo valor devia ser valida: %v", err)
	}
}

func TestValidarJogadaNumericaInvalida(t *testing.T) {
	topo := cartaNum(t, model.VERMELHO, "5")
	jogada := cartaNum(t, model.AZUL, "3")
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err == nil {
		t.Error("numerica cor e valor diferentes devia ser invalida")
	}
}

func TestValidarJogadaNumericaAposCoringa(t *testing.T) {
	coringa := cartaEsp(t, model.PRETO, model.CORINGA)
	jogada := cartaNum(t, model.VERDE, "7")
	err := ValidarJogada(&jogada, &coringa, model.VERDE)
	if err != nil {
		t.Errorf("numerica verde apos coringa verde devia ser valida: %v", err)
	}

	jogadaVermelha := cartaNum(t, model.VERMELHO, "7")
	err = ValidarJogada(&jogadaVermelha, &coringa, model.VERDE)
	if err == nil {
		t.Error("numerica vermelha apos coringa verde devia ser invalida")
	}
}

func TestValidarJogadaPularMesmaCor(t *testing.T) {
	topo := cartaNum(t, model.AZUL, "2")
	jogada := cartaEsp(t, model.AZUL, model.PULAR)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("pular mesma cor devia ser valido: %v", err)
	}
}

func TestValidarJogadaPularMesmoTipo(t *testing.T) {
	topo := cartaEsp(t, model.VERMELHO, model.PULAR)
	jogada := cartaEsp(t, model.AZUL, model.PULAR)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("pular mesmo tipo devia ser valido: %v", err)
	}
}

func TestValidarJogadaPularInvalido(t *testing.T) {
	topo := cartaNum(t, model.VERMELHO, "5")
	jogada := cartaEsp(t, model.AZUL, model.PULAR)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err == nil {
		t.Error("pular cor e tipo diferentes devia ser invalido")
	}
}

func TestValidarJogadaMaisDoisMesmoTipo(t *testing.T) {
	topo := cartaEsp(t, model.AMARELO, model.MAIS_DOIS)
	jogada := cartaEsp(t, model.VERDE, model.MAIS_DOIS)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("+2 mesmo tipo devia ser valido: %v", err)
	}
}

func TestValidarJogadaInverterMesmaCor(t *testing.T) {
	topo := cartaNum(t, model.VERDE, "8")
	jogada := cartaEsp(t, model.VERDE, model.INVERTER)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("inverter mesma cor devia ser valido: %v", err)
	}
}

func TestValidarJogadaCoringaSempreValido(t *testing.T) {
	topo := cartaNum(t, model.AMARELO, "0")
	jogada := cartaEsp(t, model.PRETO, model.CORINGA)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("coringa devia ser sempre valido: %v", err)
	}
}

func TestValidarJogadaMaisQuatroSempreValido(t *testing.T) {
	topo := cartaEsp(t, model.VERMELHO, model.PULAR)
	jogada := cartaEsp(t, model.PRETO, model.MAIS_QUATRO)
	err := ValidarJogada(&jogada, &topo, topo.Cor)
	if err != nil {
		t.Errorf("+4 devia ser sempre valido: %v", err)
	}
}

func TestValidarCorEscolhidaObrigatoriaCoringa(t *testing.T) {
	carta := cartaEsp(t, model.PRETO, model.CORINGA)
	err := ValidarCorEscolhida(&carta, nil)
	if err == nil {
		t.Error("coringa sem corEscolhida devia ser invalido")
	}
}

func TestValidarCorEscolhidaObrigatoriaMaisQuatro(t *testing.T) {
	carta := cartaEsp(t, model.PRETO, model.MAIS_QUATRO)
	err := ValidarCorEscolhida(&carta, nil)
	if err == nil {
		t.Error("+4 sem corEscolhida devia ser invalido")
	}
}

func TestValidarCorEscolhidaNaoPodeSerPreto(t *testing.T) {
	carta := cartaEsp(t, model.PRETO, model.CORINGA)
	cor := model.PRETO
	err := ValidarCorEscolhida(&carta, &cor)
	if err == nil {
		t.Error("coringa com corEscolhida PRETO devia ser invalido")
	}
}

func TestValidarCorEscolhidaValida(t *testing.T) {
	carta := cartaEsp(t, model.PRETO, model.CORINGA)
	cor := model.VERDE
	err := ValidarCorEscolhida(&carta, &cor)
	if err != nil {
		t.Errorf("coringa com corEscolhida VERDE devia ser valido: %v", err)
	}
}
