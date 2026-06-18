# ROADMAP — UNO API Distribuído

> **Fontes da verdade:**
> - `Contrato Mínimo API - Uno.pdf` (V1.1, 13-06-2026) — endpoints, modelos, regras
> - `Estrutura do trabalho final.pdf` — requisitos distribuídos, documentação, integração entre grupos
>
> Projeto em duplas/trios. Cliente deve conectar a qualquer servidor que implemente o contrato mínimo.
> Partidas de 2 a 4 jogadores. Leaderboard compartilhado entre servidores do mesmo grupo.

---

## Visão Geral das Fases

| Fase | Nome | Tarefas | Status | Dependência |
|------|------|---------|--------|-------------|
| F1 | Fundação e Modelos | 11 | Concluída | Nenhuma |
| F2 | Lógica do Jogo UNO | 13 | Concluída | F1 |
| F3 | API REST (12 Endpoints) | 15 | Concluída | F2 |
| F4 | Cliente Terminal | 14 | Concluída | F3 |
| F5 | Replicação e Failover | 12 | Concluída | F3 |
| F6 | Documentação e Testes | 12 | Pendente | F4, F5 |

---

## FASE 1 — Fundação e Modelos

**Objetivo:** Estruturar o módulo Go e definir todos os modelos de dados conforme o contrato (Seções 2-9).

### F1.1 — Inicialização do Projeto
- [x] `F1.1.1` Inicializar módulo Go: `go mod init uno-api`
- [x] `F1.1.2` Instalar framework Gin: `go get github.com/gin-gonic/gin`
- [x] `F1.1.3` Criar estrutura de diretórios: `cmd/server/`, `cmd/client/`, `internal/model/`, `internal/game/`, `internal/api/`, `internal/replication/`, `internal/client/`

### F1.2 — Constantes e Enums (Seção 2 do contrato)
- [x] `F1.2.1` Definir `Cor` enum: `AMARELO`, `AZUL`, `VERMELHO`, `VERDE`, `PRETO` (PRETO exclusivo para CORINGA/MAIS_QUATRO)
- [x] `F1.2.2` Definir `TipoCarta` enum: `NUMERICA`, `PULAR`, `INVERTER`, `MAIS_DOIS`, `CORINGA`, `MAIS_QUATRO`
- [x] `F1.2.3` Definir `StatusPartida` enum: `AGUARDANDO_JOGADORES`, `EM_ANDAMENTO`, `FINALIZADO`, `CANCELADO`
- [x] `F1.2.4` Definir `Sentido` enum: `HORARIO`, `ANTI_HORARIO`
- [x] `F1.2.5` Definir `TipoEvento` enum com todos os 10 tipos: `JOGADOR_ENTROU`, `JOGO_INICIADO`, `CARTA_JOGADA`, `CARTA_COMPRADA`, `UNO_CHAMADO`, `PENALIDADE_UNO`, `JOGADOR_BATEU`, `JOGO_FINALIZADO`, `FAILOVER`, `LIDER_ALTERADO`
- [x] `F1.2.6` Definir `CodigoErro` enum com todos os 14 códigos da Seção 6 do contrato

### F1.3 — Modelos de Dados
- [x] `F1.3.1` `model/card.go` — struct `Carta`: `id`, `cor` (Cor), `tipo` (TipoCarta), `valor` (*string, null p/ especiais)
- [x] `F1.3.2` `model/player.go` — struct `Jogador`: `jogadorId`, `nome`, `vitorias` (int), `mao` ([]Carta, privada), `chamouUno` (bool)
- [x] `F1.3.3` `model/game.go` — struct `Jogo` (estado interno completo, Seção 9): `gameId`, `status`, `versaoEstado`, `jogadores`, `jogadorDaVez`, `sentido`, `corAtual`, `cartaTopo`, `monteCompra`, `monteDescarte`, `eventos`, `vencedor`
- [x] `F1.3.4` `model/event.go` — struct `Evento`: `sequencia` (int), `tipo`, `jogadorId`, `mensagem`, `versaoEstado`
- [x] `F1.3.5` `model/response.go` — structs `RespostaSucesso`/`RespostaErro` (Seções 4-5) + `StatusHTTP` mapeando código → HTTP status

