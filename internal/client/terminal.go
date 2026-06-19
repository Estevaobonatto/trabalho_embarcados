package client

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Terminal struct {
	API     *APIClient
	Session *Session
	scanner *bufio.Scanner
}

func NewTerminal(api *APIClient) *Terminal {
	return &Terminal{
		API:     api,
		Session: NewSession(api),
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (t *Terminal) Run() {
	t.exibirCabecalho()
	fmt.Println("Digite 'ajuda' para ver os comandos disponiveis.")
	fmt.Println()

	for {
		fmt.Print("uno> ")
		if !t.scanner.Scan() {
			break
		}
		linha := strings.TrimSpace(t.scanner.Text())
		if linha == "" {
			continue
		}
		t.executarComando(linha)
	}
}

func (t *Terminal) exibirCabecalho() {
	fmt.Println("========================================")
	fmt.Println("  UNO - Terminal (V2)")
	fmt.Printf("  Servidor ativo: %s\n", t.API.ActiveURL())
	if len(t.API.Servers) > 1 {
		fmt.Printf("  Servidores configurados: %v\n", t.API.Servers)
	}
	if t.Session.TemJogador() {
		fmt.Printf("  Jogador: %s (%s)\n", t.Session.Nome, t.Session.JogadorId)
	}
	if t.Session.EstaEmPartida() {
		fmt.Printf("  Partida: %s\n", t.Session.GameId)
	}
	fmt.Println("========================================")
}

func (t *Terminal) executarComando(linha string) {
	partes := strings.Fields(linha)
	if len(partes) == 0 {
		return
	}
	cmd := strings.ToLower(partes[0])
	args := partes[1:]

	switch cmd {
	case "criar":
		t.cmdCriarJogador(args)
	case "novo":
		t.cmdCriarJogo()
	case "listar":
		t.cmdListarJogos()
	case "entrar":
		t.cmdEntrar(args)
	case "jogar":
		t.cmdJogar(args)
	case "comprar":
		t.cmdComprar()
	case "uno":
		t.cmdUno()
	case "bater":
		t.cmdBater()
	case "estado":
		t.cmdEstado()
	case "eventos":
		t.cmdEventos(args)
	case "ranking":
		t.cmdRanking()
	case "status":
		t.cmdStatus()
	case "ajuda":
		t.cmdAjuda()
	case "sair", "exit", "quit":
		fmt.Println("Ate logo!")
		os.Exit(0)
	default:
		fmt.Printf("Comando desconhecido: %s\n", cmd)
		fmt.Println("Digite 'ajuda' para ver os comandos disponiveis.")
	}
}

func (t *Terminal) cmdCriarJogador(args []string) {
	if len(args) < 1 {
		fmt.Println("Uso: criar <nome>")
		return
	}
	nome := args[0]
	jogador, err := t.API.CriarJogador(nome)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	t.Session.JogadorId = jogador.JogadorId
	t.Session.Nome = jogador.Nome
	fmt.Printf("Jogador criado: %s (%s)\n", jogador.Nome, jogador.JogadorId)
}

func (t *Terminal) cmdCriarJogo() {
	if !t.Session.TemJogador() {
		fmt.Println("Primeiro crie um jogador: criar <nome>")
		return
	}
	jogo, err := t.API.CriarJogo(t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	t.Session.GameId = jogo.GameId
	fmt.Printf("Partida criada: %s (status: %s)\n", jogo.GameId, jogo.Status)
	fmt.Println("Compartilhe o gameId com outros jogadores para entrarem.")
}

func (t *Terminal) cmdListarJogos() {
	jogos, err := t.API.ListarJogos()
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	if len(jogos) == 0 {
		fmt.Println("Nenhuma partida disponivel.")
		return
	}
	fmt.Println("Partidas disponiveis:")
	for _, j := range jogos {
		fmt.Printf("  %s | status: %s | jogadores: %d/%d\n",
			j.GameId, j.Status, j.QuantidadeJogadores, j.MaxJogadores)
	}
}

func (t *Terminal) cmdEntrar(args []string) {
	if len(args) < 1 {
		fmt.Println("Uso: entrar <gameId>")
		return
	}
	if !t.Session.TemJogador() {
		fmt.Println("Primeiro crie um jogador: criar <nome>")
		return
	}
	gameId := args[0]
	info, err := t.API.EntrarNaPartida(gameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	t.Session.GameId = info.GameId
	fmt.Printf("Entrou na partida %s | status: %s | jogadores: %d\n",
		info.GameId, info.Status, info.QuantidadeJogadores)
	if info.Status == "EM_ANDAMENTO" {
		fmt.Println("A partida foi iniciada!")
		t.exibirEstado()
	}
}

func (t *Terminal) cmdJogar(args []string) {
	if len(args) < 1 {
		fmt.Println("Uso: jogar <codigo>  Ex: jogar R3, jogar GS, jogar GX")
		return
	}
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}

	codigo := args[0]
	estado, err := t.API.ObterEstado(t.Session.GameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro ao obter estado: %v\n", err)
		return
	}

	carta, corEsc, err := EncontrarCartaPorCodigo(estado.MinhaMao, codigo)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}

	res, err := t.API.JogarCarta(t.Session.GameId, t.Session.JogadorId, carta.ID, corEsc)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}

	fmt.Printf("Carta jogada! Proximo: %s | Cor atual: %s\n", res.ProximoJogador, res.CorAtual)
	t.exibirEstado()
}

func (t *Terminal) cmdComprar() {
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}
	res, err := t.API.ComprarCarta(t.Session.GameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	cartaStr := FormatarCartaColorida(res.CartaComprada)
	fmt.Printf("Comprou: %s | Proximo: %s\n", cartaStr, res.ProximoJogador)
	t.exibirEstado()
}

func (t *Terminal) cmdUno() {
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}
	res, err := t.API.ChamarUno(t.Session.GameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	fmt.Printf("UNO! Cartas: %d\n", res.QuantidadeCartas)
}

func (t *Terminal) cmdBater() {
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}
	res, err := t.API.Bater(t.Session.GameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	fmt.Printf("VENCEU! Vencedor: %s | Status: %s\n", res.Vencedor, res.Status)
	t.Session.GameId = ""
}

func (t *Terminal) cmdEstado() {
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}
	t.exibirEstado()
}

