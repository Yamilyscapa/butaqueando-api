package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ETagHeader        = "ETag"
	IfNoneMatchHeader = "If-None-Match"
)

func BuildETag(data any) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(encoded)
	return "\"" + hex.EncodeToString(sum[:]) + "\"", nil
}

func ETagMatches(ifNoneMatch string, etag string) bool {
	current := normalizeETagToken(etag)
	if current == "" {
		return false
	}

	for _, part := range strings.Split(ifNoneMatch, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == "*" {
			return true
		}

		if normalizeETagToken(candidate) == current {
			return true
		}
	}

	return false
}

func WriteDataWithETag(c *gin.Context, status int, data any) {
	etag, err := BuildETag(data)
	if err != nil {
		WriteData(c, status, data)
		return
	}

	c.Header(ETagHeader, etag)
	if ETagMatches(c.GetHeader(IfNoneMatchHeader), etag) {
		c.Status(http.StatusNotModified)
		return
	}

	WriteData(c, status, data)
}

func normalizeETagToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToUpper(trimmed), "W/") {
		trimmed = strings.TrimSpace(trimmed[2:])
	}

	trimmed = strings.Trim(trimmed, "\"")
	return strings.TrimSpace(trimmed)
}