---

## FASE 2 — Lógica do Jogo UNO

**Objetivo:** Mecânica completa do UNO: baralho, validação, efeitos especiais, UNO e bater.
Toda validação server-side (Seção 8). Suporte a 2-4 jogadores (Estrutura PDF).

### F2.1 — Baralho (Deck)
- [x] `F2.1.1` `CriarBaralho()` — 108 cartas padrão UNO:
  - 19 por cor (1× "0" + 2× "1" a "9") = 76 numéricas
  - 2× PULAR, 2× INVERTER, 2× MAIS_DOIS por cor = 24
  - 4× CORINGA + 4× MAIS_QUATRO (PRETO) = 8
- [x] `F2.1.2` `Embaralhar()` — Fisher-Yates via `math/rand`
- [x] `F2.1.3` `ComprarCartaDoMonte()` — compra do topo; se vazio, recicla monteDescarte (exceto cartaTopo), embaralha

### F2.2 — Validação de Jogadas (Seção 8)
- [x] `F2.2.1` `ValidarJogada(carta, cartaTopo, corAtual)`:
  - NUMERICA: mesma cor (corAtual) OU mesmo valor (vs cartaTopo)
  - PULAR/INVERTER/MAIS_DOIS: mesma cor (corAtual) OU mesmo tipo
  - CORINGA/MAIS_QUATRO: SEMPRE válido
- [x] `F2.2.2` `ValidarCorEscolhida()` — corEscolhida obrigatória p/ CORINGA/MAIS_QUATRO (erro `COR_OBRIGATORIA`)
- [x] `F2.2.3` Validar que a carta pertence à mão do jogador que a enviou

### F2.3 — Gerenciamento de Partida
- [x] `F2.3.1` struct `PartidaManager` — gerencia múltiplas partidas + jogadores (mapas com RWMutex)
- [x] `F2.3.2` `CriarPartida(jogadorId)` — gera gameId, status AGUARDANDO_JOGADORES, criador entra como 1º jogador
- [x] `F2.3.3` `EntrarNaPartida(gameId, jogadorId)` — validações: jogo existe, não cheio, não iniciado. Emite JOGADOR_ENTROU.
- [x] `F2.3.4` `IniciarPartida(gameId)` — início explícito (≥2 jogadores). `IniciarPartidaSeCheia()` — auto-start quando atinge 4 jogadores.
- [x] `F2.3.5` Distribuição: 7 cartas por jogador. Primeira carta do descarte deve ser numérica (CORINGA/+4 devolvidos).
- [x] `F2.3.6` `calcularProximoJogador()` — sentido HORARIO/ANTI_HORARIO + pulos, índice circular

### F2.4 — Efeitos Especiais (Seção 8)
- [x] `F2.4.1` PULAR: próximo jogador perde a vez (pula 1)
- [x] `F2.4.2` INVERTER: troca HORARIO ↔ ANTI_HORARIO
- [x] `F2.4.3` Regra 2 jogadores: INVERTER funciona como PULAR (oponente perde a vez)
- [x] `F2.4.4` MAIS_DOIS: próximo compra 2 cartas e perde a vez
- [x] `F2.4.5` MAIS_QUATRO: próximo compra 4 cartas, perde a vez, corAtual = corEscolhida
- [x] `F2.4.6` CORINGA: corAtual = corEscolhida

### F2.5 — Regras UNO e Bater (Seção 8)
- [x] `F2.5.1` `ChamarUno(jogadorId)` — marca chamouUno=true, emite UNO_CHAMADO
- [x] `F2.5.2` `verificarPenalidadesUno()` — jogador com 1 carta sem chamar UNO recebe +2 cartas quando outro jogador age
- [x] `F2.5.3` `Bater(jogadorId)` — valida 0 cartas, declara vencedor, status=FINALIZADO, incrementa vitórias
- [x] `F2.5.4` Compra passa a vez: `passouAVez=true`, carta comprada não pode ser jogada imediatamente

