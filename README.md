# UNO API Distribuído

Jogo de cartas UNO online com arquitetura distribuída — servidor REST (Go + Gin), cliente terminal, replicação de estado e failover.

---

## Como iniciar o servidor

```bash
# Modo standalone (única instância)
go run ./cmd/server --port 8080

# Modo cluster (3 instâncias com failover)
# Terminal 1
$env:PEERS="http://localhost:8081,http://localhost:8082"
go run ./cmd/server --port 8080 --id srv-03

# Terminal 2
$env:PEERS="http://localhost:8080,http://localhost:8082"
go run ./cmd/server --port 8081 --id srv-02

# Terminal 3
$env:PEERS="http://localhost:8080,http://localhost:8081"
go run ./cmd/server --port 8082 --id srv-01
```

## Como iniciar o cliente

```bash
go run ./cmd/client --server http://localhost:8080
```

## Comandos do cliente

| Comando | Descrição |
|---------|-----------|
| `criar <nome>` | Criar jogador |
| `novo` | Criar nova partida |
| `listar` | Listar partidas disponíveis |
| `entrar <gameId>` | Entrar em uma partida |
| `jogar <codigo>` | Jogar carta (ex: `R3`, `B5`, `GS`, `GX`) |
| `comprar` | Comprar carta do monte |
| `uno` | Chamar UNO |
| `bater` | Bater (vencer) |
| `estado` | Ver estado da partida |
| `eventos [desde]` | Ver eventos |
| `ranking` | Ver leaderboard |
| `status` | Ver informações do servidor |
| `ajuda` | Listar todos os comandos |

## Códigos de carta

| Código | Carta |
|--------|-------|
| R0-R9 | Vermelha 0-9 |
| B0-B9 | Azul 0-9 |
| G0-G9 | Verde 0-9 |
| Y0-Y9 | Amarela 0-9 |
| RS/BS/GS/YS | Pular (Skip) |
| RR/BR/GR/YR | Inverter (Reverse) |
| RZ/BZ/GZ/YZ | +2 |
| RX/BX/GX/YX | Coringa (cor escolhida = 1º dígito) |
| RY/BY/GY/YY | +4 (cor escolhida = 1º dígito) |

## API REST — 12 Endpoints

### GET /servidor
Retorna informações do servidor.

```json
{
  "sucesso": true,
  "dados": {
    "servidorId": "srv-01",
    "nome": "Servidor UNO",
    "versaoContrato": "1.1",
    "status": "ATIVO",
    "lider": true,
    "enderecoLider": "http://localhost:8080",
    "versaoEstadoAtual": 15
  }
}
```

### POST /jogadores
Cria um jogador.

**Request:** `{"nome": "Ana"}`
**Response:** `{"sucesso": true, "dados": {"jogadorId": "jogador-001", "nome": "Ana"}}`
**Erros:** `NOME_INVALIDO`, `ERRO_INTERNO`

### POST /jogos
Cria uma partida. O criador entra automaticamente.

**Request:** `{"jogadorId": "jogador-001"}`
**Response:** `{"sucesso": true, "dados": {"gameId": "jogo-001", "status": "AGUARDANDO_JOGADORES"}}`
**Erros:** `JOGADOR_NAO_ENCONTRADO`, `ERRO_INTERNO`

### GET /jogos
Lista todas as partidas.

**Response:** `{"sucesso": true, "dados": [{"gameId": "jogo-001", "status": "AGUARDANDO_JOGADORES", "quantidadeJogadores": 2, "maxJogadores": 4}]}`

### POST /jogos/{gameId}/entrar
Entra em uma partida. Partida inicia automaticamente ao atingir 4 jogadores.

**Request:** `{"jogadorId": "jogador-002"}`
**Response:** `{"sucesso": true, "dados": {"gameId": "jogo-001", "status": "EM_ANDAMENTO", "quantidadeJogadores": 4}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGO_CHEIO`, `JOGO_JA_INICIADO`

### GET /jogos/{gameId}/estado?jogadorId=
Retorna o estado visível da partida para um jogador.

