INSERT INTO app.cities (name)
SELECT DISTINCT btrim(p.city)
FROM app.plays AS p
WHERE p.city IS NOT NULL
  AND btrim(p.city) <> ''
ON CONFLICT (name) DO NOTHING;

INSERT INTO app.theaters (city_id, name)
SELECT c.id, btrim(p.theater_name)
FROM app.plays AS p
JOIN app.cities AS c ON c.name = btrim(p.city)
WHERE p.city IS NOT NULL
  AND btrim(p.city) <> ''
  AND btrim(p.theater_name) <> ''
ON CONFLICT (city_id, name) DO NOTHING;