func (t *Terminal) exibirEstado() {
	estado, err := t.API.ObterEstado(t.Session.GameId, t.Session.JogadorId)
	if err != nil {
		fmt.Printf("Erro ao obter estado: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("--- MESA ---")
	if estado.CartaTopo != nil {
		cartaTopo := FormatarCartaColorida(*estado.CartaTopo)
		fmt.Printf("  Topo: %s\n", cartaTopo)
	}
	fmt.Printf("  Cor atual: %s | Sentido: %s\n", estado.CorAtual, estado.Sentido)
	fmt.Printf("  Vez de: %s\n", estado.JogadorDaVez)
	if estado.Vencedor != nil {
		fmt.Printf("  Vencedor: %s\n", *estado.Vencedor)
	}

	fmt.Println()
	fmt.Println("--- JOGADORES ---")
	for _, j := range estado.Jogadores {
		marcador := ""
		if j.JogadorId == t.Session.JogadorId {
			marcador = " (voce)"
		}
		unoStr := ""
		if j.ChamouUno {
			unoStr = " UNO!"
		}
		vezStr := ""
		if j.JogadorId == estado.JogadorDaVez {
			vezStr = " <<<"
		}
		fmt.Printf("  %s%s - %d cartas%s%s\n", j.Nome, marcador, j.QuantidadeCartas, unoStr, vezStr)
	}

	fmt.Println()
	fmt.Println("--- SUA MAO ---")
	if len(estado.MinhaMao) == 0 {
		fmt.Println("  (vazia)")
	} else {
		for i, carta := range estado.MinhaMao {
			cartaStr := FormatarCartaColorida(carta)
			fmt.Printf("  [%d] %s\n", i+1, cartaStr)
		}
	}
	fmt.Println()
}

func (t *Terminal) cmdEventos(args []string) {
	if !t.Session.EstaEmPartida() {
		fmt.Println("Voce nao esta em uma partida.")
		return
	}
	desde := 0
	if len(args) > 0 {
		d, err := strconv.Atoi(args[0])
		if err == nil {
			desde = d
		}
	}
	eventos, err := t.API.ObterEventos(t.Session.GameId, desde)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	if len(eventos) == 0 {
		fmt.Println("Nenhum evento.")
		return
	}
	for _, ev := range eventos {
		fmt.Printf("  #%d [%s] %s: %s\n", ev.Sequencia, ev.Tipo, ev.JogadorId, ev.Mensagem)
	}
}

func (t *Terminal) cmdRanking() {
	ranking, err := t.API.ObterLeaderboard()
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	if len(ranking) == 0 {
		fmt.Println("Leaderboard vazio.")
		return
	}
	fmt.Println("--- LEADERBOARD ---")
	for i, r := range ranking {
		fmt.Printf("  %d. %s - %d vitorias\n", i+1, r.Nome, r.Vitorias)
	}
}

func (t *Terminal) cmdStatus() {
	srv, err := t.API.GetServidor()
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	fmt.Printf("Servidor: %s (%s)\n", srv.ServidorId, srv.Nome)
	fmt.Printf("Versao contrato: %s\n", srv.VersaoContrato)
	fmt.Printf("Status: %s | Lider: %v\n", srv.Status, srv.Lider)
	fmt.Printf("Endereco lider: %s\n", srv.EnderecoLider)
	fmt.Printf("Versao estado: %d\n", srv.VersaoEstadoAtual)
	fmt.Printf("Este cliente esta conectado em: %s\n", t.API.ActiveURL())
}

func (t *Terminal) cmdAjuda() {
	fmt.Println("Comandos disponiveis (V2):")
	fmt.Println("  criar <nome>         Criar jogador")
	fmt.Println("  novo                 Criar nova partida")
	fmt.Println("  listar               Listar partidas disponiveis")
	fmt.Println("  entrar <gameId>      Entrar em uma partida")
	fmt.Println("  jogar <codigo>       Jogar carta (ex: R3, B5, GS, GX)")
	fmt.Println("  comprar              Comprar carta do monte")
	fmt.Println("  uno                  Chamar UNO (opcional, sem penalidade na V2)")
	fmt.Println("  bater                Bater (vencer)")
	fmt.Println("  estado               Ver estado da partida")
	fmt.Println("  eventos [desde]      Ver eventos da partida")
	fmt.Println("  ranking              Ver leaderboard")
	fmt.Println("  status               Ver informacoes do servidor")
	fmt.Println("  ajuda                Mostrar esta ajuda")
	fmt.Println("  sair                 Sair do cliente")
	fmt.Println()
	fmt.Println("Codigos de carta (V2: sem +2 e +4):")
	fmt.Println("  R=VERMELHO B=AZUL G=VERDE Y=AMARELO")
	fmt.Println("  Numeros: R0-R9, B0-B9, G0-G9, Y0-Y9")
	fmt.Println("  Especiais: RS/BS/GS/YS=Pular  RR/BR/GR/YR=Inverter")
	fmt.Println("             GX/BX/RX/YX=Coringa (cor no codigo = cor escolhida)")
	fmt.Println()
	fmt.Println("Failover automatico:")
	fmt.Println("  O cliente tenta o primeiro servidor (lider). Se nao responder,")
	fmt.Println("  tenta o proximo da lista. Em caso de SERVIDOR_NAO_E_LIDER,")
	fmt.Println("  redireciona automaticamente para o endereco indicado na resposta.")
	fmt.Println("  Para multiplos servidores: --server http://host1,http://host2")
}
