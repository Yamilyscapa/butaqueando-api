package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildETagIsStableForEquivalentData(t *testing.T) {
	t.Parallel()

	tagA, err := BuildETag(map[string]any{"title": "Hamlet", "count": 2})
	if err != nil {
		t.Fatalf("build etag A: %v", err)
	}

	tagB, err := BuildETag(map[string]any{"count": 2, "title": "Hamlet"})
	if err != nil {
		t.Fatalf("build etag B: %v", err)
	}

	if tagA != tagB {
		t.Fatalf("expected stable etag, got %q and %q", tagA, tagB)
	}
}

func TestETagMatchesSupportsWeakTagsAndWildcard(t *testing.T) {
	t.Parallel()

	etag, err := BuildETag(map[string]string{"x": "1"})
	if err != nil {
		t.Fatalf("build etag: %v", err)
	}

	if !ETagMatches("W/"+etag, etag) {
		t.Fatalf("expected weak tag to match")
	}

	if !ETagMatches("*", etag) {
		t.Fatalf("expected wildcard to match")
	}

	if ETagMatches("\"different\"", etag) {
		t.Fatalf("expected mismatched tag to fail")
	}
}

func TestWriteDataWithETagReturnsNotModifiedWhenMatch(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		WriteDataWithETag(c, http.StatusOK, map[string]any{"id": "1", "title": "Hamlet"})
	})

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(first, firstReq)

	if first.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, first.Code)
	}

	etag := first.Header().Get(ETagHeader)
	if etag == "" {
		t.Fatalf("expected etag header")
	}

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	secondReq.Header.Set(IfNoneMatchHeader, etag)
	router.ServeHTTP(second, secondReq)

	if second.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, second.Code)
	}

	if second.Body.Len() != 0 {
		t.Fatalf("expected empty body on not modified")
	}
}

func TestWriteDataWithETagReturnsBodyWhenNoMatch(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		WriteDataWithETag(c, http.StatusOK, map[string]any{"id": "1"})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(IfNoneMatchHeader, "\"other\"")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if response.Error != nil {
		t.Fatalf("expected no error payload")
	}
}
