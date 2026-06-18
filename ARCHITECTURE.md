# ARCHITECTURE.md — UNO API Distribuído

> **Fonte da verdade:** Contrato Mínimo API V1.1 — Jogo de Cartas Distribuído UNO (13-06-2026)
> Stack: Go 1.26 + Gin (HTTP REST) + Cliente Terminal

---

## 1. Visão Geral do Sistema

O sistema implementa um jogo de UNO distribuído via API REST HTTP/JSON, composto por:

- **Servidor REST (Go + Gin):** 12 endpoints conforme Seção 7 do contrato. Responsável por validar jogadas, gerenciar estado das partidas e coordenar o jogo.
- **Cliente Terminal (Go CLI):** Interface interativa que consome a API. Suporta códigos curtos (R3, B5, GX...) mas envia JSON oficial na comunicação.
- **Cluster de Servidores (Failover):** Múltiplas instâncias com eleição de líder e replicação de estado via eventos.

```
┌─────────────────────────────────────────────────────────┐
│                      CLIENTES                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ Cliente 1│  │ Cliente 2│  │ Cliente 3│  ... até 4  │
│  │ (CLI Go) │  │ (CLI Go) │  │ (CLI Go) │             │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
│       │HTTP/JSON     │             │                    │
└───────┼──────────────┼─────────────┼────────────────────┘
        │              │             │
        ▼              ▼             ▼
┌─────────────────────────────────────────────────────────┐
│                    CLUSTER DE SERVIDORES                │
│                                                         │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐│
│  │   srv-01     │◄─►│   srv-02     │◄─►│   srv-03     ││
│  │  (LÍDER)     │   │ (seguidor)   │   │ (seguidor)   ││
│  │  porta 8080  │   │  porta 8081  │   │  porta 8082  ││
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘│
│         │                  │                  │         │
│         └──── Heartbeat ───┴─── Replicação ───┘         │
│                Eventos (polling)                         │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Estrutura do Projeto

```
cmd/
├── server/main.go            # Entrypoint: configura Gin, inicia servidor HTTP
└── client/main.go            # Entrypoint: inicia CLI interativa

internal/
├── model/                    # Structs e constantes (DOMÍNIO COMPARTILHADO)
│   ├── card.go               # Carta, Cor, TipoCarta
│   ├── player.go             # Jogador
│   ├── game.go               # Jogo (estado interno completo)
│   ├── event.go              # Evento, TipoEvento
│   └── response.go           # RespostaSucesso[T], RespostaErro, CodigoErro
│
├── game/                     # LÓGICA DO JOGO UNO (stateless)
│   ├── deck.go               # Baralho: criar, embaralhar, comprar, reciclar
│   ├── rules.go              # Validação de jogadas e efeitos
│   ├── match.go              # PartidaManager (CRUD partidas, estado)
│   └── uno_penalty.go        # Regras UNO, penalidade, bater
│
├── api/                      # HANDLERS HTTP (Gin)
│   ├── middleware.go          # CORS, recovery, líder-check
│   ├── server_handler.go     # GET /servidor
│   ├── jogador_handler.go    # POST /jogadores
│   ├── jogo_handler.go       # POST /jogos, GET /jogos
│   ├── jogo_actions.go       # entrar, jogarCarta, comprar, uno, bater
│   ├── estado_handler.go     # GET /jogos/{gameId}/estado
│   ├── eventos_handler.go    # GET /jogos/{gameId}/eventos
│   └── leaderboard_handler.go # GET /leaderboard
│
├── replication/              # FAILOVER E REPLICAÇÃO
│   ├── cluster.go            # ClusterState, lista de peers
│   ├── heartbeat.go          # Goroutine de heartbeat
│   ├── leader_election.go    # Algoritmo de eleição (Bully simplificado)
│   └── state_replication.go  # Replicação via eventos (polling do líder)
│
└── client/                   # CLIENTE TERMINAL
    ├── api_client.go          # Cliente HTTP (GET/POST, parse JSON)
    ├── session.go             # Sessão do cliente (jogadorId, gameId)
    ├── terminal.go            # Loop principal, prompt, comandos
    └── short_codes.go         # Mapeamento códigos curtos ↔ cartas JSON
