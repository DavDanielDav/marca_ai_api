package models

// Jogador
type Usuario struct {
	ID       string `json:"id_usuario"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Telefone string `json:"telefone"`
	Senha    string `json:"senha,omitempty"`
}
