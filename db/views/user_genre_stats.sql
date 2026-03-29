CREATE OR REPLACE VIEW app.user_genre_stats AS
SELECT
  e.user_id,
  g.id AS genre_id,
  g.name AS genre_name,
  COUNT(*)::int AS attended_count
FROM app.user_play_engagements e
JOIN app.plays p ON p.id = e.play_id
JOIN app.play_genres pg ON pg.play_id = e.play_id
JOIN app.genres g ON g.id = pg.genre_id
WHERE e.kind = 'attended'
  AND p.curation_status = 'published'
GROUP BY e.user_id, g.id, g.name;