```

---

## 3. Modelos de Dados

### 3.1 Carta (Seção 3 do contrato)

```
Carta {
    id       string      // "carta-001" até "carta-108"
    cor      Cor         // AMARELO | AZUL | VERMELHO | VERDE | PRETO
    tipo     TipoCarta   // NUMERICA | PULAR | INVERTER | MAIS_DOIS | CORINGA | MAIS_QUATRO
    valor    *string     // "0".."9" para NUMERICA, null para demais
}
```

**Cores:** `AMARELO`, `AZUL`, `VERMELHO`, `VERDE`, `PRETO`
  - PRETO usado exclusivamente para CORINGA e MAIS_QUATRO (Seção 2)
  - Quando CORINGA/MAIS_QUATRO é jogada, a nova cor da rodada é informada via `corEscolhida`

**Tipos:** `NUMERICA`, `PULAR`, `INVERTER`, `MAIS_DOIS`, `CORINGA`, `MAIS_QUATRO`

### 3.2 Jogador (Seção 7.2)

```
Jogador {
    jogadorId  string     // "jogador-001"
    nome       string     // "Ana"
    vitorias   int        // incrementado ao vencer
    mao        []Carta    // PRIVADO — nunca exposto a outros jogadores
    chamouUno  bool       // true se chamou UNO, reset após jogar
}
```

### 3.3 Jogo — Estado Interno (Seção 9 do contrato)

```
Jogo {
    gameId         string        // "jogo-001"
    status         StatusPartida // AGUARDANDO_JOGADORES | EM_ANDAMENTO | FINALIZADO | CANCELADO
    versaoEstado   int           // incrementado a cada mudança
    jogadores      []Jogador     // ordem de jogo (circular)
    jogadorDaVez   string        // jogadorId de quem deve jogar agora
    sentido        Sentido       // HORARIO | ANTI_HORARIO
    corAtual       Cor           // cor da rodada (muda com CORINGA/MAIS_QUATRO)
    cartaTopo      Carta         // última carta do monteDescarte (visível)
    monteCompra    []Carta       // pilha de compra (topo = última posição)
    monteDescarte  []Carta       // pilha de descarte
    eventos        []Evento      // log completo de eventos
    vencedor       *string       // null até alguém vencer
    maxJogadores   int           // sempre 4
}
```

**Exposição ao cliente (Seção 8):**
- `minhaMao`: cartas do jogador solicitante (PRIVADO)
- `jogadores[]`: apenas `jogadorId`, `nome`, `quantidadeCartas`, `chamouUno` (PÚBLICO)
- Demais campos do estado: todos públicos

### 3.4 Evento (Seção 7.11)

```
Evento {
    sequencia     int         // 1, 2, 3... (monotônico por jogo)
    tipo          TipoEvento  // JOGADOR_ENTROU | CARTA_JOGADA | ...
    jogadorId     string      // jogador que gerou o evento
    mensagem      string      // descrição legível
    versaoEstado  int         // versão do estado após este evento
}
```

**Tipos de evento:** `JOGADOR_ENTROU`, `JOGO_INICIADO`, `CARTA_JOGADA`, `CARTA_COMPRADA`, `UNO_CHAMADO`, `PENALIDADE_UNO`, `JOGADOR_BATEU`, `JOGO_FINALIZADO`, `FAILOVER`, `LIDER_ALTERADO`

### 3.5 Respostas Padronizadas (Seções 4 e 5)

**Sucesso:**
```json
{
    "sucesso": true,
    "mensagem": "Operação realizada com sucesso",
    "dados": { ... }
}
```

**Erro:**
```json
{
    "sucesso": false,
    "erro": {
        "codigo": "JOGADA_INVALIDA",
        "mensagem": "A carta jogada não possui a mesma cor, valor ou símbolo da carta do topo."
    }
}
```

---

## 4. API REST — 12 Endpoints (Seção 7)

### Mapa Completo

| # | Método | Endpoint | Seção | Request Body | Response dados |
|---|--------|----------|-------|-------------|----------------|
| 1 | GET | `/servidor` | 7.1 | — | servidorId, nome, versaoContrato, status, lider, enderecoLider, versaoEstadoAtual |
| 2 | POST | `/jogadores` | 7.2 | `{nome}` | jogadorId, nome |
| 3 | POST | `/jogos` | 7.3 | `{jogadorId}` | gameId, status |
| 4 | GET | `/jogos` | 7.4 | — | [ {gameId, status, quantidadeJogadores, maxJogadores} ] |
| 5 | POST | `/jogos/{gameId}/entrar` | 7.5 | `{jogadorId}` | gameId, status, quantidadeJogadores |
| 6 | GET | `/jogos/{gameId}/estado?jogadorId=` | 7.6 | — (query param) | gameId, status, versaoEstado, jogadorDaVez, sentido, corAtual, cartaTopo, minhaMao, jogadores[] |
| 7 | POST | `/jogos/{gameId}/jogarCarta` | 7.7 | `{jogadorId, cartaId, corEscolhida}` | gameId, versaoEstado, proximoJogador, corAtual |
| 8 | POST | `/jogos/{gameId}/comprar` | 7.8 | `{jogadorId}` | cartaComprada, passouAVez, proximoJogador |
| 9 | POST | `/jogos/{gameId}/uno` | 7.9 | `{jogadorId}` | jogadorId, quantidadeCartas |
| 10 | POST | `/jogos/{gameId}/bater` | 7.10 | `{jogadorId}` | vencedor, status |
| 11 | GET | `/jogos/{gameId}/eventos?desde=` | 7.11 | — (query param) | [ {sequencia, tipo, jogadorId, mensagem, versaoEstado} ] |
| 12 | GET | `/leaderboard` | 7.12 | — | [ {jogadorId, nome, vitorias} ] |

### Matriz de Erros por Endpoint

| Código de Erro | /servidor | /jogadores | /jogos (POST) | /jogos (GET) | /entrar | /estado | /jogar | /comprar | /uno | /bater | /eventos | /leaderboard |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `JOGO_NAO_ENCONTRADO` | | | | | x | x | x | x | x | x | x | |
| `JOGADOR_NAO_ENCONTRADO` | | | x | | x | x | x | x | x | x | | |
| `JOGO_CHEIO` | | | | | x | | | | | | | |
| `JOGO_JA_INICIADO` | | | | | x | | | | | | | |
| `NAO_E_SUA_VEZ` | | | | | | | x | x | | | | |
| `CARTA_NAO_ENCONTRADA` | | | | | | | x | | | | | |
| `JOGADA_INVALIDA` | | | | | | | x | | | | | |
| `COR_OBRIGATORIA` | | | | | | | x | | | | | |
| `JOGADOR_NAO_ESTA_COM_UMA_CARTA` | | | | | | | | | x | | | |
| `JOGADOR_AINDA_TEM_CARTAS` | | | | | | | | | | x | | |
| `JOGO_FINALIZADO` | | | | | | | x | x | x | x | | |
| `SERVIDOR_NAO_E_LIDER` | x | | | | | | | | | | | |
| `NOME_INVALIDO` | | x | | | | | | | | | | |
| `ERRO_INTERNO` | x | x | x | x | | | | | | | x | x |

---

## 5. Diagrama de Fluxo Principal

### 5.1 Fluxo de uma Jogada (POST /jogos/{gameId}/jogarCarta)

```
Cliente                          Servidor
  │                                 │
  ├─ POST /jogos/{id}/jogarCarta ──►
  │  {jogadorId, cartaId,           │
  │   corEscolhida}                 │
  │                                 ├─ 1. GET /servidor (seguidor redireciona)
  │                                 ├─ 2. Validar jogo existe
  │                                 ├─ 3. Validar jogador existe na partida
  │                                 ├─ 4. Validar é sua vez (jogadorId == jogadorDaVez)
  │                                 ├─ 5. Validar carta existe na mão do jogador
  │                                 ├─ 6. Validar jogada (rules.go):
  │                                 │     - NUMERICA: cor OU valor igual
  │                                 │     - PULAR/INVERTER/+2: cor OU tipo igual
  │                                 │     - CORINGA/+4: sempre válido
  │                                 ├─ 7. Validar corEscolhida se CORINGA/+4
  │                                 ├─ 8. Remover carta da mão
  │                                 ├─ 9. Colocar carta no monteDescarte, atualizar cartaTopo
  │                                 ├─ 10. Aplicar efeito especial:
  │                                 │      - PULAR → pular próximo
  │                                 │      - INVERTER → trocar sentido
  │                                 │      - +2 → próximo compra 2
  │                                 │      - +4 → próximo compra 4, atualizar corAtual
  │                                 │      - CORINGA → atualizar corAtual
  │                                 ├─ 11. Verificar UNO (1 carta restante):
  │                                 │      - Se não chamou → PENALIDADE: +2 cartas
  │                                 ├─ 12. Verificar vitória (0 cartas)
  │                                 ├─ 13. Calcular próximo jogador
  │                                 ├─ 14. Registrar evento CARTA_JOGADA
  │                                 ├─ 15. Incrementar versaoEstado
  │◄── resposta ────────────────────┤
  │  {sucesso, gameId,              │
  │   versaoEstado,                 │
  │   proximoJogador,               │
  │   corAtual}                     │
