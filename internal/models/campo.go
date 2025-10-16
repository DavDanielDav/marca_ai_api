package models

// Jogador
type Campo struct {
	Nome         string `json:"nome_campo"`
	MaxJogadores int    `json:"qtde_jogador"`
	Modalidade   string `json:"modalidade"`
	TipoCampo    string `json:"tipo_campo"`
	Imagem       string `json:"imagem"`
}
