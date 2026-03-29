DROP TRIGGER IF EXISTS trg_plays_updated_at ON app.plays;
CREATE TRIGGER trg_plays_updated_at
BEFORE UPDATE ON app.plays
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_reviews_updated_at ON app.reviews;
CREATE TRIGGER trg_reviews_updated_at
BEFORE UPDATE ON app.reviews
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_review_comments_updated_at ON app.review_comments;
CREATE TRIGGER trg_review_comments_updated_at
BEFORE UPDATE ON app.review_comments
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON app.user_profiles;
CREATE TRIGGER trg_user_profiles_updated_at
BEFORE UPDATE ON app.user_profiles
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_users_updated_at ON app.users;
CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON app.users
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();

DROP TRIGGER IF EXISTS trg_user_refresh_tokens_updated_at ON app.user_refresh_tokens;
CREATE TRIGGER trg_user_refresh_tokens_updated_at
BEFORE UPDATE ON app.user_refresh_tokens
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();