### F2.6 — Sistema de Eventos (Seção 7.11)
- [x] `F2.6.1` Eventos registrados em toda operação que altera estado (entrada, início, jogada, compra, uno, penalidade, bater, finalização)
- [x] `F2.6.2` `versaoEstado` incrementado a cada mudança. `sequencia` monotônica por jogo.
- [x] `F2.6.3` Eventos são a fonte da verdade para replicação (Fase 5)

---

## FASE 3 — API REST (12 Endpoints)

**Objetivo:** Implementar todos os endpoints HTTP REST da Seção 7 do contrato usando Gin.
Respostas no formato padrão (Seções 4-5). Status HTTP conforme `model/response.go`.

### F3.1 — Configuração do Servidor
- [x] `F3.1.1` `cmd/server/main.go` — configurar `gin.Engine` com middleware logging + recovery
- [x] `F3.1.2` Porta via flag `--port` (default 8080). servidorId via flag `--id` (default srv-01).
- [x] `F3.1.3` Middleware CORS para permitir clientes externos (qualquer origem)
- [x] `F3.1.4` Middleware recovery customizado: captura panics → JSON `ERRO_INTERNO`

### F3.2 — GET /servidor (Seção 7.1)
- [x] `F3.2.1` Response dados: `servidorId`, `nome`, `versaoContrato` ("1.1"), `status` ("ATIVO"), `lider` (true), `enderecoLider`, `versaoEstadoAtual`
- [x] `F3.2.2` Erros: `SERVIDOR_NAO_E_LIDER`, `ERRO_INTERNO` (contrato Seção 7.1)
- [x] `F3.2.3` Modo single: sempre `lider: true`. Modo cluster (Fase 5): reflete estado real.

### F3.3 — POST /jogadores (Seção 7.2)
- [x] `F3.3.1` Request: `{"nome": "Ana"}`. Valida nome não vazio → `NOME_INVALIDO`.
- [x] `F3.3.2` Gera `jogadorId` único (prefixo "jogador-" + sequencial 3 dígitos).
- [x] `F3.3.3` Response dados: `jogadorId`, `nome`.
- [x] `F3.3.4` Erros: `NOME_INVALIDO`, `ERRO_INTERNO`.

### F3.4 — POST /jogos (Seção 7.3)
- [x] `F3.4.1` Request: `{"jogadorId": "jogador-001"}`.
- [x] `F3.4.2` Cria partida, criador entra automaticamente. Status: `AGUARDANDO_JOGADORES`.
- [x] `F3.4.3` Response dados: `gameId`, `status`.
- [x] `F3.4.4` Erros: `JOGADOR_NAO_ENCONTRADO`, `ERRO_INTERNO`.

### F3.5 — GET /jogos (Seção 7.4)
- [x] `F3.5.1` Lista todas as partidas existentes (qualquer status).
- [x] `F3.5.2` Cada item: `gameId`, `status`, `quantidadeJogadores`, `maxJogadores` (4).
- [x] `F3.5.3` Erros: `ERRO_INTERNO`.

