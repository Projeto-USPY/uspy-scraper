package collector

import (
	"context"
	"testing"

	"github.com/Projeto-USPY/uspy-backend/db"
)

func buildQueryParams(institute string) map[string][]string {
	return map[string][]string{
		"institute": {institute},
	}
}

func BenchmarkCollectJupiter(b *testing.B) {
	ctx := context.Background()
	CollectJupiter(ctx, db.Env{}, map[string][]string{}, Noop)
}

func BenchmarkCollectICMCSubjects(b *testing.B) {
	ctx := context.Background()
	CollectJupiter(ctx, db.Env{}, buildQueryParams("55"), Noop)
}

func BenchmarkCollectICMCOfferings(b *testing.B) {
	ctx := context.Background()
	CollectOfferings(ctx, db.Env{}, buildQueryParams("55"), Noop)
}

func BenchmarkCollectPoliSubjects(b *testing.B) {
	ctx := context.Background()
	CollectJupiter(ctx, db.Env{}, buildQueryParams("3"), Noop)
}

func BenchmarkCollectPoliOfferings(b *testing.B) {
	ctx := context.Background()
	CollectOfferings(ctx, db.Env{}, buildQueryParams("3"), Noop)
}