**Response:** `{"sucesso": true, "dados": {"gameId": "...", "status": "...", "versaoEstado": 10, "jogadorDaVez": "jogador-002", "sentido": "HORARIO", "corAtual": "AZUL", "cartaTopo": {...}, "minhaMao": [...], "jogadores": [{"jogadorId": "...", "nome": "...", "quantidadeCartas": 4, "chamouUno": false}], "vencedor": null}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `ERRO_INTERNO`

### POST /jogos/{gameId}/jogarCarta
Realiza uma jogada. Para cartas normais, `corEscolhida` é `null`. Para CORINGA/+4, é obrigatório.

**Request:** `{"jogadorId": "jogador-001", "cartaId": "carta-011", "corEscolhida": null}`
**Response:** `{"sucesso": true, "dados": {"gameId": "jogo-001", "versaoEstado": 11, "proximoJogador": "jogador-002", "corAtual": "VERDE"}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `NAO_E_SUA_VEZ`, `CARTA_NAO_ENCONTRADA`, `JOGADA_INVALIDA`, `COR_OBRIGATORIA`, `JOGO_FINALIZADO`

### POST /jogos/{gameId}/comprar
Compra uma carta. A carta comprada não pode ser jogada imediatamente.

**Request:** `{"jogadorId": "jogador-001"}`
**Response:** `{"sucesso": true, "dados": {"cartaComprada": {...}, "passouAVez": true, "proximoJogador": "jogador-003"}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `NAO_E_SUA_VEZ`, `JOGO_FINALIZADO`

### POST /jogos/{gameId}/uno
Chama UNO. Jogador deve ter exatamente 1 carta.

**Request:** `{"jogadorId": "jogador-001"}`
**Response:** `{"sucesso": true, "dados": {"jogadorId": "jogador-001", "quantidadeCartas": 1}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGADOR_NAO_ESTA_COM_UMA_CARTA`, `JOGO_FINALIZADO`

### POST /jogos/{gameId}/bater
Confirma vitória. Jogador deve ter 0 cartas.

**Request:** `{"jogadorId": "jogador-001"}`
**Response:** `{"sucesso": true, "dados": {"vencedor": "jogador-001", "status": "FINALIZADO"}}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `JOGADOR_NAO_ENCONTRADO`, `JOGADOR_AINDA_TEM_CARTAS`, `JOGO_FINALIZADO`

### GET /jogos/{gameId}/eventos?desde=
Retorna eventos da partida a partir de uma sequência.

**Response:** `{"sucesso": true, "dados": [{"sequencia": 1, "tipo": "JOGADOR_ENTROU", "jogadorId": "jogador-001", "mensagem": "Ana entrou na partida", "versaoEstado": 1}]}`
**Erros:** `JOGO_NAO_ENCONTRADO`, `ERRO_INTERNO`

### GET /leaderboard
Retorna ranking de vitórias.

**Response:** `{"sucesso": true, "dados": [{"jogadorId": "jogador-001", "nome": "Ana", "vitorias": 3}]}`

---

## Códigos de erro

| Código | HTTP Status |
|--------|-------------|
| `JOGO_NAO_ENCONTRADO` | 404 |
| `JOGADOR_NAO_ENCONTRADO` | 404 |
| `CARTA_NAO_ENCONTRADA` | 404 |
| `JOGO_CHEIO` | 400 |
| `JOGO_JA_INICIADO` | 400 |
| `NAO_E_SUA_VEZ` | 400 |
| `JOGADA_INVALIDA` | 400 |
| `COR_OBRIGATORIA` | 400 |
| `JOGADOR_NAO_ESTA_COM_UMA_CARTA` | 400 |
| `JOGADOR_AINDA_TEM_CARTAS` | 400 |
| `JOGO_FINALIZADO` | 400 |
| `NOME_INVALIDO` | 400 |
| `SERVIDOR_NAO_E_LIDER` | 409 |
| `ERRO_INTERNO` | 500 |

---

## Como testar failover

```bash
# Iniciar 3 instâncias
# Terminal 1 (será líder — maior ID)
$env:PEERS="http://localhost:8081,http://localhost:8082"
go run ./cmd/server --port 8080 --id srv-03

# Terminal 2
$env:PEERS="http://localhost:8080,http://localhost:8082"
go run ./cmd/server --port 8081 --id srv-02

# Terminal 3
$env:PEERS="http://localhost:8080,http://localhost:8081"
go run ./cmd/server --port 8082 --id srv-01

# Criar jogo no líder (porta 8080)
curl -X POST http://localhost:8080/jogadores -d '{"nome":"Ana"}'
curl -X POST http://localhost:8080/jogos -d '{"jogadorId":"jogador-001"}'

# Verificar replicação nos seguidores
curl http://localhost:8081/jogos
curl http://localhost:8082/jogos

# Derrubar o líder (Ctrl+C no Terminal 1)
# Novo líder será eleito (srv-02, porta 8081)
# Partida continua do estado replicado

# Verificar redirecionamento (escrita em seguidor)
curl -X POST http://localhost:8082/jogadores -d '{"nome":"Teste"}'
# Resposta: SERVIDOR_NAO_E_LIDER com endereço do líder
```

---

## Arquitetura

```
Clientes (Terminal CLI) ──► Servidores REST (Gin + Go)
                               │
                    ┌──────────┼──────────┐
                    │          │          │
                  srv-03    srv-02    srv-01
                 (LÍDER)  (seguidor) (seguidor)
                    │          │          │
                    └── Heartbeat + Replicação ──┘
```

- **Linguagem:** Go 1.26
- **Framework HTTP:** Gin
- **Persistência:** Em memória (mapas com `sync.RWMutex`)
- **Comunicação:** HTTP REST + JSON
- **Replicação:** Polling de snapshots a cada 2s (endpoint `/_replicacao`)
- **Eleição de líder:** Bully simplificado (maior `servidorId` ativo)
- **Heartbeat:** GET /servidor a cada 2s, timeout 6s
- **Redirecionamento:** Escrita em seguidor → HTTP 409 `SERVIDOR_NAO_E_LIDER`

---

## Testes

```bash
# Todos os testes
go test ./...

# Testes de lógica do jogo
go test -v ./internal/game/...

# Testes de API
go test -v ./internal/api/...

# Com race detector
go test -race ./...
```
