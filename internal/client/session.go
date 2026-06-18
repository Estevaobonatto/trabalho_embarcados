package client

type Session struct {
	JogadorId string
	Nome      string
	GameId    string
	API       *APIClient
}

func NewSession(api *APIClient) *Session {
	return &Session{API: api}
}

func (s *Session) EstaEmPartida() bool {
	return s.GameId != "" && s.JogadorId != ""
}

func (s *Session) TemJogador() bool {
	return s.JogadorId != ""
}
