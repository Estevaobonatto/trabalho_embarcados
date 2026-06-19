package game

import (
	"testing"

	"uno-api/internal/model"
)

func TestCriarJogador(t *testing.T) {
	pm := setupManager()
	j, err := pm.CriarJogador("Ana")
	if err != nil {
		t.Fatal(err)
	}
	if j.JogadorId != "jogador-001" {
		t.Errorf("esperado jogador-001, veio %s", j.JogadorId)
	}
	if j.Nome != "Ana" {
		t.Errorf("esperado Ana, veio %s", j.Nome)
	}
	if j.Vitorias != 0 {
		t.Errorf("vitorias iniciais deviam ser 0")
	}
}

func TestCriarJogadorNomeVazio(t *testing.T) {
	pm := setupManager()
	_, err := pm.CriarJogador("")
	if err == nil {
		t.Error("nome vazio devia gerar erro")
	}
}

func TestCriarPartida(t *testing.T) {
	pm := setupManager()
	pm.CriarJogador("Ana")

	jogo, err := pm.CriarPartida("jogador-001")
	if err != nil {
		t.Fatal(err)
	}
	if jogo.GameId != "jogo-001" {
		t.Errorf("esperado jogo-001, veio %s", jogo.GameId)
	}
	if jogo.Status != model.AGUARDANDO_JOGADORES {
		t.Errorf("status inicial devia ser AGUARDANDO_JOGADORES")
	}
	if len(jogo.Jogadores) != 1 {
		t.Errorf("devia ter 1 jogador, tem %d", len(jogo.Jogadores))
	}
}

func TestCriarPartidaJogadorInexistente(t *testing.T) {
	pm := setupManager()
	_, err := pm.CriarPartida("jogador-999")
	if err == nil {
		t.Error("jogador inexistente devia gerar erro")
	}
}

func TestObterJogador(t *testing.T) {
	pm := setupManager()
	pm.CriarJogador("Ana")

	j, err := pm.ObterJogador("jogador-001")
	if err != nil {
		t.Fatal(err)
	}
	if j.Nome != "Ana" {
		t.Errorf("nome errado: %s", j.Nome)
	}
}

func TestObterJogadorInexistente(t *testing.T) {
	pm := setupManager()
	_, err := pm.ObterJogador("jogador-999")
	if err == nil {
		t.Error("devia retornar erro")
	}
}

func setupPartidaComJogadores(t *testing.T, pm *PartidaManager, n int) string {
	t.Helper()
	nomes := []string{"Ana", "Bruno", "Carla", "Daniel"}
	var gameId string
	for i := range n {
		j, _ := pm.CriarJogador(nomes[i])
		if i == 0 {
			jogo, _ := pm.CriarPartida(j.JogadorId)
			gameId = jogo.GameId
		}
	}
	for i := 1; i < n; i++ {
		pm.EntrarNaPartida(gameId, "jogador-"+formatSeq(i+1))
	}
	return gameId
}

func formatSeq(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	return "0" + string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestIniciarPartidaAutomatico2Jogadores(t *testing.T) {
	pm := setupManager()
	nomes := []string{"Ana", "Bruno"}
	var jogo *model.Jogo
	for i, nome := range nomes {
		j, _ := pm.CriarJogador(nome)
		if i == 0 {
			jogo, _ = pm.CriarPartida(j.JogadorId)
		} else {
			jogo, _ = pm.EntrarNaPartida(jogo.GameId, j.JogadorId)
		}
	}

	// V2: partida inicia automaticamente com 2 jogadores.
	if jogo.Status != model.EM_ANDAMENTO {
		t.Error("partida V2 devia ter iniciado automaticamente com 2 jogadores")
	}
	for _, jog := range jogo.Jogadores {
		if len(jog.Mao) != 7 {
			t.Errorf("jogador %s devia ter 7 cartas, tem %d", jog.Nome, len(jog.Mao))
		}
	}
	if jogo.JogadorDaVez == "" {
		t.Error("jogadorDaVez nao foi definido")
	}
	if jogo.CartaTopo == nil {
		t.Error("cartaTopo nao foi definida")
	}
}

func TestEntrarPartidaCheia(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)
	pm.CriarJogador("Extra")
	_, err := pm.EntrarNaPartida(gid, "jogador-003")
	if err == nil {
		t.Error("devia rejeitar entrada em partida cheia/iniciada (V2 = 2 jogadores)")
	}
}

func TestJogarCartaForaDaVez(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)

	jogo, _ := pm.ObterJogo(gid)
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogo.RUnlock()

	var outroJogador string
	for _, j := range jogo.Jogadores {
		if j.JogadorId != vez {
			outroJogador = j.JogadorId
			break
		}
	}

	_, err := pm.JogarCarta(gid, outroJogador, "carta-001", nil)
	if err == nil {
		t.Error("jogar fora da vez devia gerar erro")
	}
}

