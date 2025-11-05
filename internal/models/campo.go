package models

type Campo struct {
	IDCampo      int    `json:"id_campo"`
	Nome         string `json:"nome"`
	MaxJogadores int    `json:"max_jogadores"`
	Modalidade   string `json:"modalidade"`
	TipoCampo    string `json:"tipo_campo"`
	Imagem       string `json:"imagem"`
	IdArena      int    `json:"id_arena"`
	NomeArena    string `json:"nome_arena,omitempty"`
}
