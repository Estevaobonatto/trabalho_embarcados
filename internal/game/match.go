package game

import (
	"fmt"
	"sync"

	"uno-api/internal/model"
)

// ErroJogo representa um erro de regra de negócio do jogo com código padronizado.
type ErroJogo struct {
	Codigo model.CodigoErro
	Msg    string
}

func (e *ErroJogo) Error() string {
	return string(e.Codigo) + ": " + e.Msg
}

// NovoErro cria um ErroJogo com o código e mensagem padrão do contrato.
func NovoErro(codigo model.CodigoErro) *ErroJogo {
	msg, ok := model.MensagensErro[codigo]
	if !ok {
		msg = "Erro desconhecido."
	}
	return &ErroJogo{Codigo: codigo, Msg: msg}
}

// PartidaManager gerencia todas as partidas e jogadores do servidor.
// Toda operação que modifica estado é thread-safe via mutexes.
type PartidaManager struct {
	mu              sync.RWMutex
	jogos           map[string]*model.Jogo
	jogadores       map[string]*model.Jogador
	contadorJogador int
	contadorJogo    int
	contadorCarta   int
}

// NewPartidaManager cria um novo gerenciador de partidas.
func NewPartidaManager() *PartidaManager {
	return &PartidaManager{
		jogos:     make(map[string]*model.Jogo),
		jogadores: make(map[string]*model.Jogador),
	}
}

// --- OPERAÇÕES DE JOGADOR ---