```

### 5.2 Fluxo de Compra (POST /jogos/{gameId}/comprar)

```
Cliente                          Servidor
  │                                 │
  ├─ POST /jogos/{id}/comprar ─────►
  │  {jogadorId}                    │
  │                                 ├─ Validar é sua vez
  │                                 ├─ Comprar 1 carta do monteCompra
  │                                 │   Se vazio → reciclar descarte
  │                                 ├─ Adicionar carta à mão do jogador
  │                                 ├─ Passar a vez (passouAVez = true)
  │                                 │   Carta comprada NÃO pode ser jogada
  │                                 ├─ Registrar evento CARTA_COMPRADA
  │◄── resposta ────────────────────┤
  │  {cartaComprada,                │
  │   passouAVez: true,             │
  │   proximoJogador}               │
```

---

## 6. Lógica do Jogo UNO

### 6.1 Composição do Baralho (108 cartas)

| Tipo | Quantidade | Por cor | Total |
|------|-----------|---------|-------|
| NUMERICA "0" | 1 por cor | ×4 cores | 4 |
| NUMERICA "1" a "9" | 2 por cor | ×4 cores | 72 |
| PULAR | 2 por cor | ×4 cores | 8 |
| INVERTER | 2 por cor | ×4 cores | 8 |
| MAIS_DOIS | 2 por cor | ×4 cores | 8 |
| CORINGA | — | PRETO | 4 |
| MAIS_QUATRO | — | PRETO | 4 |
| **Total** | | | **108** |

### 6.2 Regras de Validação

```
ValidarJogada(carta, cartaTopo, corAtual) → error

