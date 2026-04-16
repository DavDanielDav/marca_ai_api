BEGIN;

ALTER TABLE public.campo
ADD COLUMN IF NOT EXISTS horarios_disponiveis JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE public.campo
DROP CONSTRAINT IF EXISTS campo_horarios_disponiveis_array_chk;

ALTER TABLE public.campo
ADD CONSTRAINT campo_horarios_disponiveis_array_chk
CHECK (jsonb_typeof(horarios_disponiveis) = 'array');

COMMIT;
