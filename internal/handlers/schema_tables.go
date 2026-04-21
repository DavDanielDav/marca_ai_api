package handlers

import "github.com/danpi/marca_ai_backend/internal/config"

func arenaTableName(table string) string {
	return config.QualifiedName(table)
}

func usuarioTableName() string {
	return arenaTableName("usuario")
}

func arenasTableName() string {
	return arenaTableName("arenas")
}

func campoTableName() string {
	return arenaTableName("campo")
}

func agendamentosTableName() string {
	return arenaTableName("agendamentos")
}

func pagamentosPorAgendamentoTableName() string {
	return arenaTableName("pagamentos_por_agendamento")
}

func emailCodesTableName() string {
	return arenaTableName("email_codes")
}

func jogadorTableName(table string) string {
	return config.QualifiedNameForSchema("jogador", table)
}

func usuarioJogadorTableName() string {
	return jogadorTableName("usuario_jogador")
}
