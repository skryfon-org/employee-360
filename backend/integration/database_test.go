//go:build integration

package integration

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/testutil/pgtest"
)

func TestConnect_SuccessLiveDatabase(t *testing.T) {
	db := pgtest.Open(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping failed on live database: %v", err)
	}
}
