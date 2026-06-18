package model

import "net/http"

// CodigoErro representa os códigos de erro padronizados conforme Seção 6 do contrato.
type CodigoErro string

const (
	JOGO_NAO_ENCONTRADO            CodigoErro = "JOGO_NAO_ENCONTRADO"
	JOGADOR_NAO_ENCONTRADO         CodigoErro = "JOGADOR_NAO_ENCONTRADO"
	JOGO_CHEIO                     CodigoErro = "JOGO_CHEIO"
	JOGO_JA_INICIADO               CodigoErro = "JOGO_JA_INICIADO"
	NAO_E_SUA_VEZ                  CodigoErro = "NAO_E_SUA_VEZ"
	CARTA_NAO_ENCONTRADA           CodigoErro = "CARTA_NAO_ENCONTRADA"
	JOGADA_INVALIDA                CodigoErro = "JOGADA_INVALIDA"
	COR_OBRIGATORIA                CodigoErro = "COR_OBRIGATORIA"
	JOGADOR_NAO_ESTA_COM_UMA_CARTA CodigoErro = "JOGADOR_NAO_ESTA_COM_UMA_CARTA"
	JOGADOR_AINDA_TEM_CARTAS       CodigoErro = "JOGADOR_AINDA_TEM_CARTAS"
	JOGO_FINALIZADO                CodigoErro = "JOGO_FINALIZADO"
	SERVIDOR_NAO_E_LIDER           CodigoErro = "SERVIDOR_NAO_E_LIDER"
	NOME_INVALIDO                  CodigoErro = "NOME_INVALIDO"
	ERRO_INTERNO                   CodigoErro = "ERRO_INTERNO"
)

// MensagensErro mapeia cada código de erro para sua mensagem padrão.
var MensagensErro = map[CodigoErro]string{
	JOGO_NAO_ENCONTRADO:            "O jogo solicitado não foi encontrado.",
	JOGADOR_NAO_ENCONTRADO:         "O jogador solicitado não foi encontrado.",
	JOGO_CHEIO:                     "O jogo já atingiu o número máximo de jogadores.",
	JOGO_JA_INICIADO:               "O jogo já foi iniciado e não aceita novos jogadores.",
	NAO_E_SUA_VEZ:                  "Não é a sua vez de jogar.",
	CARTA_NAO_ENCONTRADA:           "A carta especificada não foi encontrada na sua mão.",
	JOGADA_INVALIDA:                "A carta jogada não possui a mesma cor, valor ou símbolo da carta do topo.",
	COR_OBRIGATORIA:                "É obrigatório informar a cor escolhida para cartas CORINGA ou MAIS_QUATRO.",
	JOGADOR_NAO_ESTA_COM_UMA_CARTA: "O jogador não está com apenas uma carta para chamar UNO.",
	JOGADOR_AINDA_TEM_CARTAS:       "O jogador ainda possui cartas na mão e não pode bater.",
	JOGO_FINALIZADO:                "O jogo já foi finalizado.",
	SERVIDOR_NAO_E_LIDER:           "Este servidor não é o líder. Redirecione para o líder.",
	NOME_INVALIDO:                  "O nome do jogador é inválido ou está vazio.",
	ERRO_INTERNO:                   "Ocorreu um erro interno no servidor.",
}

// DetalheErro representa a estrutura de erro na resposta.
type DetalheErro struct {
	Codigo   CodigoErro `json:"codigo"`
	Mensagem string     `json:"mensagem"`
}

// RespostaSucesso representa a resposta padrão de sucesso conforme Seção 4 do contrato.
type RespostaSucesso struct {
	Sucesso  bool        `json:"sucesso"`
	Mensagem string      `json:"mensagem,omitempty"`
	Dados    interface{} `json:"dados,omitempty"`
}

// RespostaErro representa a resposta padrão de erro conforme Seção 5 do contrato.
type RespostaErro struct {
	Sucesso bool        `json:"sucesso"`
	Erro    DetalheErro `json:"erro"`
}

// NovaRespostaSucesso cria uma resposta de sucesso padronizada.
func NovaRespostaSucesso(mensagem string, dados interface{}) RespostaSucesso {
	return RespostaSucesso{
		Sucesso:  true,
		Mensagem: mensagem,
		Dados:    dados,
	}
}

// NovaRespostaErro cria uma resposta de erro padronizada.
// Se mensagem for vazia, usa a mensagem padrão do código de erro.
func NovaRespostaErro(codigo CodigoErro, mensagem string) RespostaErro {
	if mensagem == "" {
		if msg, ok := MensagensErro[codigo]; ok {
			mensagem = msg
		}
	}
	return RespostaErro{
		Sucesso: false,
		Erro: DetalheErro{
			Codigo:   codigo,
			Mensagem: mensagem,
		},
	}
}

// StatusHTTP mapeia códigos de erro para status HTTP.
func StatusHTTP(codigo CodigoErro) int {
	switch codigo {
	case JOGO_NAO_ENCONTRADO, JOGADOR_NAO_ENCONTRADO, CARTA_NAO_ENCONTRADA:
		return http.StatusNotFound
	case JOGO_CHEIO, JOGO_JA_INICIADO, NAO_E_SUA_VEZ, JOGADA_INVALIDA,
		COR_OBRIGATORIA, JOGADOR_NAO_ESTA_COM_UMA_CARTA,
		JOGADOR_AINDA_TEM_CARTAS, JOGO_FINALIZADO, NOME_INVALIDO:
		return http.StatusBadRequest
	case SERVIDOR_NAO_E_LIDER:
		return http.StatusConflict
	case ERRO_INTERNO:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
