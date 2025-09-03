package models

// Jogador
type Usuario struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Telefone string `json:"telefone"`
	// Password string `json:"password"` // futuro, se quiser
}

// Dono de arena
type DonoDeArena struct {
	NomeDonoArena string `json:"nome_dono_arena"`
	Cnpj          string `json:"cnpj"`
	Arena         string `json:"arena"`
}
