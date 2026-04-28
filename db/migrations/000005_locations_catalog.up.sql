CREATE TABLE IF NOT EXISTS app.cities (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT cities_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS app.theaters (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  city_id uuid NOT NULL REFERENCES app.cities(id) ON DELETE RESTRICT,
  name text NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT theaters_city_name_unique UNIQUE (city_id, name)
);

CREATE INDEX IF NOT EXISTS idx_cities_is_active_name
  ON app.cities (is_active, name, id);

CREATE INDEX IF NOT EXISTS idx_theaters_city_active_name
  ON app.theaters (city_id, is_active, name, id);

DROP TRIGGER IF EXISTS trg_cities_updated_at ON app.cities;
CREATE TRIGGER trg_cities_updated_at
BEFORE UPDATE ON app.cities
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_theaters_updated_at ON app.theaters;
CREATE TRIGGER trg_theaters_updated_at
BEFORE UPDATE ON app.theaters
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

INSERT INTO app.cities (name)
SELECT DISTINCT btrim(p.city)
FROM app.plays AS p
WHERE p.city IS NOT NULL AND btrim(p.city) <> ''
ON CONFLICT (name) DO NOTHING;

INSERT INTO app.theaters (city_id, name)
SELECT c.id, btrim(p.theater_name)
FROM app.plays AS p
JOIN app.cities AS c ON c.name = btrim(p.city)
WHERE p.city IS NOT NULL AND btrim(p.city) <> '' AND btrim(p.theater_name) <> ''
ON CONFLICT (city_id, name) DO NOTHING;
