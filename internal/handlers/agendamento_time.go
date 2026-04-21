package handlers

import "time"

const agendamentoTimezone = "America/Sao_Paulo"

func agendamentoLocation() *time.Location {
	location, err := time.LoadLocation(agendamentoTimezone)
	if err == nil {
		return location
	}

	return time.FixedZone("BRT", -3*60*60)
}

func agendamentoNow() time.Time {
	return time.Now().In(agendamentoLocation())
}
