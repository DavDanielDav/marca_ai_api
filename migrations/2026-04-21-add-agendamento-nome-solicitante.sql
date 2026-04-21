ALTER TABLE arena.agendamentos
  ADD COLUMN IF NOT EXISTS nome_solicitante VARCHAR(255);
