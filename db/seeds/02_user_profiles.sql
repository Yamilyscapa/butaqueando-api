INSERT INTO app.user_profiles (
  user_id,
  bio,
  created_at,
  updated_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'Administrador de catalogo y moderacion de comunidad.',
    now() - interval '120 days',
    now() - interval '120 days'
  ),
  (
    '00000000-0000-0000-0000-000000000002',
    'Fan de dramas clasicos y montajes contemporaneos.',
    now() - interval '90 days',
    now() - interval '30 days'
  ),
  (
    '00000000-0000-0000-0000-000000000003',
    'Critico amateur. Siempre buscando nuevas puestas en escena.',
    now() - interval '85 days',
    now() - interval '20 days'
  ),
  (
    '00000000-0000-0000-0000-000000000004',
    'Me encantan los musicales y las funciones familiares.',
    now() - interval '70 days',
    now() - interval '18 days'
  ),
  (
    '00000000-0000-0000-0000-000000000005',
    'Prefiero obras historicas y clasicos del repertorio.',
    now() - interval '65 days',
    now() - interval '14 days'
  ),
  (
    '00000000-0000-0000-0000-000000000006',
    'Curiosa por el teatro experimental y nuevas dramaturgias.',
    now() - interval '55 days',
    now() - interval '8 days'
  )
ON CONFLICT (user_id)
DO UPDATE SET
  bio = EXCLUDED.bio,
  updated_at = EXCLUDED.updated_at;
