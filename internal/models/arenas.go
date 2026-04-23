package models

type Arenas struct {
	ID                 int     `json:"id"`
	Nome               string  `json:"nome"`
	Cnpj               string  `json:"cnpj"`
	QtdCampos          int     `json:"qtdCampos"`
	Tipo               string  `json:"tipo"`
	Imagem             string  `json:"imagem"`
	Endereco           string  `json:"endereco"`
	Observacoes        string  `json:"observacoes"`
	EsportesOferecidos string  `json:"esportes_oferecidos"`
	InformacoesArena   string  `json:"informacoes_arena"`
	EmManutencao       bool    `json:"em_manutencao"`
	Campos             []Campo `json:"campos,omitempty"`
}
