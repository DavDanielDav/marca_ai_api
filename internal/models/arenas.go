package models

type Arenas struct {
	Nome      string `json:"nome"`
	Cnpj      string `json:"cnpj"`
	QtdCampos int    `json:"qtdCampos"`
	Tipo      string `json:"tipo"`
	Imagem    string `json:"imagem"`
	Endereco  string `json:"endereco"`
}
