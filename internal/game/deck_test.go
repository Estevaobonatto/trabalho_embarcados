package game

import (
	"testing"

	"uno-api/internal/model"
)

func setupManager() *PartidaManager {
	return NewPartidaManager()
}

func TestCriarBaralho(t *testing.T) {
	pm := setupManager()
	baralho := pm.CriarBaralho()

	// V2: baralho reduzido (sem MAIS_DOIS e MAIS_QUATRO): 96 cartas.
	if len(baralho) != 96 {
		t.Fatalf("baralho V2 deve ter 96 cartas, tem %d", len(baralho))
	}

	contagem := make(map[model.TipoCarta]int)
	for _, c := range baralho {
		contagem[c.Tipo]++
		if c.ID == "" {
			t.Error("carta sem ID")
		}
	}

	if contagem[model.NUMERICA] != 76 {
		t.Errorf("numericas: esperado 76, tem %d", contagem[model.NUMERICA])
	}
	if contagem[model.PULAR] != 8 {
		t.Errorf("pular: esperado 8, tem %d", contagem[model.PULAR])
	}
	if contagem[model.INVERTER] != 8 {
		t.Errorf("inverter: esperado 8, tem %d", contagem[model.INVERTER])
	}
	if contagem[model.MAIS_DOIS] != 0 {
		t.Errorf("+2: esperado 0 (removido na V2), tem %d", contagem[model.MAIS_DOIS])
	}
	if contagem[model.CORINGA] != 4 {
		t.Errorf("coringa: esperado 4, tem %d", contagem[model.CORINGA])
	}
	if contagem[model.MAIS_QUATRO] != 0 {
		t.Errorf("+4: esperado 0 (removido na V2), tem %d", contagem[model.MAIS_QUATRO])
	}

	coresPretas := 0
	for _, c := range baralho {
		if c.Cor == model.PRETO {
			coresPretas++
		}
	}
	if coresPretas != 4 {
		t.Errorf("cartas PRETO: esperado 4 (apenas coringa), tem %d", coresPretas)
	}
}

func TestEmbaralhar(t *testing.T) {
	pm := setupManager()
	b1 := pm.CriarBaralho()
	b2 := pm.CriarBaralho()

	iguais := true
	for i := range b1 {
		if b1[i].ID != b2[i].ID {
			iguais = false
			break
		}
	}
	if !iguais {
		t.Skip("baralhos já vieram diferentes sem embaralhar")
	}

	Embaralhar(b1)
	diferentes := false
	for i := range b1 {
		if b1[i].ID != b2[i].ID {
			diferentes = true
			break
		}
	}
	if !diferentes {
		t.Error("baralho não foi embaralhado")
	}
}

func TestComprarCartaDoMonte(t *testing.T) {
	jogo := model.NewJogo("jogo-teste")
	pm := setupManager()
	baralho := pm.CriarBaralho()
	Embaralhar(baralho)
	jogo.MonteCompra = baralho

	totalInicial := len(jogo.MonteCompra)
	carta, err := ComprarCartaDoMonte(jogo)
	if err != nil {
		t.Fatal(err)
	}
	if carta.ID == "" {
		t.Error("carta comprada sem ID")
	}
	if len(jogo.MonteCompra) != totalInicial-1 {
		t.Errorf("monte devia ter %d cartas, tem %d", totalInicial-1, len(jogo.MonteCompra))
	}
}

func TestReciclarDescarte(t *testing.T) {
	jogo := model.NewJogo("jogo-teste")
	pm := setupManager()
	baralho := pm.CriarBaralho()
	Embaralhar(baralho)

	jogo.MonteDescarte = baralho
	jogo.CartaTopo = &jogo.MonteDescarte[len(jogo.MonteDescarte)-1]
	jogo.MonteCompra = nil

	carta, err := ComprarCartaDoMonte(jogo)
	if err != nil {
		t.Fatal(err)
	}
	if carta.ID == "" {
		t.Error("carta comprada sem ID apos reciclagem")
	}
	if len(jogo.MonteDescarte) != 1 {
		t.Errorf("descarte devia ter 1 carta (topo), tem %d", len(jogo.MonteDescarte))
	}
	// V2: baralho tem 96 cartas, após mover 1 para descarte restam 95.
	if len(jogo.MonteCompra)+len(jogo.MonteDescarte) != 95 {
		t.Errorf("total de cartas devia ser 95, tem %d", len(jogo.MonteCompra)+len(jogo.MonteDescarte))
	}
}

func TestComprarMultiplasCartas(t *testing.T) {
	jogo := model.NewJogo("jogo-teste")
	pm := setupManager()
	jogo.MonteCompra = pm.CriarBaralho()

	cartas, err := ComprarMultiplasCartas(jogo, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(cartas) != 5 {
		t.Errorf("esperado 5 cartas, tem %d", len(cartas))
	}
}
