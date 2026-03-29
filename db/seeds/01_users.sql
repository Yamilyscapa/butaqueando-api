INSERT INTO app.users (
  id,
  display_name,
  email,
  password_hash,
  role,
  email_verified_at,
  created_at,
  updated_at
)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'Butaqueando Admin',
    'admin@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'admin',
    now() - interval '120 days',
    now() - interval '120 days',
    now() - interval '120 days'
  ),
  (
    '00000000-0000-0000-0000-000000000002',
    'Ana Torres',
    'ana@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'user',
    now() - interval '90 days',
    now() - interval '90 days',
    now() - interval '90 days'
  ),
  (
    '00000000-0000-0000-0000-000000000003',
    'Marco Rios',
    'marco@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'user',
    now() - interval '85 days',
    now() - interval '85 days',
    now() - interval '85 days'
  ),
  (
    '00000000-0000-0000-0000-000000000004',
    'Luna Perez',
    'luna@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'user',
    now() - interval '70 days',
    now() - interval '70 days',
    now() - interval '70 days'
  ),
  (
    '00000000-0000-0000-0000-000000000005',
    'Diego Suarez',
    'diego@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'user',
    now() - interval '65 days',
    now() - interval '65 days',
    now() - interval '65 days'
  ),
  (
    '00000000-0000-0000-0000-000000000006',
    'Carla Mendez',
    'carla@butaqueando.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'user',
    now() - interval '55 days',
    now() - interval '55 days',
    now() - interval '55 days'
  )
ON CONFLICT (id)
DO UPDATE SET
  display_name = EXCLUDED.display_name,
  email = EXCLUDED.email,
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  email_verified_at = EXCLUDED.email_verified_at,
  updated_at = EXCLUDED.updated_at;
