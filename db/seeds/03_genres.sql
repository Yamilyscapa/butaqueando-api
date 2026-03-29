INSERT INTO app.genres (id, name, created_at)
VALUES
  ('00000000-0000-0000-0000-000000000101', 'Drama', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000102', 'Comedia', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000103', 'Musical', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000104', 'Clasico', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000105', 'Experimental', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000106', 'Familiar', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000107', 'Historico', now() - interval '200 days'),
  ('00000000-0000-0000-0000-000000000108', 'Suspenso', now() - interval '200 days')
ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name;
