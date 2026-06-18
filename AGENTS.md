# AGENTS.md — UNO API Distribuído

Instruções para agentes de IA que trabalham neste repositório.

---

## 1. Fonte da Verdade

**O contrato PDF é a especificação suprema.** (`Contrato Mínimo API - Uno.pdf`)

Qualquer dúvida ou ambiguidade deve ser resolvida consultando o contrato. Os arquivos `ROADMAP.md` e `ARCHITECTURE.md` são derivados do contrato — se houver conflito, **o contrato vence**.

---

## 2. Idioma e Convenções

- **Código:** identificadores Go em camelCase/PascalCase
- **JSON tags:** em português (ex: `json:"jogadorId"`, `json:"corEscolhida"`)
- **Campos de negócio:** seguir EXATAMENTE o contrato (ex: `gameId`, não `gameID`; `jogadorDaVez`, não `jogador_da_vez`)
- **Enums/Strings:** usar os valores exatos do contrato (ex: `"EM_ANDAMENTO"`, não `"EM ANDAMENTO"`; `"HORARIO"`, não `"horário"`)
- **Comentários e documentação:** português
- **Mensagens de commit:** português, formato convencional (`feat:`, `fix:`, `test:`, `docs:`)

---

## 3. Comandos

```bash
# Inicializar dependências (já feito)
go mod tidy

# Build do servidor
go build -o bin/server.exe ./cmd/server

# Build do cliente
go build -o bin/client.exe ./cmd/client

# Rodar servidor (instância única)
go run ./cmd/server --port 8080

# Rodar servidor (modo cluster — 3 instâncias)
set PEERS=http://localhost:8081,http://localhost:8082 && go run ./cmd/server --port 8080
set PEERS=http://localhost:8080,http://localhost:8082 && go run ./cmd/server --port 8081
set PEERS=http://localhost:8080,http://localhost:8081 && go run ./cmd/server --port 8082

# Rodar cliente
go run ./cmd/client --server http://localhost:8080

# Rodar testes (todos os pacotes)
go test ./...

# Rodar testes com verbose
go test -v ./...

# Rodar testes de um pacote específico
go test -v ./internal/game/...

# Verificar cobertura
go test -cover ./...

# Verificar race conditions
go test -race ./...

# Lint
go vet ./...
```

---

## 4. Estrutura de Diretórios (Resumo)

```
cmd/server/main.go        # Entrypoint servidor REST (Gin)
cmd/client/main.go        # Entrypoint cliente terminal
internal/model/           # Structs, constantes, enums
internal/game/            # Lógica pura do UNO (sem HTTP)
internal/api/             # Handlers Gin (12 endpoints)
internal/replication/     # Failover, eleição de líder, replicação
internal/client/          # Cliente HTTP + terminal CLI
```

**Regra de dependência circular:** `model` ← `game` ← `api` ← `replication`. `api` e `replication` podem depender de `game`. `game` NUNCA importa `api`.

---

## 5. Convenções de Código

### 5.1 Concorrência
- Cada `Jogo` tem seu próprio `sync.RWMutex`
- `PartidaManager` tem um `sync.RWMutex` para o mapa de jogos
- Sempre usar `defer mu.Unlock()` após `mu.Lock()`

### 5.2 IDs
- `jogadorId`: prefixo `"jogador-"` + número sequencial de 3 dígitos
- `gameId`: prefixo `"jogo-"` + número sequencial de 3 dígitos
- `cartaId`: prefixo `"carta-"` + número sequencial de 3 dígitos
- IDs são gerados pelo servidor (NUNCA pelo cliente)

### 5.3 Respostas HTTP
- Sucesso: sempre `{"sucesso": true, "mensagem": "...", "dados": {...}}`
- Erro: sempre `{"sucesso": false, "erro": {"codigo": "...", "mensagem": "..."}}`
- Status HTTP: 200 para sucesso, 400 para erro de validação, 404 para não encontrado, 409 para SERVIDOR_NAO_E_LIDER, 500 para erro interno
- Gin context: sempre usar `c.JSON(httpStatus, resposta)`, NUNCA `c.String()` ou `c.XML()`

### 5.4 Validação
- Toda validação de jogo ocorre no SERVIDOR (Seção 8 do contrato)
- O cliente NUNCA decide se uma jogada é válida
- Ordem de validação nos handlers: existência do jogo → existência do jogador → vez do jogador → posse da carta → validade da jogada
- A validação de cor usa `corAtual` (cor efetiva da rodada), não `cartaTopo.Cor`

---

## 6. Regras Críticas do Jogo (Não Esquecer)