### F3.6 — POST /jogos/{gameId}/entrar (Seção 7.5)
- [x] `F3.6.1` Request: `{"jogadorId": "jogador-002"}`.
- [x] `F3.6.2` Validações: jogo existe, não cheio, não iniciado. Jogador já está na partida → ok silencioso.
- [x] `F3.6.3` Auto-start quando partida atinge 4 jogadores (`IniciarPartidaSeCheia`).
- [x] `F3.6.4` Response dados: `gameId`, `status`, `quantidadeJogadores`.
- [x] `F3.6.5` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGO_CHEIO`, `JOGO_JA_INICIADO`.

### F3.7 — GET /jogos/{gameId}/estado (Seção 7.6)
- [x] `F3.7.1` Query param: `?jogadorId=jogador-001` (obrigatório, pois é GET).
- [x] `F3.7.2` Response dados: `gameId`, `status`, `versaoEstado`, `jogadorDaVez`, `sentido`, `corAtual`, `cartaTopo`, `minhaMao`, `jogadores[]`, `vencedor`.
- [x] `F3.7.3` `jogadores[]`: apenas `jogadorId`, `nome`, `quantidadeCartas`, `chamouUno` (NUNCA expor mão).
- [x] `F3.7.4` `minhaMao`: array de cartas do jogador solicitante (privado).
- [x] `F3.7.5` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `ERRO_INTERNO`.

### F3.8 — POST /jogos/{gameId}/jogarCarta (Seção 7.7)
- [x] `F3.8.1` Request: `{"jogadorId", "cartaId", "corEscolhida"}`. corEscolhida: null p/ normais, string p/ CORINGA/+4.
- [x] `F3.8.2` Validações (ordem): jogo existe → jogador existe → é sua vez → carta na mão → jogada válida → corEscolhida se necessário → jogo não finalizado.
- [x] `F3.8.3` Remove carta da mão, coloca no monteDescarte, atualiza cartaTopo e corAtual.
- [x] `F3.8.4` Aplica efeito especial (PULAR/INVERTER/MAIS_DOIS/MAIS_QUATRO/CORINGA).
- [x] `F3.8.5` Verifica penalidade UNO (outros jogadores com 1 carta sem chamar).
- [x] `F3.8.6` Response dados: `gameId`, `versaoEstado`, `proximoJogador`, `corAtual`.
- [x] `F3.8.7` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `NAO_E_SUA_VEZ`, `CARTA_NAO_ENCONTRADA`, `JOGADA_INVALIDA`, `COR_OBRIGATORIA`, `JOGO_FINALIZADO`.

### F3.9 — POST /jogos/{gameId}/comprar (Seção 7.8)
- [x] `F3.9.1` Request: `{"jogadorId": "jogador-001"}`.
- [x] `F3.9.2` Compra 1 carta do monteCompra (recicla descarte se vazio). Adiciona à mão.
- [x] `F3.9.3` Passa a vez (`passouAVez: true`). Carta comprada NÃO pode ser jogada imediatamente.
- [x] `F3.9.4` Response dados: `cartaComprada` (objeto Carta completo), `passouAVez` (true), `proximoJogador`.
- [x] `F3.9.5` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `NAO_E_SUA_VEZ`, `JOGO_FINALIZADO`.

### F3.10 — POST /jogos/{gameId}/uno (Seção 7.9)
- [x] `F3.10.1` Request: `{"jogadorId": "jogador-001"}`.
- [x] `F3.10.2` Valida que jogador tem exatamente 1 carta na mão.
- [x] `F3.10.3` Marca `chamouUno = true`. Emite evento `UNO_CHAMADO`.
- [x] `F3.10.4` Response dados: `jogadorId`, `quantidadeCartas` (1).
- [x] `F3.10.5` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGADOR_NAO_ESTA_COM_UMA_CARTA`, `JOGO_FINALIZADO`.

### F3.11 — POST /jogos/{gameId}/bater (Seção 7.10)
- [x] `F3.11.1` Request: `{"jogadorId": "jogador-001"}`.
- [x] `F3.11.2` Valida que jogador tem 0 cartas na mão (já jogou a última).
- [x] `F3.11.3` Declara vencedor: status=FINALIZADO, incrementa vitórias, emite JOGADOR_BATEU e JOGO_FINALIZADO.
- [x] `F3.11.4` Response dados: `vencedor` (jogadorId), `status` ("FINALIZADO").
- [x] `F3.11.5` Erros: `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGADOR_AINDA_TEM_CARTAS`, `JOGO_FINALIZADO`.

### F3.12 — GET /jogos/{gameId}/eventos (Seção 7.11)
- [x] `F3.12.1` Query param: `?desde=0` (sequencia a partir da qual retornar eventos. Exclusivo: > desde).
- [x] `F3.12.2` Response dados: array de `{sequencia, tipo, jogadorId, mensagem, versaoEstado}`.
- [x] `F3.12.3` Se `desde` omitido ou ≤ 0, retorna todos os eventos do jogo.
- [x] `F3.12.4` Erros: `JOGO_NAO_ENCONTRADO`, `ERRO_INTERNO`.