// CriarJogador cria um novo jogador com o nome informado.
// Retorna erro NOME_INVALIDO se o nome estiver vazio.
func (pm *PartidaManager) CriarJogador(nome string) (*model.Jogador, error) {
	if nome == "" {
		return nil, NovoErro(model.NOME_INVALIDO)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.contadorJogador++
	id := fmt.Sprintf("jogador-%03d", pm.contadorJogador)

	jogador := &model.Jogador{
		JogadorId: id,
		Nome:      nome,
		Vitorias:  0,
		Mao:       make([]model.Carta, 0),
		ChamouUno: false,
	}
	pm.jogadores[id] = jogador
	return jogador, nil
}

// ObterJogador retorna um jogador pelo ID ou erro se não encontrado.
func (pm *PartidaManager) ObterJogador(jogadorId string) (*model.Jogador, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	jog, ok := pm.jogadores[jogadorId]
	if !ok {
		return nil, NovoErro(model.JOGADOR_NAO_ENCONTRADO)
	}
	return jog, nil
}

// --- OPERAÇÕES DE PARTIDA ---

// CriarPartida cria uma nova partida com o jogador como criador (entra automaticamente).
func (pm *PartidaManager) CriarPartida(jogadorId string) (*model.Jogo, error) {
	jogador, err := pm.ObterJogador(jogadorId)
	if err != nil {
		return nil, err
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.contadorJogo++
	gameId := fmt.Sprintf("jogo-%03d", pm.contadorJogo)

	jogo := model.NewJogo(gameId)
	jogo.Jogadores = append(jogo.Jogadores, jogador)

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.JOGADOR_ENTROU, jogadorId,
		fmt.Sprintf("%s entrou na partida", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	pm.jogos[gameId] = jogo
	return jogo, nil
}

// EntrarNaPartida adiciona um jogador a uma partida existente.
func (pm *PartidaManager) EntrarNaPartida(gameId, jogadorId string) (*model.Jogo, error) {
	jogador, err := pm.ObterJogador(jogadorId)
	if err != nil {
		return nil, err
	}

	pm.mu.Lock()
	jogo, ok := pm.jogos[gameId]
	if !ok {
		pm.mu.Unlock()
		return nil, NovoErro(model.JOGO_NAO_ENCONTRADO)
	}

	jogo.Lock()

	if jogo.EstaIniciado() {
		jogo.Unlock()
		pm.mu.Unlock()
		return nil, NovoErro(model.JOGO_JA_INICIADO)
	}
	if jogo.EstaCheio() {
		jogo.Unlock()
		pm.mu.Unlock()
		return nil, NovoErro(model.JOGO_CHEIO)
	}
	if jogo.BuscarJogador(jogadorId) != nil {
		jogo.Unlock()
		pm.mu.Unlock()
		return jogo, nil
	}

	jogo.Jogadores = append(jogo.Jogadores, jogador)

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.JOGADOR_ENTROU, jogadorId,
		fmt.Sprintf("%s entrou na partida", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	if jogo.EstaCheio() {
		pm.mu.Unlock()
		pm.IniciarPartidaSeCheia(jogo)
	} else {
		pm.mu.Unlock()
	}

	jogo.Unlock()
	return jogo, nil
}

// IniciarPartidaSeCheia inicia a partida automaticamente se atingiu 4 jogadores.
// Deve ser chamada com o lock do jogo já adquirido.
func (pm *PartidaManager) IniciarPartidaSeCheia(jogo *model.Jogo) error {
	if jogo.EstaCheio() && !jogo.EstaIniciado() {
		return pm.iniciarPartidaInterno(jogo)
	}
	return nil
}

// iniciarPartidaInterno executa a inicialização da partida (distribuir cartas, etc.).
// O lock do jogo deve estar adquirido antes de chamar.
func (pm *PartidaManager) iniciarPartidaInterno(jogo *model.Jogo) error {
	baralho := pm.CriarBaralho()
	Embaralhar(baralho)
	jogo.MonteCompra = baralho
	jogo.Status = model.EM_ANDAMENTO

	for _, jog := range jogo.Jogadores {
		for range model.CartasIniciaisPorJogador {
			carta, err := ComprarCartaDoMonte(jogo)
			if err != nil {
				return err
			}
			jog.Mao = append(jog.Mao, carta)
		}
	}

	for {
		carta, err := ComprarCartaDoMonte(jogo)
		if err != nil {
			return err
		}
		jogo.MonteDescarte = append(jogo.MonteDescarte, carta)
		jogo.CartaTopo = &jogo.MonteDescarte[len(jogo.MonteDescarte)-1]
		jogo.CorAtual = carta.Cor

		if carta.EhNumerica() {
			break
		}
	}

	jogo.JogadorDaVez = jogo.Jogadores[0].JogadorId
	jogo.Sentido = model.HORARIO

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.JOGO_INICIADO, "",
		"O jogo foi iniciado", versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	return nil
}

// IniciarPartida inicia uma partida que está em AGUARDANDO_JOGADORES.
// Distribui 7 cartas por jogador, vira a primeira carta do descarte (deve ser numérica).
// Requer pelo menos 2 jogadores.
func (pm *PartidaManager) IniciarPartida(gameId string) (*model.Jogo, error) {
	pm.mu.RLock()
	jogo, ok := pm.jogos[gameId]
	pm.mu.RUnlock()
	if !ok {
		return nil, NovoErro(model.JOGO_NAO_ENCONTRADO)
	}

	jogo.Lock()
	defer jogo.Unlock()

	if jogo.EstaIniciado() {
		return nil, NovoErro(model.JOGO_JA_INICIADO)
	}
	if len(jogo.Jogadores) < 2 {
		return nil, NovoErro(model.ERRO_INTERNO)
	}

	if err := pm.iniciarPartidaInterno(jogo); err != nil {
		return nil, NovoErro(model.ERRO_INTERNO)
	}

	return jogo, nil
}

// --- OPERAÇÕES DE JOGO ---

// ResultadoJogada contém os dados retornados após uma jogada bem-sucedida.
type ResultadoJogada struct {
	GameId         string `json:"gameId"`
	VersaoEstado   int    `json:"versaoEstado"`
	ProximoJogador string `json:"proximoJogador"`
	CorAtual       string `json:"corAtual"`
}

// JogarCarta realiza uma jogada completa com validação e aplicação de efeitos.
// Fluxo conforme Seções 7.7 e 8 do contrato.
func (pm *PartidaManager) JogarCarta(gameId, jogadorId, cartaId string, corEscolhida *model.Cor) (*ResultadoJogada, error) {
	jogo, jogador, err := pm.obterJogoEJogador(gameId, jogadorId)
	if err != nil {
		return nil, err
	}

	jogo.Lock()
	defer jogo.Unlock()

	if jogo.EstaFinalizado() {
		return nil, NovoErro(model.JOGO_FINALIZADO)
	}
	if jogadorId != jogo.JogadorDaVez {
		return nil, NovoErro(model.NAO_E_SUA_VEZ)
	}

	carta, ok := jogador.ObterCarta(cartaId)
	if !ok {
		return nil, NovoErro(model.CARTA_NAO_ENCONTRADA)
	}

	if err := ValidarCorEscolhida(&carta, corEscolhida); err != nil {
		return nil, err
	}

	if err := ValidarJogada(&carta, jogo.CartaTopo, jogo.CorAtual); err != nil {
		return nil, err
	}

	// V2: penalidade UNO removida (Estrutura V2 — "Chamar UNO e levar penalidade" é opcional).
	// Mantemos o estado `chamouUno` apenas para fins de exibição no estado público.

	jogador.RemoverCarta(cartaId)
	jogador.ChamouUno = false

	jogo.MonteDescarte = append(jogo.MonteDescarte, carta)
	jogo.CartaTopo = &jogo.MonteDescarte[len(jogo.MonteDescarte)-1]

	if carta.PrecisaCorEscolhida() {
		jogo.CorAtual = *corEscolhida
	} else {
		jogo.CorAtual = carta.Cor
	}

	pulos := pm.aplicarEfeitoCarta(jogo, &carta)
	proximo := pm.calcularProximoJogador(jogo, pulos)
	jogo.JogadorDaVez = proximo

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.CARTA_JOGADA, jogadorId,
		fmt.Sprintf("%s jogou uma carta", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	return &ResultadoJogada{
		GameId:         gameId,
		VersaoEstado:   versao,
		ProximoJogador: proximo,
		CorAtual:       string(jogo.CorAtual),
	}, nil
}

// ResultadoCompra contém os dados retornados após uma compra bem-sucedida.
type ResultadoCompra struct {
	CartaComprada  model.Carta `json:"cartaComprada"`
	PassouAVez     bool        `json:"passouAVez"`
	ProximoJogador string      `json:"proximoJogador"`
}

// ComprarCarta permite ao jogador comprar uma carta do monte.
// A carta comprada NÃO pode ser jogada imediatamente (passouAVez = true).
func (pm *PartidaManager) ComprarCarta(gameId, jogadorId string) (*ResultadoCompra, error) {
	jogo, jogador, err := pm.obterJogoEJogador(gameId, jogadorId)
	if err != nil {
		return nil, err
	}

	jogo.Lock()
	defer jogo.Unlock()

	if jogo.EstaFinalizado() {
		return nil, NovoErro(model.JOGO_FINALIZADO)
	}
	if jogadorId != jogo.JogadorDaVez {
		return nil, NovoErro(model.NAO_E_SUA_VEZ)
	}

	// V2: penalidade UNO removida.

	carta, err := ComprarCartaDoMonte(jogo)
	if err != nil {
		return nil, NovoErro(model.ERRO_INTERNO)
	}

	jogador.Mao = append(jogador.Mao, carta)
	jogador.ChamouUno = false

	proximo := pm.calcularProximoJogador(jogo, 0)
	jogo.JogadorDaVez = proximo

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.CARTA_COMPRADA, jogadorId,
		fmt.Sprintf("%s comprou uma carta", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	return &ResultadoCompra{
		CartaComprada:  carta,
		PassouAVez:     true,
		ProximoJogador: proximo,
	}, nil
}

// ChamarUno registra que o jogador chamou UNO.
// Conforme Estrutura V2: chamar UNO é opcional e não há penalidade. O endpoint
// permanece para compatibilidade com o contrato, mas a chamada é sempre aceita
// (exceto se o jogo já estiver finalizado).
func (pm *PartidaManager) ChamarUno(gameId, jogadorId string) error {
	jogo, jogador, err := pm.obterJogoEJogador(gameId, jogadorId)
	if err != nil {
		return err
	}

	jogo.Lock()
	defer jogo.Unlock()

	if jogo.EstaFinalizado() {
		return NovoErro(model.JOGO_FINALIZADO)
	}

	jogador.ChamouUno = true

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.UNO_CHAMADO, jogadorId,
		fmt.Sprintf("%s chamou UNO", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	return nil
}

// ResultadoBater contém os dados retornados após bater.
type ResultadoBater struct {
	Vencedor string `json:"vencedor"`
	Status   string `json:"status"`
}

// Bater confirma a vitória do jogador que ficou com 0 cartas.
func (pm *PartidaManager) Bater(gameId, jogadorId string) (*ResultadoBater, error) {
	jogo, jogador, err := pm.obterJogoEJogador(gameId, jogadorId)
	if err != nil {
		return nil, err
	}

	jogo.Lock()
	defer jogo.Unlock()

	if jogo.EstaFinalizado() {
		return nil, NovoErro(model.JOGO_FINALIZADO)
	}
	if len(jogador.Mao) > 0 {
		return nil, NovoErro(model.JOGADOR_AINDA_TEM_CARTAS)
	}

	jogo.Status = model.FINALIZADO
	jogo.Vencedor = &jogadorId

	jogador.Vitorias++

	versao := jogo.IncrementarVersao()
	evento := model.NovoEvento(len(jogo.Eventos)+1, model.JOGADOR_BATEU, jogadorId,
		fmt.Sprintf("%s bateu", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, evento)

	eventoFim := model.NovoEvento(len(jogo.Eventos)+1, model.PARTIDA_FINALIZADA, jogadorId,
		fmt.Sprintf("Jogo finalizado. Vencedor: %s", jogador.Nome), versao)
	jogo.Eventos = append(jogo.Eventos, eventoFim)

	return &ResultadoBater{
		Vencedor: jogadorId,
		Status:   string(model.FINALIZADO),
	}, nil
}

// --- CONSULTAS ---

// EstadoPublico contém a visão pública do estado de uma partida para um jogador específico.
type EstadoPublico struct {
	GameId       string                 `json:"gameId"`
	Status       string                 `json:"status"`
	VersaoEstado int                    `json:"versaoEstado"`
	JogadorDaVez string                 `json:"jogadorDaVez"`
	Sentido      string                 `json:"sentido"`
	CorAtual     string                 `json:"corAtual"`
	CartaTopo    *model.Carta           `json:"cartaTopo"`
	MinhaMao     []model.Carta          `json:"minhaMao"`
	Jogadores    []model.JogadorPublico `json:"jogadores"`
	Vencedor     *string                `json:"vencedor"`
}

// ObterEstadoPublico retorna o estado visível da partida para um jogador.
func (pm *PartidaManager) ObterEstadoPublico(gameId, jogadorId string) (*EstadoPublico, error) {
	jogo, err := pm.ObterJogo(gameId)
	if err != nil {
		return nil, err
	}

	jogo.RLock()
	defer jogo.RUnlock()

	jogador := jogo.BuscarJogador(jogadorId)
	if jogador == nil {
		return nil, NovoErro(model.JOGADOR_NAO_ENCONTRADO)
	}

	jogadoresPublicos := make([]model.JogadorPublico, len(jogo.Jogadores))
	for i, j := range jogo.Jogadores {
		jogadoresPublicos[i] = j.ParaJogadorPublico()
	}

	estado := &EstadoPublico{
		GameId:       jogo.GameId,
		Status:       string(jogo.Status),
		VersaoEstado: jogo.VersaoEstado,
		JogadorDaVez: jogo.JogadorDaVez,
		Sentido:      string(jogo.Sentido),
		CorAtual:     string(jogo.CorAtual),
		CartaTopo:    jogo.CartaTopo,
		MinhaMao:     jogador.Mao,
		Jogadores:    jogadoresPublicos,
		Vencedor:     jogo.Vencedor,
	}

	return estado, nil
}

// ResumoJogo contém dados resumidos de uma partida para listagem.
type ResumoJogo struct {
	GameId              string `json:"gameId"`
	Status              string `json:"status"`
	QuantidadeJogadores int    `json:"quantidadeJogadores"`
	MaxJogadores        int    `json:"maxJogadores"`
}

// ListarJogos retorna a lista resumida de todas as partidas.
func (pm *PartidaManager) ListarJogos() []ResumoJogo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	resumo := make([]ResumoJogo, 0, len(pm.jogos))
	for _, jogo := range pm.jogos {
		jogo.RLock()
		resumo = append(resumo, ResumoJogo{
			GameId:              jogo.GameId,
			Status:              string(jogo.Status),
			QuantidadeJogadores: len(jogo.Jogadores),
			MaxJogadores:        model.MaxJogadoresPorPartida,
		})
		jogo.RUnlock()
	}
	return resumo
}

// ObterJogo retorna um jogo pelo ID (sem lock).
func (pm *PartidaManager) ObterJogo(gameId string) (*model.Jogo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	jogo, ok := pm.jogos[gameId]
	if !ok {
		return nil, NovoErro(model.JOGO_NAO_ENCONTRADO)
	}
	return jogo, nil
}

// ObterEventos retorna os eventos de uma partida a partir da sequência `desde`.
func (pm *PartidaManager) ObterEventos(gameId string, desde int) ([]model.Evento, error) {
	jogo, err := pm.ObterJogo(gameId)
	if err != nil {
		return nil, err
	}

	jogo.RLock()
	defer jogo.RUnlock()

	if desde <= 0 {
		return jogo.Eventos, nil
	}

	resultado := make([]model.Evento, 0)
	for _, evt := range jogo.Eventos {
		if evt.Sequencia > desde {
			resultado = append(resultado, evt)
		}
	}
	return resultado, nil
}

// JogadorRanking contém os dados de um jogador no leaderboard.
type JogadorRanking struct {
	JogadorId string `json:"jogadorId"`
	Nome      string `json:"nome"`
	Vitorias  int    `json:"vitorias"`
}

// ObterLeaderboard retorna o ranking de vitórias ordenado decrescente.
func (pm *PartidaManager) ObterLeaderboard() []JogadorRanking {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ranking := make([]JogadorRanking, 0, len(pm.jogadores))
	for _, jog := range pm.jogadores {
		ranking = append(ranking, JogadorRanking{
			JogadorId: jog.JogadorId,
			Nome:      jog.Nome,
			Vitorias:  jog.Vitorias,
		})
	}

	for i := 0; i < len(ranking); i++ {
		for j := i + 1; j < len(ranking); j++ {
			if ranking[j].Vitorias > ranking[i].Vitorias {
				ranking[i], ranking[j] = ranking[j], ranking[i]
			}
		}
	}

	return ranking
}

// MaxVersaoEstado retorna a maior versão de estado entre todas as partidas.
func (pm *PartidaManager) MaxVersaoEstado() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	max := 0
	for _, jogo := range pm.jogos {
		jogo.RLock()
		if jogo.VersaoEstado > max {
			max = jogo.VersaoEstado
		}
		jogo.RUnlock()
	}
	return max
}

// --- FUNÇÕES INTERNAS ---

// obterJogoEJogador obtém o jogo e o jogador com locks de leitura.
func (pm *PartidaManager) obterJogoEJogador(gameId, jogadorId string) (*model.Jogo, *model.Jogador, error) {
	jogo, err := pm.ObterJogo(gameId)
	if err != nil {
		return nil, nil, err
	}

	jogo.RLock()
	jogador := jogo.BuscarJogador(jogadorId)
	jogo.RUnlock()

	if jogador == nil {
		return nil, nil, NovoErro(model.JOGADOR_NAO_ENCONTRADO)
	}

	return jogo, jogador, nil
}

// aplicarEfeitoCarta aplica o efeito da carta jogada e retorna quantos jogadores pular.
// Conforme Estrutura V2: apenas NUMERICA, PULAR, INVERTER e CORINGA têm efeito.
// As cartas MAIS_DOIS e MAIS_QUATRO não fazem parte do baralho V2.
func (pm *PartidaManager) aplicarEfeitoCarta(jogo *model.Jogo, carta *model.Carta) int {
	switch carta.Tipo {
	case model.PULAR:
		return 1

	case model.INVERTER:
		if jogo.Sentido == model.HORARIO {
			jogo.Sentido = model.ANTI_HORARIO
		} else {
			jogo.Sentido = model.HORARIO
		}
		// Com 2 jogadores (V2), INVERTER funciona como PULAR: o oponente perde a vez.
		// A carta já é a do jogadorDaVez, e o sentido é trocado.
		// O próximo jogador no novo sentido é o próprio jogador atual.
		if len(jogo.Jogadores) == 2 {
			return 1
		}
		return 0

	case model.CORINGA, model.MAIS_DOIS, model.MAIS_QUATRO:
		// CORINGA (V2): só muda a cor, sem efeito de pulo.
		// MAIS_DOIS/MAIS_QUATRO: não fazem parte do baralho V2 (ignorados).
		return 0

	default:
		return 0
	}
}

// calcularProximoJogador calcula o próximo jogador considerando sentido e pulos.
func (pm *PartidaManager) calcularProximoJogador(jogo *model.Jogo, pulos int) string {
	n := len(jogo.Jogadores)
	if n == 0 {
		return ""
	}

	idx := jogo.IndiceJogador(jogo.JogadorDaVez)
	if idx < 0 {
		return jogo.Jogadores[0].JogadorId
	}

	if jogo.Sentido == model.HORARIO {
		idx = (idx + 1 + pulos) % n
	} else {
		idx = (idx - 1 - pulos) % n
		if idx < 0 {
			idx += n
		}
	}

	return jogo.Jogadores[idx].JogadorId
}

// verificarPenalidadesUno é mantido como stub para compatibilidade.
// Conforme Estrutura V2, a penalidade de UNO foi removida (era opcional).
// O estado `chamouUno` ainda é exibido publicamente mas não há consequência.
func (pm *PartidaManager) verificarPenalidadesUno(jogo *model.Jogo) {
	_ = jogo
}

func (pm *PartidaManager) ExportarSnapshot(gameId string) (*model.JogoSnapshot, error) {
	jogo, err := pm.ObterJogo(gameId)
	if err != nil {
		return nil, err
	}
	jogo.RLock()
	defer jogo.RUnlock()
	return model.JogoParaSnapshot(jogo), nil
}

func (pm *PartidaManager) ImportarSnapshot(snap *model.JogoSnapshot) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	jogo, exists := pm.jogos[snap.GameId]
	if !exists {
		jogo = model.NewJogo(snap.GameId)
		pm.jogos[snap.GameId] = jogo
	}
	pm.atualizarContadorPorId(snap.GameId, &pm.contadorJogo)

	jogo.Lock()
	defer jogo.Unlock()

	jogo.Status = model.StatusPartida(snap.Status)
	jogo.VersaoEstado = snap.VersaoEstado
	jogo.JogadorDaVez = snap.JogadorDaVez
	jogo.Sentido = model.Sentido(snap.Sentido)
	jogo.CorAtual = model.Cor(snap.CorAtual)
	jogo.CartaTopo = snap.CartaTopo
	jogo.MonteCompra = snap.MonteCompra
	jogo.MonteDescarte = snap.MonteDescarte
	jogo.Eventos = snap.Eventos
	jogo.Vencedor = snap.Vencedor

	jogadores := make([]*model.Jogador, len(snap.Jogadores))
	for i, js := range snap.Jogadores {
		mao := make([]model.Carta, len(js.Mao))
		copy(mao, js.Mao)
		jogadores[i] = &model.Jogador{
			JogadorId: js.JogadorId,
			Nome:      js.Nome,
			Vitorias:  js.Vitorias,
			Mao:       mao,
			ChamouUno: js.ChamouUno,
		}
		pm.atualizarOuCriarJogador(js.JogadorId, js.Nome, js.Vitorias)
	}
	jogo.Jogadores = jogadores
}

func (pm *PartidaManager) atualizarOuCriarJogador(jogadorId, nome string, vitorias int) {
	pm.atualizarContadorPorId(jogadorId, &pm.contadorJogador)
	if existente, ok := pm.jogadores[jogadorId]; ok {
		if vitorias > existente.Vitorias {
			existente.Vitorias = vitorias
		}
		existente.Nome = nome
	} else {
		pm.jogadores[jogadorId] = &model.Jogador{
			JogadorId: jogadorId,
			Nome:      nome,
			Vitorias:  vitorias,
			Mao:       make([]model.Carta, 0),
		}
	}
}

func (pm *PartidaManager) atualizarContadorPorId(id string, contador *int) {
	var num int
	if _, err := fmt.Sscanf(id, "jogador-%d", &num); err != nil {
		if _, err := fmt.Sscanf(id, "jogo-%d", &num); err != nil {
			return
		}
	}
	if num > *contador {
		*contador = num
	}
}

func (pm *PartidaManager) ListarGameIds() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	ids := make([]string, 0, len(pm.jogos))
	for id := range pm.jogos {
		ids = append(ids, id)
	}
	return ids
}

func (pm *PartidaManager) SincronizarJogador(jogadorId, nome string, vitorias int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.atualizarOuCriarJogador(jogadorId, nome, vitorias)
}
