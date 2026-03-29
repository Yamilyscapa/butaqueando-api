package health

import (
	"context"
	"testing"
)

func TestServiceCheckWithNilDB(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	result := svc.Check(context.Background())

	if result.Ready {
		t.Fatalf("expected ready false, got true")
	}

	if result.Database {
		t.Fatalf("expected database false, got true")
	}
}