### F3.13 — GET /leaderboard (Seção 7.12)
- [x] `F3.13.1` Response dados: array de `{jogadorId, nome, vitorias}` ordenado por vitórias decrescente.
- [x] `F3.13.2` Leaderboard deve ser compartilhado entre servidores do mesmo grupo (Fase 5).
- [x] `F3.13.3` Erros: `ERRO_INTERNO`.

### F3.14 — Middleware de Líder (preparação para Fase 5)
- [x] `F3.14.1` Middleware que verifica se servidor é líder para endpoints de escrita (jogarCarta, comprar, uno, bater, entrar, criar).
- [x] `F3.14.2` Modo single (Fase 3): middleware é no-op (sempre líder). Modo cluster (Fase 5): bloqueia e retorna `SERVIDOR_NAO_E_LIDER` com endereço do líder.

---

## FASE 4 — Cliente Terminal

**Objetivo:** Interface CLI que consome a API REST. Códigos curtos (Seção 1: R3, B5, YX) no terminal, JSON oficial na comunicação HTTP.

### F4.1 — Cliente HTTP
- [x] `F4.1.1` Cliente HTTP genérico com GET/POST, Content-Type JSON.
- [x] `F4.1.2` Endereço do servidor via flag `--server` ou env `SERVER_URL` (default: `http://localhost:8080`).
- [x] `F4.1.3` Timeout 5s. Tratamento de erros de conexão e servidor indisponível.
- [x] `F4.1.4` Parse automático de respostas JSON (sucesso/erro). Exibição amigável de erros.

### F4.2 — Mapeamento de Códigos Curtos (Seção 1 e p.16-17 do contrato)
- [x] `F4.2.1` Tabela de códigos: Cor (1º dígito: R/B/G/Y) + Tipo (2º dígito: 0-9 núm, S=Skip, R=Reverse, Z=+2, X=Coringa, Y=+4).
  - `R0`-`R9`: vermelha 0-9 | `B0`-`B9`: azul | `G0`-`G9`: verde | `Y0`-`Y9`: amarela
  - `RS`/`BS`/`GS`/`YS`: PULAR | `RR`/`BR`/`GR`/`YR`: INVERTER
  - `RZ`/`BZ`/`GZ`/`YZ`: MAIS_DOIS | `RX`/`BX`/`GX`/`YX`: CORINGA (cor escolhida = 1º dígito)
  - `RY`/`BY`/`GY`/`YY`: MAIS_QUATRO (cor escolhida = 1º dígito)
- [x] `F4.2.2` `ParsearCodigoCurto(codigo)` → busca carta na mão que corresponde ao código curto.
- [x] `F4.2.3` `FormatarCartaCurta(carta)` → exibe carta no formato curto colorido.

### F4.3 — Sessão do Cliente
- [x] `F4.3.1` Armazenar `jogadorId` da sessão atual.
- [x] `F4.3.2` Armazenar `gameId` da partida atual.
- [x] `F4.3.3` Polling automático do estado após cada ação.

### F4.4 — Comandos do Terminal
- [x] `F4.4.1` `criar <nome>` — POST /jogadores
- [x] `F4.4.2` `novo` — POST /jogos (cria partida)
- [x] `F4.4.3` `listar` — GET /jogos (lista partidas)
- [x] `F4.4.4` `entrar <gameId>` — POST /jogos/{gameId}/entrar
- [x] `F4.4.5` `jogar <codigo>` — POST /jogos/{gameId}/jogarCarta (ex: `jogar R3`, `jogar GX`)
- [x] `F4.4.6` `comprar` — POST /jogos/{gameId}/comprar
- [x] `F4.4.7` `uno` — POST /jogos/{gameId}/uno
- [x] `F4.4.8` `bater` — POST /jogos/{gameId}/bater
- [x] `F4.4.9` `estado` — GET /jogos/{gameId}/estado (exibe mão + mesa)
- [x] `F4.4.10` `eventos [desde]` — GET /jogos/{gameId}/eventos
- [x] `F4.4.11` `ranking` — GET /leaderboard
- [x] `F4.4.12` `status` — GET /servidor
- [x] `F4.4.13` `ajuda` — lista comandos disponíveis
- [x] `F4.4.14` `sair` — encerra o cliente

