package httpx

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const CacheControlHeader = "Cache-Control"

type CachePolicy struct {
	Public   bool
	MaxAge   time.Duration
	SMaxAge  time.Duration
	NoStore  bool
	NoCache  bool
	Private  bool
	MustRev  bool
}

func PublicMaxAge(maxAge time.Duration) CachePolicy {
	return CachePolicy{Public: true, MaxAge: maxAge}
}

func PrivateNoStore() CachePolicy {
	return CachePolicy{Private: true, NoStore: true}
}

func (p CachePolicy) Header() string {
	parts := make([]string, 0, 4)
	switch {
	case p.NoStore:
		parts = append(parts, "no-store")
	case p.Public:
		parts = append(parts, "public")
	case p.Private:
		parts = append(parts, "private")
	}

	if p.NoCache {
		parts = append(parts, "no-cache")
	}
	if p.MustRev {
		parts = append(parts, "must-revalidate")
	}
	if p.MaxAge > 0 {
		parts = append(parts, fmt.Sprintf("max-age=%d", int(p.MaxAge.Seconds())))
	}
	if p.SMaxAge > 0 {
		parts = append(parts, fmt.Sprintf("s-maxage=%d", int(p.SMaxAge.Seconds())))
	}

	return joinHeaderParts(parts)
}

func ApplyCachePolicy(c *gin.Context, policy CachePolicy) {
	value := policy.Header()
	if value == "" {
		return
	}
	c.Header(CacheControlHeader, value)
}

func joinHeaderParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += ", " + part
	}
	return out
}
