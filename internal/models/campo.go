package models

type Campo struct {
	IDCampo                  int      `json:"id_campo"`
	Nome                     string   `json:"nome_campo"`
	MaxJogadores             int      `json:"max_jogadores"`
	Modalidade               string   `json:"modalidade"`
	TipoCampo                string   `json:"tipo_campo"`
	Imagem                   string   `json:"imagem"`
	ValorHora                float64  `json:"valor_hora"`
	Ativo                    bool     `json:"ativo"`
	EmManutencao             bool     `json:"em_manutencao"`
	Horarios                 []string `json:"horarios,omitempty"`
	HorariosDisponiveis      []string `json:"horarios_disponiveis,omitempty"`
	HorariosDisponiveisCamel []string `json:"horariosDisponiveis,omitempty"`
	HorariosCampo            []string `json:"horarios_campo,omitempty"`
	IdArena                  int      `json:"id_arena"`
	NomeArena                string   `json:"nome_arena,omitempty"`
}