Switch carta.tipo:
  case NUMERICA:
    if carta.cor != cartaTopo.cor AND carta.valor != cartaTopo.valor:
      return JOGADA_INVALIDA

  case PULAR, INVERTER, MAIS_DOIS:
    if carta.cor != cartaTopo.cor AND carta.tipo != cartaTopo.tipo:
      return JOGADA_INVALIDA

  case CORINGA, MAIS_QUATRO:
    // Sempre válido
    // Mas requer corEscolhida != null (COR_OBRIGATORIA)
```

### 6.3 Efeitos das Cartas Especiais

| Carta | Efeito |
|-------|--------|
| PULAR | Próximo jogador perde a vez. Jogador seguinte joga. |
| INVERTER | `sentido` troca HORARIO↔ANTI_HORARIO. Com 2 jogadores, funciona como PULAR (oponente perde vez). |
| MAIS_DOIS | Próximo jogador compra 2 cartas do monteCompra E perde a vez. |
| CORINGA | `corAtual` = `corEscolhida`. Jogador escolhe qualquer cor. |
| MAIS_QUATRO | Próximo jogador compra 4 cartas E perde a vez. `corAtual` = `corEscolhida`. |

### 6.4 Cálculo do Próximo Jogador

```
ProximoJogador(jogo) → string (jogadorId)

  idxAtual = index de jogadorDaVez na lista
  n = len(jogadores)

  if sentido == HORARIO:
    return jogadores[(idxAtual + 1 + pulos) % n].jogadorId
  else: // ANTI_HORARIO
    return jogadores[(idxAtual - 1 - pulos + n) % n].jogadorId
