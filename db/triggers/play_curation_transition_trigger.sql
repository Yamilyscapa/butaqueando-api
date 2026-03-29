DROP TRIGGER IF EXISTS trg_plays_curation_transition ON app.plays;
CREATE TRIGGER trg_plays_curation_transition
BEFORE UPDATE ON app.plays
FOR EACH ROW
EXECUTE FUNCTION app.enforce_play_curation_transition();