### F4.5 — Exibição e Interface
- [x] `F4.5.1` Prompt interativo: `uno>`
- [x] `F4.5.2` Mão do jogador com índices e códigos curtos coloridos (ANSI).
- [x] `F4.5.3` Mesa: carta do topo, cor atual, sentido, jogador da vez.
- [x] `F4.5.4` Outros jogadores: nome + quantidade de cartas + indicador UNO.
- [x] `F4.5.5` Auto-exibição do estado após cada ação.

---

## FASE 5 — Replicação e Failover

**Objetivo:** Arquitetura multi-servidor com eleição de líder, replicação de estado e continuidade após queda.
Conforme Seção 10 do contrato e requisitos do Estrutura PDF:
- Continuidade da partida após queda do líder (obrigatório)
- Leaderboard compartilhado entre servidores do mesmo grupo (obrigatório)
- Retomada após queda total de todos os servidores (bônus)

### F5.1 — Descoberta de Peers
- [x] `F5.1.1` Lista de peers via env `PEERS` (ex: `http://localhost:8081,http://localhost:8082`).
- [x] `F5.1.2` `servidorId` único por instância (flag `--id`, ex: srv-01, srv-02, srv-03).
- [x] `F5.1.3` Estrutura `ClusterState`: peers ativos, inativos, líder atual, enderecoLider.

### F5.2 — Heartbeat
- [x] `F5.2.1` Goroutine: GET /servidor para cada peer a cada 2s.
- [x] `F5.2.2` Timeout 6s sem resposta → peer marcado inativo.
- [x] `F5.2.3` Líder inativo → dispara eleição.
- [x] `F5.2.4` Peer inativo que volta → reconciliação de estado via eventos.

### F5.3 — Eleição de Líder (Bully Simplificado)
- [x] `F5.3.1` Maior `servidorId` ativo vence.
- [x] `F5.3.2` Ao detectar líder inativo: consulta todos os peers, determina novo líder.
- [x] `F5.3.3` Se este servidor for eleito: assume liderança, emite `LIDER_ALTERADO`.
- [x] `F5.3.4` Se outro for eleito: atualiza `enderecoLider`.

### F5.4 — Replicação de Estado (via Eventos)
- [x] `F5.4.1` Fonte da verdade: sequência de eventos. Estado derivado do replay determinístico.
- [x] `F5.4.2` Seguidor faz polling do líder: `GET /_replicacao/jogos` + `GET /_replicacao/jogos/{gameId}`.
- [x] `F5.4.3` Seguidor aplica snapshots recebidos para reconstruir estado local.
- [x] `F5.4.4` Escrita sempre no líder. Leitura pode ser local (consistência eventual).

### F5.5 — Failover e Continuidade
- [x] `F5.5.1` Líder cai → detectado via heartbeat → eleição → novo líder assume.
- [x] `F5.5.2` Registrar evento `LIDER_ALTERADO` no log.
- [x] `F5.5.3` Partida continua do estado replicado no novo líder.
- [x] `F5.5.4` Cliente que tentar escrever em seguidor recebe `SERVIDOR_NAO_E_LIDER` + `enderecoLider`.

### F5.6 — Leaderboard Compartilhado
- [x] `F5.6.1` Vitórias registradas em qualquer servidor são replicadas para os peers.
- [x] `F5.6.2` GET /leaderboard retorna ranking consolidado de todos os servidores do grupo.
- [x] `F5.6.3` Sincronização de jogadores (jogadorId, nome, vitórias) entre peers.

---

## FASE 6 — Documentação e Testes

**Objetivo:** Garantir conformidade com o contrato e documentar para integração entre grupos (Seção 10 + Estrutura PDF).

