package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/joho/godotenv"
)

const (
	unsplashAPIBase      = "https://api.unsplash.com"
	unsplashAccessKeyEnv = "UNSPLASH_ACCESS_KEY"
)

type seedAsset struct {
	ObjectKey string
	PhotoID   string
	Width     int
	Height    int
	Crop      string
}

type unsplashPhoto struct {
	ID   string `json:"id"`
	URLs struct {
		Raw string `json:"raw"`
	} `json:"urls"`
	Links struct {
		DownloadLocation string `json:"download_location"`
	} `json:"links"`
	User struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Links    struct {
			HTML string `json:"html"`
		} `json:"links"`
	} `json:"user"`
}

var playSeedAssets = []seedAsset{
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000201/poster.jpg", PhotoID: "VPjGmAzQb2I", Width: 1200, Height: 1800, Crop: "entropy"},
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000201/scene-01.jpg", PhotoID: "WW1jsInXgwM", Width: 1600, Height: 900, Crop: "entropy"},
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000202/poster.jpg", PhotoID: "IiCVB_YrJWY", Width: 1200, Height: 1800, Crop: "entropy"},
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000203/poster.jpg", PhotoID: "dIGYalC67v4", Width: 1200, Height: 1800, Crop: "entropy"},
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000204/poster.jpg", PhotoID: "30woXsCEX3c", Width: 1200, Height: 1800, Crop: "entropy"},
	{ObjectKey: "plays/00000000-0000-0000-0000-000000000205/poster.jpg", PhotoID: "l0SiVK5WBH0", Width: 1200, Height: 1800, Crop: "entropy"},
}

var userSeedAssets = []seedAsset{
	{ObjectKey: "users/00000000-0000-0000-0000-000000000002/avatar/mock-avatar.jpg", PhotoID: "D41bRxx-a48", Width: 512, Height: 512, Crop: "faces,entropy"},
	{ObjectKey: "users/00000000-0000-0000-0000-000000000003/avatar/mock-avatar.jpg", PhotoID: "a19OVaa2rzA", Width: 512, Height: 512, Crop: "faces,entropy"},
	{ObjectKey: "users/00000000-0000-0000-0000-000000000004/avatar/mock-avatar.jpg", PhotoID: "kRPEkPXyexw", Width: 512, Height: 512, Crop: "faces,entropy"},
	{ObjectKey: "users/00000000-0000-0000-0000-000000000005/avatar/mock-avatar.jpg", PhotoID: "oB-sJdwpsb0", Width: 512, Height: 512, Crop: "faces,entropy"},
	{ObjectKey: "users/00000000-0000-0000-0000-000000000006/avatar/mock-avatar.jpg", PhotoID: "tyOpmKhLeiA", Width: 512, Height: 512, Crop: "faces,entropy"},
}

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 45 * time.Second}

	unsplashAccessKey := strings.TrimSpace(os.Getenv(unsplashAccessKeyEnv))
	if unsplashAccessKey == "" {
		log.Printf("skip mock media seed: %s is not set", unsplashAccessKeyEnv)
		return
	}

	playsCfg := bucketConfig("PLAYS")
	usersCfg := bucketConfig("USERS")

	if !isCompleteBucketConfig(playsCfg) || !isCompleteBucketConfig(usersCfg) {
		log.Printf("skip mock media seed: missing one or more bucket environment variables")
		return
	}

	playsClient, err := storage.NewS3Client(ctx, playsCfg)
	if err != nil {
		log.Fatalf("build plays storage client: %v", err)
	}

	usersClient, err := storage.NewS3Client(ctx, usersCfg)
	if err != nil {
		log.Fatalf("build users storage client: %v", err)
	}

	if err := seedBucketFromUnsplash(ctx, httpClient, playsClient, "plays", playSeedAssets, unsplashAccessKey); err != nil {
		log.Fatalf("seed plays media from unsplash: %v", err)
	}

	if err := seedBucketFromUnsplash(ctx, httpClient, usersClient, "users", userSeedAssets, unsplashAccessKey); err != nil {
		log.Fatalf("seed users media from unsplash: %v", err)
	}

	log.Printf("mock media seed complete using Unsplash assets")
}

