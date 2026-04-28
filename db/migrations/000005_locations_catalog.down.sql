DROP TRIGGER IF EXISTS trg_theaters_updated_at ON app.theaters;
DROP TRIGGER IF EXISTS trg_cities_updated_at ON app.cities;

DROP INDEX IF EXISTS app.idx_theaters_city_active_name;
DROP INDEX IF EXISTS app.idx_cities_is_active_name;

DROP TABLE IF EXISTS app.theaters;
DROP TABLE IF EXISTS app.cities;