### F6.1 — Documentação Técnica (Estrutura PDF)
- [ ] `F6.1.1` Documentar a API completa (12 endpoints, request/response, erros).
- [ ] `F6.1.2` Documentar o formato das mensagens JSON (Seções 3-5 do contrato).
- [ ] `F6.1.3` Documentar a arquitetura do sistema (diagrama, componentes, fluxo).
- [ ] `F6.1.4` Documentar estratégia de replicação (eventos, polling, consistência).
- [ ] `F6.1.5` Documentar estratégia de eleição de líder (Bully simplificado).
- [ ] `F6.1.6` Documentar persistência (em memória, sem banco externo).
- [ ] `F6.1.7` Instruções para clientes de outros grupos se conectarem:
  - Endereço, porta, como iniciar servidor, como iniciar cliente
  - Como criar jogador, criar partida, entrar, jogar, comprar, chamar UNO, bater
  - Como consultar estado, eventos, leaderboard
  - Como testar failover

### F6.2 — Testes Unitários
- [ ] `F6.2.1` Baralho: criação (108 cartas), embaralhamento, compra, reciclagem do descarte.
- [ ] `F6.2.2` Validação de jogadas: todas as combinações válidas e inválidas.
- [ ] `F6.2.3` Efeitos especiais: PULAR, INVERTER (2 e 3+ jogadores), MAIS_DOIS, MAIS_QUATRO, CORINGA.
- [ ] `F6.2.4` Regras UNO: chamar UNO, penalidade (+2), bater (0 cartas).

### F6.3 — Testes de Integração
- [ ] `F6.3.1` Todos os 12 endpoints: cenários de sucesso e erro para cada um.
- [ ] `F6.3.2` Fluxo completo: criar jogadores → criar partida → entrar → auto-start → jogar → UNO → bater → leaderboard.
- [ ] `F6.3.3` Partida com 2 jogadores (INVERTER = PULAR), 3 e 4 jogadores.

### F6.4 — Testes de Failover
- [ ] `F6.4.1` Iniciar 3 instâncias, verificar eleição de líder.
- [ ] `F6.4.2` Derrubar líder, verificar eleição de novo líder + evento LIDER_ALTERADO.
- [ ] `F6.4.3` Verificar replicação: partida criada no líder aparece nos seguidores.
- [ ] `F6.4.4` Verificar redirecionamento: escrita em seguidor → `SERVIDOR_NAO_E_LIDER` + endereço líder.
- [ ] `F6.4.5` Verificar continuidade: partida em andamento sobrevive à queda do líder.

---

## Resumo de Artefatos do Projeto

| Arquivo | Descrição |
|---------|-----------|
| `README.md` | Visão geral, instruções rápidas para outros grupos |
| `ARCHITECTURE.md` | Arquitetura detalhada (componentes, fluxos, decisões) |
| `ROADMAP.md` | Este arquivo — plano de implementação por fases |
| `AGENTS.md` | Instruções para agentes de IA no repositório |
| `Contrato Mínimo API - Uno.pdf` | Contrato oficial V1.1 — endpoints, modelos, regras |
| `Estrutura do trabalho final.pdf` | Requisitos do trabalho — distribuição, documentação, integração |
| `go.mod` / `go.sum` | Dependências Go |
| `cmd/server/main.go` | Entrypoint do servidor REST |
| `cmd/client/main.go` | Entrypoint do cliente terminal |
| `internal/` | Código fonte (model, game, api, replication, client) |

---

## Convenções do Projeto

1. **Nomes em português** para entidades de negócio (campos JSON, enums). **camelCase/PascalCase** para identificadores Go.
2. **JSON tags** em português (ex: `json:"jogadorId"`, `json:"corEscolhida"`).
3. **Valores de enums** idênticos ao contrato (ex: `"EM_ANDAMENTO"`, `"HORARIO"`, `"JOGADA_INVALIDA"`).
4. **Sem banco de dados externo** — estado mantido em memória (mapas com `sync.RWMutex`).
5. **Validação server-side** — cliente NUNCA decide se jogada é válida.
6. **Mão privada** — outros jogadores só veem `quantidadeCartas`, nunca as cartas.
7. **Contrato é a fonte suprema** — qualquer ambiguidade, seguir os PDFs.
8. **Commit em português**, formato convencional (`feat:`, `fix:`, `test:`, `docs:`).