func bucketConfig(prefix string) storage.BucketConfig {
	p := strings.TrimSpace(prefix)

	return storage.BucketConfig{
		Endpoint:        strings.TrimSpace(os.Getenv(p + "_S3_ENDPOINT")),
		Region:          strings.TrimSpace(os.Getenv(p + "_S3_REGION")),
		AccessKeyID:     strings.TrimSpace(os.Getenv(p + "_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv(p + "_S3_SECRET_ACCESS_KEY")),
		Bucket:          strings.TrimSpace(os.Getenv(p + "_S3_BUCKET")),
	}
}

func isCompleteBucketConfig(cfg storage.BucketConfig) bool {
	return strings.TrimSpace(cfg.Endpoint) != "" &&
		strings.TrimSpace(cfg.Region) != "" &&
		strings.TrimSpace(cfg.AccessKeyID) != "" &&
		strings.TrimSpace(cfg.SecretAccessKey) != "" &&
		strings.TrimSpace(cfg.Bucket) != ""
}

func seedBucketFromUnsplash(ctx context.Context, httpClient *http.Client, storageClient storage.Client, bucketName string, assets []seedAsset, accessKey string) error {
	if httpClient == nil {
		return fmt.Errorf("http client is required")
	}

	photoCache := make(map[string]unsplashPhoto, len(assets))

	for _, asset := range assets {
		photo, ok := photoCache[asset.PhotoID]
		if !ok {
			fetched, err := fetchUnsplashPhoto(ctx, httpClient, accessKey, asset.PhotoID)
			if err != nil {
				return fmt.Errorf("fetch unsplash photo %q: %w", asset.PhotoID, err)
			}

			photo = fetched
			photoCache[asset.PhotoID] = photo
		}

		imageURL, err := buildTransformedImageURL(photo.URLs.Raw, asset.Width, asset.Height, asset.Crop)
		if err != nil {
			return fmt.Errorf("build transformed image URL for %q: %w", asset.ObjectKey, err)
		}

		content, contentType, err := downloadImage(ctx, httpClient, imageURL)
		if err != nil {
			return fmt.Errorf("download image for %q: %w", asset.ObjectKey, err)
		}

		err = storageClient.PutObject(ctx, storage.PutObjectInput{
			ObjectKey:    asset.ObjectKey,
			Content:      content,
			ContentType:  contentType,
			CacheControl: "public, max-age=31536000, immutable",
		})
		if err != nil {
			return fmt.Errorf("upload %q to %s bucket: %w", asset.ObjectKey, bucketName, err)
		}

		if err := trackUnsplashDownload(ctx, httpClient, accessKey, photo.Links.DownloadLocation); err != nil {
			log.Printf("warn: failed to track unsplash download for photo=%s: %v", photo.ID, err)
		}

		photoOwner := strings.TrimSpace(photo.User.Name)
		if photoOwner == "" {
			photoOwner = "unknown"
		}

		photoProfile := strings.TrimSpace(photo.User.Links.HTML)
		if photoProfile == "" && strings.TrimSpace(photo.User.Username) != "" {
			photoProfile = "https://unsplash.com/@" + strings.TrimSpace(photo.User.Username)
		}

		log.Printf("uploaded media: bucket=%s objectKey=%s unsplashPhoto=%s photographer=%s profile=%s", bucketName, asset.ObjectKey, photo.ID, photoOwner, photoProfile)
	}

	return nil
}

func fetchUnsplashPhoto(ctx context.Context, httpClient *http.Client, accessKey string, photoID string) (unsplashPhoto, error) {
	if strings.TrimSpace(photoID) == "" {
		return unsplashPhoto{}, fmt.Errorf("photo id is required")
	}

	endpoint := fmt.Sprintf("%s/photos/%s?content_filter=high", unsplashAPIBase, url.PathEscape(strings.TrimSpace(photoID)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return unsplashPhoto{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Client-ID "+strings.TrimSpace(accessKey))
	req.Header.Set("Accept-Version", "v1")
	req.Header.Set("User-Agent", "butaqueando-seedmedia/1.0")

	res, err := httpClient.Do(req)
	if err != nil {
		return unsplashPhoto{}, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return unsplashPhoto{}, fmt.Errorf("status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var photo unsplashPhoto
	if err := json.NewDecoder(res.Body).Decode(&photo); err != nil {
		return unsplashPhoto{}, fmt.Errorf("decode response: %w", err)
	}

	if strings.TrimSpace(photo.ID) == "" || strings.TrimSpace(photo.URLs.Raw) == "" {
		return unsplashPhoto{}, fmt.Errorf("unsplash photo response is missing required fields")
	}

	if strings.TrimSpace(photo.Links.DownloadLocation) == "" {
		return unsplashPhoto{}, fmt.Errorf("unsplash photo response missing download_location")
	}

	return photo, nil
}

func buildTransformedImageURL(rawURL string, width int, height int, crop string) (string, error) {
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("width and height must be greater than zero")
	}

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse raw URL: %w", err)
	}

	query := parsed.Query()
	query.Set("w", strconv.Itoa(width))
	query.Set("h", strconv.Itoa(height))
	query.Set("fit", "crop")

	trimmedCrop := strings.TrimSpace(crop)
	if trimmedCrop == "" {
		trimmedCrop = "entropy"
	}
	query.Set("crop", trimmedCrop)
	query.Set("fm", "jpg")
	query.Set("q", "82")
	query.Set("auto", "compress")

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func downloadImage(ctx context.Context, httpClient *http.Client, imageURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "image/*")
	req.Header.Set("User-Agent", "butaqueando-seedmedia/1.0")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, "", fmt.Errorf("status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, "", fmt.Errorf("downloaded image is empty")
	}

	contentType := strings.TrimSpace(res.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "image/jpeg"
	} else if strings.Contains(contentType, ";") {
		contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	}

	return body, contentType, nil
}

func trackUnsplashDownload(ctx context.Context, httpClient *http.Client, accessKey string, downloadLocation string) error {
	endpoint := strings.TrimSpace(downloadLocation)
	if endpoint == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Client-ID "+strings.TrimSpace(accessKey))
	req.Header.Set("Accept-Version", "v1")
	req.Header.Set("User-Agent", "butaqueando-seedmedia/1.0")

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
