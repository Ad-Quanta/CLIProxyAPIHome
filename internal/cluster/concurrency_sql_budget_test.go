package cluster

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestConcurrencySQLBudgets(t *testing.T) {
	for _, backend := range []struct {
		name string
		open func(*testing.T) *Repository
	}{
		{"sqlite", newCredentialFoundationTestRepository},
		{"postgres", newPostgresQuiescenceRepository},
	} {
		t.Run(backend.name, func(t *testing.T) {
			repo := backend.open(t)
			seedConcurrencyAdmissionLifetime(t, repo, "fp-a", "home-a")
			seedConcurrencyAdmissionAuth(t, repo, "cred-1")
			seedConcurrencyAdmissionPolicy(t, repo, "cred-1", int64Pointer(10), map[string]int64{"gpt": 10})
			request := ConcurrencyAdmissionRequest{CredentialID: "cred-1", Model: "gpt", ProtocolVersion: 1, Lifetime: concurrencyAdmissionLifetime(t, repo, "fp-a", "home-a")}
			counter := &lifecycleSQLCounter{Interface: logger.Default.LogMode(logger.Silent)}
			repo.db = repo.db.Session(&gorm.Session{Logger: counter})
			if total, model, errCounts := sumConcurrencyCountsTx(t.Context(), repo.db, "cred-1", "gpt"); errCounts != nil || total != 0 || model != 0 {
				t.Fatalf("sum=(%d,%d,%v)", total, model, errCounts)
			}
			if count := counter.statements.Load(); count != 1 {
				t.Fatalf("aggregate SQL statements=%d, want 1", count)
			}
			for _, phase := range []string{"create counter", "update counter"} {
				counter.statements.Store(0)
				result, errAdmit := repo.AdmitCredentialConcurrency(t.Context(), request)
				if errAdmit != nil || !result.Accounted {
					t.Fatalf("%s admission=(%#v,%v)", phase, result, errAdmit)
				}
				// GORM traces business SQL, not database/sql Begin/Commit calls.
				if count := counter.statements.Load(); count != 8 {
					t.Fatalf("%s SQL statements=%d, want 8", phase, count)
				}
			}
			t.Log("aggregate: 1 SQL statement; limited admission: 8 business SQL statements")
		})
	}
}