```

### 6.5 Regras UNO e Penalidade

```
ApósJogada(jogador):
  if len(jogador.mao) == 1 AND !jogador.chamouUno:
    // Penalidade: comprar 2 cartas
    jogador.mao.append(comprarCarta(), comprarCarta())
    registrarEvento(PENALIDADE_UNO, ...)

  if len(jogador.mao) == 0:
    // Não finaliza automaticamente — aguarda POST /bater
```

### 6.6 Início da Partida

```
IniciarPartida(gameId):
  1. Criar baralho (108 cartas)
  2. Embaralhar
  3. Distribuir 7 cartas para cada jogador
  4. Virar primeira carta do monteCompra para monteDescarte
     - Se for CORINGA ou MAIS_QUATRO: devolver, embaralhar, virar outra
     - Se for carta especial numérica 0 na primeira rodada: ok
  5. Definir jogadorDaVez como primeiro da lista
  6. sentido = HORARIO
  7. status = EM_ANDAMENTO
  8. Registrar evento JOGO_INICIADO
```

---

## 7. Arquitetura de Failover (Implementado)

### 7.1 Cluster e Eleição de Líder

**Algoritmo:** Bully simplificado — maior `servidorId` ativo vence.

**Fluxo:**
1. Cada servidor lê a lista de peers da variável de ambiente `PEERS`
2. Na inicialização, cada servidor se declara líder temporariamente
3. Após 4s, eleição inicial: consulta peers ativos, maior ID vence
4. Heartbeat a cada 2s: GET /servidor em cada peer para verificar saúde
5. Timeout de 6s sem resposta → peer marcado inativo
6. Se líder inativo OU peer com ID superior detectado → nova eleição
7. Eleição bem-sucedida → registra evento `LIDER_ALTERADO`

**Estrutura ClusterState:**
```
ClusterState {
    ServidorID, Endereco       // este servidor
    Peers map[URL]*PeerInfo    // peers conhecidos
    IsLider, LiderID, EnderecoLider  // estado da liderança
    PM *PartidaManager         // referência ao estado do jogo
}
```

### 7.2 Replicação de Estado

**Mecanismo:** Snapshot completo via polling (não replay de eventos individuais).

**Fluxo (seguidor a cada 2s):**
1. `GET /_replicacao/jogos` no líder → lista de {gameId, versaoEstado}
2. Para cada jogo com versão superior à local: `GET /_replicacao/jogos/{gameId}` → `JogoSnapshot` completo
3. `ImportarSnapshot()` → reconstrói estado local (jogadores, mãos, baralho, eventos)
4. `GET /leaderboard` no líder → sincroniza vitórias dos jogadores

**Endpoints internos de replicação** (prefixo `/_replicacao`, não fazem parte dos 12 endpoints públicos):
- `GET /_replicacao/jogos` — lista de jogos com versão
- `GET /_replicacao/jogos/{gameId}` — snapshot completo do jogo (inclui mãos)

**Estrutura JogoSnapshot:**
```
JogoSnapshot {
    gameId, status, versaoEstado, jogadorDaVez, sentido, corAtual
    cartaTopo, monteCompra[], monteDescarte[], eventos[], vencedor
    jogadores[] { jogadorId, nome, vitorias, mao[], chamouUno }
}
```

### 7.3 Estratégia de Consistência

- **Escrita sempre no líder:** Middleware `LiderMiddleware` bloqueia POST em seguidores → HTTP 409 `SERVIDOR_NAO_E_LIDER` + `enderecoLider`
- **Leitura local:** GET /estado, GET /eventos, GET /jogos, GET /leaderboard disponíveis localmente (consistência eventual, ~2s de atraso)
- **Leaderboard compartilhado:** Seguidores sincronizam vitórias do líder a cada 2s. Dados consolidados entre servidores do grupo.
- **Continuidade após failover:** Se líder cai, seguidor com estado mais recente é eleito. Partida continua do último snapshot replicado.
- **Contadores resilientes:** `ImportarSnapshot` ajusta contadores locais (`contadorJogador`, `contadorJogo`) para evitar conflitos de ID.

---

## 8. Cliente Terminal

### 8.1 Interface

```
┌──────────────────────────────────────────────┐
│  UNO — Terminal                              │
│  Servidor: http://localhost:8080             │
│  Jogador: Ana (jogador-001)                  │
│  Partida: jogo-001 | Status: EM_ANDAMENTO    │
│──────────────────────────────────────────────│
│  Carta no topo: [VERMELHO 5]                 │
│  Cor atual: VERMELHO | Sentido: HORARIO      │
│  Vez de: Bruno                               │
│──────────────────────────────────────────────│
│  Jogadores:                                  │
│    Ana (você) — 4 cartas                     │
│    Bruno       — 3 cartas                    │
│    Carla       — 5 cartas UNO!               │
│──────────────────────────────────────────────│
│  Sua mão:                                    │
│    [1] R3  [2] B5  [3] GS  [4] YX(coringa)  │
│──────────────────────────────────────────────│
│  uno>                                        │
└──────────────────────────────────────────────┘
```

### 8.2 Tabela de Códigos Curtos

| Código | Carta | JSON |
|--------|-------|------|
| R0-R9 | Vermelha 0-9 | `{"cor":"VERMELHO","tipo":"NUMERICA","valor":"3"}` |
| B0-B9 | Azul 0-9 | `{"cor":"AZUL","tipo":"NUMERICA","valor":"5"}` |
| G0-G9 | Verde 0-9 | `{"cor":"VERDE","tipo":"NUMERICA","valor":"7"}` |
| Y0-Y9 | Amarela 0-9 | `{"cor":"AMARELO","tipo":"NUMERICA","valor":"2"}` |
| RS/BS/GS/YS | Pular | `{"cor":"VERMELHO","tipo":"PULAR"}` |
| RR/BR/GR/YR | Inverter | `{"cor":"AZUL","tipo":"INVERTER"}` |
| RZ/BZ/GZ/YZ | +2 | `{"cor":"VERDE","tipo":"MAIS_DOIS"}` |
| RX/BX/GX/YX | Coringa (com cor) | `{"cor":"PRETO","tipo":"CORINGA","corEscolhida":"AMARELO"}` |
| RY/BY/GY/YY | +4 (com cor) | `{"cor":"PRETO","tipo":"MAIS_QUATRO","corEscolhida":"AZUL"}` |

---

## 9. Segurança e Robustez

| Camada | Medida |
|--------|--------|
| API | Validação de todos os inputs no servidor |
| API | Middleware de recovery para panics |
| API | Timeout de 5s em todas as requisições |
| Jogo | `sync.RWMutex` protege estado de cada partida contra acesso concorrente |
| Jogo | Validação de que carta pertence à mão do jogador que a enviou |
| Jogo | Validação de que é a vez do jogador |
| Failover | Heartbeat com timeout para detecção de falhas |
| Failover | Cliente trata servidor indisponível (reconexão, redirecionamento) |
| Cliente | Reconexão automática com retry exponencial |

---

## 10. Dependências

| Pacote | Versão | Uso |
|--------|--------|-----|
| `github.com/gin-gonic/gin` | ^1.10 | Framework HTTP |
| `math/rand` | stdlib | Embaralhamento do baralho |
| `sync` | stdlib | Mutexes para concorrência |
| `encoding/json` | stdlib | Serialização JSON |
| `net/http` | stdlib | Cliente HTTP para o terminal e replicação |
| `flag` | stdlib | Flags de linha de comando |
| `os` | stdlib | Variáveis de ambiente |
| `bufio` | stdlib | Input do terminal |
| `fmt` | stdlib | Formatação de saída |

**Zero dependências externas além do Gin.**
