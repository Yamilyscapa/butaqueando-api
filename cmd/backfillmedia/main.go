// Command backfillmedia generates variant + blurhash metadata for existing media rows.
//
// Idempotent: a row whose `variants` column already contains entries is skipped.
// On each unprocessed row it:
//   1. Downloads the canonical (legacy) object from S3.
//   2. Runs imageproc.OptimizeImageVariants with the configured plan.
//   3. Uploads each variant (overwriting the legacy key for the largest size).
//   4. Generates a blurhash from the source bytes.
//   5. Updates the row with variants jsonb + blurhash.
//
// Usage:
//   DATABASE_URL=... PLAYS_S3_*=... USERS_S3_*=... go run ./cmd/backfillmedia
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/butaqueando/api/internal/config"
	"github.com/butaqueando/api/internal/database"
	"github.com/butaqueando/api/internal/shared/imageproc"
	"github.com/butaqueando/api/internal/shared/storage"
	"gorm.io/gorm"
)

type variantRecord struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ObjectKey string `json:"objectKey"`
	SizeBytes int64  `json:"sizeBytes"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, sqlDB, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()

	playsClient, err := storage.NewS3Client(ctx, storage.BucketConfig{
		Endpoint:        cfg.PlaysS3.Endpoint,
		Region:          cfg.PlaysS3.Region,
		AccessKeyID:     cfg.PlaysS3.AccessKeyID,
		SecretAccessKey: cfg.PlaysS3.SecretAccessKey,
		Bucket:          cfg.PlaysS3.Bucket,
	})
	if err != nil {
		log.Fatalf("plays storage client: %v", err)
	}

	usersClient, err := storage.NewS3Client(ctx, storage.BucketConfig{
		Endpoint:        cfg.UsersS3.Endpoint,
		Region:          cfg.UsersS3.Region,
		AccessKeyID:     cfg.UsersS3.AccessKeyID,
		SecretAccessKey: cfg.UsersS3.SecretAccessKey,
		Bucket:          cfg.UsersS3.Bucket,
	})
	if err != nil {
		log.Fatalf("users storage client: %v", err)
	}

	playPlan := buildVariantPlan(cfg.ImageVariantWidthsPlays, cfg.ImageWebPQuality, true)
	avatarPlan := buildVariantPlan(cfg.ImageVariantWidthsAvatars, cfg.ImageWebPQuality, false)

	if err := backfillPlayMedia(ctx, db, playsClient, playPlan, cfg.ImageBlurhashEnabled); err != nil {
		log.Fatalf("backfill play_media: %v", err)
	}
	if err := backfillSuggestionMedia(ctx, db, playsClient, playPlan, cfg.ImageBlurhashEnabled); err != nil {
		log.Fatalf("backfill play_edit_suggestion_media: %v", err)
	}
	if err := backfillAvatars(ctx, db, usersClient, avatarPlan, cfg.ImageBlurhashEnabled); err != nil {
		log.Fatalf("backfill avatars: %v", err)
	}

	log.Printf("backfillmedia complete")
}

func buildVariantPlan(widths []int, quality int, portrait bool) []imageproc.VariantSpec {
	plan := make([]imageproc.VariantSpec, 0, len(widths))
	sorted := append([]int(nil), widths...)
	sortInts(sorted)
	for _, w := range sorted {
		if w <= 0 {
			continue
		}
		h := w
		if portrait {
			// match service-side 9:16 envelope
			h = (w * 1920) / 1080
		}
		plan = append(plan, imageproc.VariantSpec{
			MaxWidth:          w,
			MaxHeight:         h,
			Quality:           quality,
			TargetContentType: "image/webp",
		})
	}
	return plan
}

func sortInts(values []int) {
	for i := 1; i < len(values); i++ {
		j := i
		for j > 0 && values[j-1] > values[j] {
			values[j-1], values[j] = values[j], values[j-1]
			j--
		}
	}
}

func backfillPlayMedia(ctx context.Context, db *gorm.DB, client storage.Client, plan []imageproc.VariantSpec, blurhashEnabled bool) error {
	type row struct {
		ID        string `gorm:"column:id"`
		ObjectKey string `gorm:"column:object_key"`
	}
	var rows []row
	err := db.WithContext(ctx).
		Raw(`SELECT id::text AS id, object_key FROM app.play_media WHERE COALESCE(jsonb_array_length(variants), 0) = 0`).
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("query play_media: %w", err)
	}
	log.Printf("play_media rows to backfill: %d", len(rows))
	for _, r := range rows {
		if err := processRow(ctx, client, plan, blurhashEnabled, r.ObjectKey, func(variantsJSON []byte, blurhash *string) error {
			return db.WithContext(ctx).
				Exec(`UPDATE app.play_media SET variants = ?, blurhash = ? WHERE id = ?::uuid`, variantsJSON, blurhash, r.ID).
				Error
		}); err != nil {
			log.Printf("play_media %s: %v", r.ID, err)
			continue
		}
		log.Printf("play_media backfilled: id=%s object_key=%s", r.ID, r.ObjectKey)
	}
	return nil
}

func backfillSuggestionMedia(ctx context.Context, db *gorm.DB, client storage.Client, plan []imageproc.VariantSpec, blurhashEnabled bool) error {
	type row struct {
		ID        string `gorm:"column:id"`
		ObjectKey string `gorm:"column:object_key"`
	}
	var rows []row
	err := db.WithContext(ctx).
		Raw(`SELECT id::text AS id, object_key FROM app.play_edit_suggestion_media WHERE COALESCE(jsonb_array_length(variants), 0) = 0`).
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("query suggestion media: %w", err)
	}
	log.Printf("play_edit_suggestion_media rows to backfill: %d", len(rows))
	for _, r := range rows {
		if err := processRow(ctx, client, plan, blurhashEnabled, r.ObjectKey, func(variantsJSON []byte, blurhash *string) error {
			return db.WithContext(ctx).
				Exec(`UPDATE app.play_edit_suggestion_media SET variants = ?, blurhash = ? WHERE id = ?::uuid`, variantsJSON, blurhash, r.ID).
				Error
		}); err != nil {
			log.Printf("suggestion_media %s: %v", r.ID, err)
			continue
		}
		log.Printf("suggestion_media backfilled: id=%s object_key=%s", r.ID, r.ObjectKey)
	}
	return nil
}

func backfillAvatars(ctx context.Context, db *gorm.DB, client storage.Client, plan []imageproc.VariantSpec, blurhashEnabled bool) error {
	type row struct {
		UserID    string `gorm:"column:user_id"`
		ObjectKey string `gorm:"column:avatar_object_key"`
	}
	var rows []row
	err := db.WithContext(ctx).
		Raw(`SELECT user_id::text AS user_id, avatar_object_key FROM app.user_profiles WHERE avatar_object_key IS NOT NULL AND COALESCE(jsonb_array_length(avatar_variants), 0) = 0`).
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("query user_profiles: %w", err)
	}
	log.Printf("user_profiles rows to backfill: %d", len(rows))
	for _, r := range rows {
		if err := processRow(ctx, client, plan, blurhashEnabled, r.ObjectKey, func(variantsJSON []byte, blurhash *string) error {
			return db.WithContext(ctx).
				Exec(`UPDATE app.user_profiles SET avatar_variants = ?, avatar_blurhash = ? WHERE user_id = ?::uuid`, variantsJSON, blurhash, r.UserID).
				Error
		}); err != nil {
			log.Printf("user_profile %s: %v", r.UserID, err)
			continue
		}
		log.Printf("user_profile avatar backfilled: user_id=%s object_key=%s", r.UserID, r.ObjectKey)
	}
	return nil
}

func processRow(ctx context.Context, client storage.Client, plan []imageproc.VariantSpec, blurhashEnabled bool, primaryObjectKey string, persist func([]byte, *string) error) error {
	src, err := client.GetObject(ctx, storage.GetObjectInput{ObjectKey: primaryObjectKey})
	if err != nil {
		return fmt.Errorf("download %s: %w", primaryObjectKey, err)
	}

	results, err := imageproc.OptimizeImageVariants(src, plan)
	if err != nil {
		return fmt.Errorf("optimize: %w", err)
	}

	primaryWidth := plan[len(plan)-1].MaxWidth
	variants := make([]variantRecord, 0, len(results))

	for index, res := range results {
		spec := plan[index]
		variantKey := variantObjectKey(primaryObjectKey, spec.MaxWidth, primaryWidth)
		if err := client.PutObject(ctx, storage.PutObjectInput{
			ObjectKey:    variantKey,
			Content:      res.Content,
			ContentType:  res.ContentType,
			CacheControl: "public, max-age=31536000, immutable",
		}); err != nil {
			return fmt.Errorf("upload %s: %w", variantKey, err)
		}
		variants = append(variants, variantRecord{
			Width:     res.Width,
			Height:    res.Height,
			ObjectKey: variantKey,
			SizeBytes: int64(len(res.Content)),
		})
	}

	variantsJSON, err := json.Marshal(variants)
	if err != nil {
		return fmt.Errorf("encode variants: %w", err)
	}

	var blurhash *string
	if blurhashEnabled {
		if hash, hashErr := imageproc.GenerateBlurhashFromBytes(src); hashErr == nil && hash != "" {
			h := hash
			blurhash = &h
		}
	}

	if err := persist(variantsJSON, blurhash); err != nil {
		return fmt.Errorf("persist: %w", err)
	}
	return nil
}

func variantObjectKey(originalKey string, width int, primaryWidth int) string {
	base := normalizeWebPKey(originalKey)
	if width == primaryWidth {
		return base
	}
	ext := path.Ext(base)
	if ext == "" {
		return fmt.Sprintf("%s-%d", base, width)
	}
	return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(base, ext), width, ext)
}

func normalizeWebPKey(objectKey string) string {
	trimmed := strings.TrimSpace(objectKey)
	if trimmed == "" {
		return ""
	}
	lastSlash := strings.LastIndex(trimmed, "/")
	lastDot := strings.LastIndex(trimmed, ".")
	if lastDot > lastSlash {
		return trimmed[:lastDot] + ".webp"
	}
	return trimmed + ".webp"
}