func TestComprarCarta(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)

	jogo, _ := pm.ObterJogo(gid)
	jogo.RLock()
	vez := jogo.JogadorDaVez
	cartasAntes := len(jogo.BuscarJogador(vez).Mao)
	jogo.RUnlock()

	res, err := pm.ComprarCarta(gid, vez)
	if err != nil {
		t.Fatal(err)
	}
	if !res.PassouAVez {
		t.Error("passouAVez devia ser true apos comprar")
	}
	if res.ProximoJogador == vez {
		t.Error("proximoJogador nao devia ser o mesmo que comprou")
	}

	jogo.RLock()
	cartasDepois := len(jogo.BuscarJogador(vez).Mao)
	jogo.RUnlock()
	if cartasDepois != cartasAntes+1 {
		t.Errorf("devia ter %d cartas, tem %d", cartasAntes+1, cartasDepois)
	}
}

func TestChamarUnoSempreAceitaV2(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)
	jogo, _ := pm.ObterJogo(gid)
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogador := jogo.BuscarJogador(vez)
	jogo.RUnlock()

	// V2: chamar UNO é opcional e sempre aceito, mesmo com várias cartas.
	jogo.Lock()
	for len(jogador.Mao) > 1 {
		jogador.Mao = jogador.Mao[:len(jogador.Mao)-1]
	}
	jogo.Unlock()

	if err := pm.ChamarUno(gid, vez); err != nil {
		t.Errorf("V2: devia poder chamar UNO com 1 carta: %v", err)
	}

	// V2: também aceita com várias cartas (sem penalidade).
	pm2 := setupManager()
	gid2 := setupPartidaComJogadores(t, pm2, 2)
	jogo2, _ := pm2.ObterJogo(gid2)
	jogo2.RLock()
	vez2 := jogo2.JogadorDaVez
	jogo2.RUnlock()
	if err := pm2.ChamarUno(gid2, vez2); err != nil {
		t.Errorf("V2: UNO deve ser aceito com várias cartas (sem penalidade): %v", err)
	}
}

func TestBaterComCartas(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)
	jogo, _ := pm.ObterJogo(gid)
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogo.RUnlock()

	_, err := pm.Bater(gid, vez)
	if err == nil {
		t.Error("nao devia bater com cartas na mao")
	}
}

func TestBaterSemCartas(t *testing.T) {
	pm := setupManager()
	gid := setupPartidaComJogadores(t, pm, 2)
	jogo, _ := pm.ObterJogo(gid)
	jogo.RLock()
	vez := jogo.JogadorDaVez
	jogador := jogo.BuscarJogador(vez)
	jogo.RUnlock()

	jogo.Lock()
	jogador.Mao = nil
	jogo.Unlock()

	res, err := pm.Bater(gid, vez)
	if err != nil {
		t.Fatalf("devia poder bater com 0 cartas: %v", err)
	}
	if res.Status != "FINALIZADO" {
		t.Error("status devia ser FINALIZADO apos bater")
	}
	if res.Vencedor != vez {
		t.Errorf("vencedor devia ser %s, veio %s", vez, res.Vencedor)
	}
}

func TestInverterCom2Jogadores(t *testing.T) {
	pm := setupManager()
	pm.CriarJogador("Ana")
	pm.CriarJogador("Bruno")
	jogo, _ := pm.CriarPartida("jogador-001")
	pm.EntrarNaPartida(jogo.GameId, "jogador-002")
	// V2: não precisa chamar IniciarPartida — auto-start com 2 jogadores.
	// (aqui já inicia automaticamente)

	jogo.RLock()
	if len(jogo.Jogadores) != 2 {
		jogo.RUnlock()
		t.Skip("precisa de 2 jogadores")
		return
	}
	vez := jogo.JogadorDaVez
	jogo.RUnlock()

	cartaInverter := model.Carta{ID: "inv-test", Cor: jogo.CorAtual, Tipo: model.INVERTER}
	jogador := jogo.BuscarJogador(vez)

	jogo.Lock()
	jogador.Mao = append(jogador.Mao, cartaInverter)
	jogo.Unlock()

	res, err := pm.JogarCarta(jogo.GameId, vez, cartaInverter.ID, nil)
	if err != nil {
		t.Fatalf("jogar inverter devia funcionar: %v", err)
	}
	// V2: com 2 jogadores, INVERTER funciona como PULAR (mesmo jogador joga de novo).
	if res.ProximoJogador != vez {
		t.Error("com 2 jogadores, inverter devia pular o oponente (proximo = mesmo jogador)")
	}
}

func TestLeaderboard(t *testing.T) {
	pm := setupManager()
	pm.CriarJogador("Ana")
	pm.CriarJogador("Bruno")

	lb := pm.ObterLeaderboard()
	if len(lb) != 2 {
		t.Errorf("leaderboard devia ter 2 jogadores, tem %d", len(lb))
	}
}

func TestListarJogos(t *testing.T) {
	pm := setupManager()
	pm.CriarJogador("Ana")
	pm.CriarPartida("jogador-001")

	jogos := pm.ListarJogos()
	if len(jogos) != 1 {
		t.Errorf("devia ter 1 jogo, tem %d", len(jogos))
	}
	// V2: maxJogadores = 2.
	if jogos[0].MaxJogadores != 2 {
		t.Errorf("maxJogadores V2 devia ser 2, veio %d", jogos[0].MaxJogadores)
	}
}