1. **Mão privada:** outros jogadores só veem `quantidadeCartas`, nunca a mão
2. **Comprar passa a vez:** `passouAVez: true`, carta comprada não pode ser jogada imediatamente
3. **CORINGA/+4 sempre válido:** pode ser jogado a qualquer momento independente da cor/tipo do topo
4. **corEscolhida obrigatória:** para CORINGA e MAIS_QUATRO (erro: `COR_OBRIGATORIA`)
5. **2 jogadores + INVERTER = PULAR:** inverte não faz sentido com 2 jogadores
6. **UNO antes de jogar:** jogador deve chamar UNO quando fica com 1 carta; se não chamar e outro jogador jogar, penalidade de +2
7. **Bater após jogar:** vencer (0 cartas) requer POST /bater separado após a jogada
8. **Primeira carta do descarte:** deve ser numérica (se vier CORINGA/+4, devolver ao baralho)
9. **Reciclagem do descarte:** quando monteCompra esvazia, pegar monteDescarte (exceto cartaTopo), embaralhar, virar monteCompra
10. **Eventos são a fonte da verdade para replicação:** o estado é derivado da sequência de eventos
11. **Auto-start quando cheio:** partida inicia automaticamente quando atinge 4 jogadores (`IniciarPartidaSeCheia`)

---

## 7. Contrato — Referência Rápida

### 7.1 Vocabulário (Seção 2)

```
Cores:       AMARELO, AZUL, VERMELHO, VERDE, PRETO
Tipos:       NUMERICA, PULAR, INVERTER, MAIS_DOIS, CORINGA, MAIS_QUATRO
Status:      AGUARDANDO_JOGADORES, EM_ANDAMENTO, FINALIZADO, CANCELADO
Sentido:     HORARIO, ANTI_HORARIO
Eventos:     JOGADOR_ENTROU, JOGO_INICIADO, CARTA_JOGADA, CARTA_COMPRADA,
             UNO_CHAMADO, PENALIDADE_UNO, JOGADOR_BATEU, JOGO_FINALIZADO,
             FAILOVER, LIDER_ALTERADO
```

### 7.2 Códigos de Erro (Seção 6)

```
JOGO_NAO_ENCONTRADO          JOGADOR_NAO_ENCONTRADO
JOGO_CHEIO                   JOGO_JA_INICIADO
NAO_E_SUA_VEZ                CARTA_NAO_ENCONTRADA
JOGADA_INVALIDA              COR_OBRIGATORIA
JOGADOR_NAO_ESTA_COM_UMA_CARTA  JOGADOR_AINDA_TEM_CARTAS
JOGO_FINALIZADO              SERVIDOR_NAO_E_LIDER
NOME_INVALIDO                ERRO_INTERNO
```

### 7.3 Modelo de Carta (Seção 3)

```json
{ "id": "carta-001", "cor": "VERMELHO", "tipo": "NUMERICA", "valor": "3" }
{ "id": "carta-002", "cor": "AZUL",    "tipo": "PULAR",    "valor": null }
{ "id": "carta-005", "cor": "PRETO",   "tipo": "CORINGA",  "valor": null }
```

### 7.4 Estado Interno (Seção 9)

```
gameId, status, versaoEstado, jogadores[], jogadorDaVez,
sentido, corAtual, cartaTopo, monteCompra[], monteDescarte[],
eventos[], vencedor
```

---

## 8. Fluxo de Trabalho para Novas Features

1. **Ler o contrato** — verificar se a feature está definida no PDF
2. **Ler ROADMAP.md** — verificar em qual fase/tarefa a feature se encaixa
3. **Ler ARCHITECTURE.md** — verificar diagramas e design decisions
4. **Implementar** — seguindo a ordem: `model` → `game` → `api` → `cmd`
5. **Testar** — `go test -v -race ./...` no pacote relevante
6. **Atualizar ROADMAP.md** — marcar tarefa como concluída `[x]`
7. **Commit** — mensagem descritiva referenciando a tarefa do roadmap

---

## 9. Gotchas Comuns

- **Não use `database/sql` nem SQLite** — tudo em memória conforme contrato
- **Não crie endpoint novo** — o contrato define exatamente 12 endpoints
- **Não altere o formato JSON** — ele é o contrato entre grupos
- **Não exponha a mão de outros jogadores** — erro grave de segurança
- **Não permita jogada do cliente sem validação** — toda validação é server-side
- **Não esqueça de incrementar `versaoEstado`** a cada mudança
- **Não esqueça de registrar evento** em toda operação que altera estado
- **Cuidado com índices circulares no próximo jogador** — usar `% len(jogadores)`
- **Cuidado com 2 jogadores + INVERTER** — funciona como PULAR
- **Mutex por jogo, não global** — evita gargalo entre partidas diferentes
