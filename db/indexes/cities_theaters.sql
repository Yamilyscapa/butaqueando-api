CREATE INDEX IF NOT EXISTS idx_cities_is_active_name
  ON app.cities (is_active, name, id);

CREATE INDEX IF NOT EXISTS idx_theaters_city_active_name
  ON app.theaters (city_id, is_active, name, id);
