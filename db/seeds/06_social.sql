INSERT INTO app.reviews (
  id,
  play_id,
  user_id,
  rating,
  title,
  body,
  contains_spoilers,
  status,
  created_at,
  updated_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000003',
    5,
    'Una puesta impecable',
    'Direccion precisa, actuaciones solidas y un cierre memorable.',
    false,
    'published',
    now() - interval '18 days',
    now() - interval '18 days'
  ),
  (
    '00000000-0000-0000-0000-000000000502',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000004',
    4,
    'Muy recomendable',
    'Gran ritmo escenico y una Ofelia poderosa en escena.',
    false,
    'published',
    now() - interval '16 days',
    now() - interval '16 days'
  ),
  (
    '00000000-0000-0000-0000-000000000503',
    '00000000-0000-0000-0000-000000000202',
    '00000000-0000-0000-0000-000000000002',
    5,
    'Clasico bien resuelto',
    'La tension crece escena a escena y la propuesta visual funciona.',
    false,
    'published',
    now() - interval '150 days',
    now() - interval '150 days'
  ),
  (
    '00000000-0000-0000-0000-000000000504',
    '00000000-0000-0000-0000-000000000203',
    '00000000-0000-0000-0000-000000000005',
    4,
    'Ideal para ir en familia',
    'Canciones pegadizas, escenografia colorida y buen desempeno del elenco.',
    false,
    'published',
    now() - interval '9 days',
    now() - interval '9 days'
  ),
  (
    '00000000-0000-0000-0000-000000000505',
    '00000000-0000-0000-0000-000000000203',
    '00000000-0000-0000-0000-000000000006',
    3,
    'Buena energia, final flojo',
    'Disfrutable en general, aunque el ultimo acto pierde fuerza.',
    true,
    'published',
    now() - interval '8 days',
    now() - interval '8 days'
  ),
  (
    '00000000-0000-0000-0000-000000000506',
    '00000000-0000-0000-0000-000000000204',
    '00000000-0000-0000-0000-000000000002',
    4,
    'Lorca con personalidad propia',
    'La musica en vivo aporta mucho y la direccion logra gran intensidad.',
    false,
    'published',
    now() - interval '6 days',
    now() - interval '6 days'
  ),
  (
    '00000000-0000-0000-0000-000000000507',
    '00000000-0000-0000-0000-000000000204',
    '00000000-0000-0000-0000-000000000004',
    2,
    'No conecte con la propuesta',
    'La puesta me parecio irregular y con problemas de ritmo.',
    false,
    'hidden',
    now() - interval '5 days',
    now() - interval '5 days'
  )
ON CONFLICT (id)
DO UPDATE SET
  rating = EXCLUDED.rating,
  title = EXCLUDED.title,
  body = EXCLUDED.body,
  contains_spoilers = EXCLUDED.contains_spoilers,
  status = EXCLUDED.status,
  updated_at = EXCLUDED.updated_at;

INSERT INTO app.review_comments (
  id,
  review_id,
  user_id,
  parent_comment_id,
  body,
  status,
  created_at,
  updated_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000601',
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000002',
    NULL,
    'Totalmente de acuerdo, especialmente con la actuacion de Ofelia.',
    'published',
    now() - interval '17 days',
    now() - interval '17 days'
  ),
  (
    '00000000-0000-0000-0000-000000000602',
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000601',
    'La escena final fue de mis favoritas.',
    'published',
    now() - interval '16 days',
    now() - interval '16 days'
  ),
  (
    '00000000-0000-0000-0000-000000000603',
    '00000000-0000-0000-0000-000000000504',
    '00000000-0000-0000-0000-000000000003',
    NULL,
    'Me sorprendio la calidad musical para una funcion familiar.',
    'published',
    now() - interval '8 days',
    now() - interval '8 days'
  ),
  (
    '00000000-0000-0000-0000-000000000604',
    '00000000-0000-0000-0000-000000000507',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'Comentario ocultado por moderacion interna.',
    'hidden',
    now() - interval '5 days',
    now() - interval '5 days'
  )
ON CONFLICT (id)
DO UPDATE SET
  body = EXCLUDED.body,
  status = EXCLUDED.status,
  updated_at = EXCLUDED.updated_at;

INSERT INTO app.user_play_engagements (
  id,
  user_id,
  play_id,
  kind,
  created_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000701',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000201',
    'attended',
    now() - interval '19 days'
  ),
  (
    '00000000-0000-0000-0000-000000000702',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000203',
    'wishlist',
    now() - interval '11 days'
  ),
  (
    '00000000-0000-0000-0000-000000000703',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000201',
    'wishlist',
    now() - interval '22 days'
  ),
  (
    '00000000-0000-0000-0000-000000000704',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000201',
    'attended',
    now() - interval '20 days'
  ),
  (
    '00000000-0000-0000-0000-000000000705',
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000202',
    'attended',
    now() - interval '145 days'
  ),
  (
    '00000000-0000-0000-0000-000000000706',
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000204',
    'wishlist',
    now() - interval '7 days'
  ),
  (
    '00000000-0000-0000-0000-000000000707',
    '00000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000203',
    'attended',
    now() - interval '10 days'
  ),
  (
    '00000000-0000-0000-0000-000000000708',
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000204',
    'attended',
    now() - interval '6 days'
  ),
  (
    '00000000-0000-0000-0000-000000000709',
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000201',
    'wishlist',
    now() - interval '9 days'
  )
ON CONFLICT (user_id, play_id, kind)
DO NOTHING;

INSERT INTO app.user_follows (
  follower_user_id,
  following_user_id,
  created_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003',
    now() - interval '60 days'
  ),
  (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000004',
    now() - interval '50 days'
  ),
  (
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    now() - interval '48 days'
  ),
  (
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000002',
    now() - interval '30 days'
  ),
  (
    '00000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000002',
    now() - interval '28 days'
  ),
  (
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000002',
    now() - interval '21 days'
  ),
  (
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000003',
    now() - interval '18 days'
  )
ON CONFLICT (follower_user_id, following_user_id)
DO NOTHING;
