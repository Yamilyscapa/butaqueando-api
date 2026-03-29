INSERT INTO app.play_genres (play_id, genre_id)
VALUES
  ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000101'),
  ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000104'),
  ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000101'),
  ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000104'),
  ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000103'),
  ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000106'),
  ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000101'),
  ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000107'),
  ('00000000-0000-0000-0000-000000000205', '00000000-0000-0000-0000-000000000105'),
  ('00000000-0000-0000-0000-000000000205', '00000000-0000-0000-0000-000000000108'),
  ('00000000-0000-0000-0000-000000000206', '00000000-0000-0000-0000-000000000102'),
  ('00000000-0000-0000-0000-000000000206', '00000000-0000-0000-0000-000000000105')
ON CONFLICT (play_id, genre_id)
DO NOTHING;

INSERT INTO app.play_cast_members (
  id,
  play_id,
  person_name,
  role_name,
  billing_order,
  created_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000301',
    '00000000-0000-0000-0000-000000000201',
    'Luis Medina',
    'Hamlet',
    1,
    now() - interval '30 days'
  ),
  (
    '00000000-0000-0000-0000-000000000302',
    '00000000-0000-0000-0000-000000000201',
    'Paula Cardenas',
    'Ofelia',
    2,
    now() - interval '30 days'
  ),
  (
    '00000000-0000-0000-0000-000000000303',
    '00000000-0000-0000-0000-000000000203',
    'Ivan Guerra',
    'El Principito',
    1,
    now() - interval '20 days'
  ),
  (
    '00000000-0000-0000-0000-000000000304',
    '00000000-0000-0000-0000-000000000203',
    'Marta Leon',
    'La Rosa',
    2,
    now() - interval '20 days'
  ),
  (
    '00000000-0000-0000-0000-000000000305',
    '00000000-0000-0000-0000-000000000204',
    'Sergio Navas',
    'Novio',
    1,
    now() - interval '14 days'
  ),
  (
    '00000000-0000-0000-0000-000000000306',
    '00000000-0000-0000-0000-000000000204',
    'Claudia Nieto',
    'Novia',
    2,
    now() - interval '14 days'
  )
ON CONFLICT (play_id, person_name, role_name)
DO NOTHING;

INSERT INTO app.play_media (
  id,
  play_id,
  kind,
  url,
  alt_text,
  sort_order,
  created_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000201',
    'poster',
    'https://cdn.butaqueando.local/plays/hamlet-habana/poster.jpg',
    'Poster oficial de Hamlet en la Habana',
    0,
    now() - interval '30 days'
  ),
  (
    '00000000-0000-0000-0000-000000000402',
    '00000000-0000-0000-0000-000000000201',
    'photo',
    'https://cdn.butaqueando.local/plays/hamlet-habana/scene-01.jpg',
    'Escena principal de Hamlet en la Habana',
    1,
    now() - interval '25 days'
  ),
  (
    '00000000-0000-0000-0000-000000000403',
    '00000000-0000-0000-0000-000000000202',
    'poster',
    'https://cdn.butaqueando.local/plays/bernarda-alba/poster.jpg',
    'Poster oficial de La Casa de Bernarda Alba',
    0,
    now() - interval '220 days'
  ),
  (
    '00000000-0000-0000-0000-000000000404',
    '00000000-0000-0000-0000-000000000203',
    'poster',
    'https://cdn.butaqueando.local/plays/principito-musical/poster.jpg',
    'Poster oficial de El Principito Musical',
    0,
    now() - interval '20 days'
  ),
  (
    '00000000-0000-0000-0000-000000000405',
    '00000000-0000-0000-0000-000000000204',
    'poster',
    'https://cdn.butaqueando.local/plays/bodas-de-sangre/poster.jpg',
    'Poster oficial de Bodas de Sangre',
    0,
    now() - interval '14 days'
  ),
  (
    '00000000-0000-0000-0000-000000000406',
    '00000000-0000-0000-0000-000000000205',
    'poster',
    'https://cdn.butaqueando.local/plays/horizonte-roto/poster.jpg',
    'Poster enviado por la comunidad para Horizonte Roto',
    0,
    now() - interval '4 days'
  )
ON CONFLICT (play_id, url)
DO NOTHING;
