BEGIN;

DO $$
BEGIN
	IF to_regclass('public.agendamentos') IS NOT NULL THEN
		ALTER TABLE public.agendamentos
			ADD COLUMN IF NOT EXISTS origem_agendamento VARCHAR(100),
			ADD COLUMN IF NOT EXISTS valor_total NUMERIC(10, 2),
			ADD COLUMN IF NOT EXISTS valor_restante NUMERIC(10, 2);

		ALTER TABLE public.agendamentos
			ALTER COLUMN origem_agendamento SET DEFAULT 'manual',
			ALTER COLUMN valor_total SET DEFAULT 0,
			ALTER COLUMN valor_restante SET DEFAULT 0;

		IF to_regclass('public.campo') IS NOT NULL THEN
			UPDATE public.agendamentos ag
			SET
				origem_agendamento = COALESCE(ag.origem_agendamento, 'manual'),
				valor_total = COALESCE(ag.valor_total, c.valor_hora, 0),
				valor_restante = COALESCE(ag.valor_restante, COALESCE(ag.valor_total, c.valor_hora, 0))
			FROM public.campo c
			WHERE ag.id_campo = c.id_campo;
		END IF;

		UPDATE public.agendamentos
		SET
			origem_agendamento = COALESCE(origem_agendamento, 'manual'),
			valor_total = COALESCE(valor_total, 0),
			valor_restante = COALESCE(valor_restante, COALESCE(valor_total, 0));
	END IF;

	IF to_regclass('arena.agendamentos') IS NOT NULL THEN
		ALTER TABLE arena.agendamentos
			ADD COLUMN IF NOT EXISTS origem_agendamento VARCHAR(100),
			ADD COLUMN IF NOT EXISTS valor_total NUMERIC(10, 2),
			ADD COLUMN IF NOT EXISTS valor_restante NUMERIC(10, 2);

		ALTER TABLE arena.agendamentos
			ALTER COLUMN origem_agendamento SET DEFAULT 'manual',
			ALTER COLUMN valor_total SET DEFAULT 0,
			ALTER COLUMN valor_restante SET DEFAULT 0;

		IF to_regclass('arena.campo') IS NOT NULL THEN
			UPDATE arena.agendamentos ag
			SET
				origem_agendamento = COALESCE(ag.origem_agendamento, 'manual'),
				valor_total = COALESCE(ag.valor_total, c.valor_hora, 0),
				valor_restante = COALESCE(ag.valor_restante, COALESCE(ag.valor_total, c.valor_hora, 0))
			FROM arena.campo c
			WHERE ag.id_campo = c.id_campo;
		END IF;

		UPDATE arena.agendamentos
		SET
			origem_agendamento = COALESCE(origem_agendamento, 'manual'),
			valor_total = COALESCE(valor_total, 0),
			valor_restante = COALESCE(valor_restante, COALESCE(valor_total, 0));
	END IF;
END $$;

COMMIT;
