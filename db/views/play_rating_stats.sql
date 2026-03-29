CREATE OR REPLACE VIEW app.play_rating_stats AS
SELECT
  r.play_id,
  COUNT(*)::int AS review_count,
  ROUND(AVG(r.rating)::numeric, 2) AS avg_rating
FROM app.reviews r
JOIN app.plays p ON p.id = r.play_id
WHERE r.status = 'published'
  AND p.curation_status = 'published'
GROUP BY r.play_id;
