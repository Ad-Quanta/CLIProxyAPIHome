package cluster

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func TestGateTransactionTracePreservesRedactionAndThreshold(t *testing.T) {
	repo := newCredentialFoundationTestRepository(t)
	var sqlOutput bytes.Buffer
	baseLogger := log.New()
	baseLogger.SetOutput(&sqlOutput)
	trace := gateTransactionTrace{
		Interface: newParameterizedGORMLogger(newHomeGORMLogger(baseLogger, 2*time.Second)),
		operation: "test", startedAt: time.Now(),
	}
	db := repo.db.Session(&gorm.Session{Logger: &trace})
	if _, ok := db.Logger.(gorm.ParamsFilter); !ok {
		t.Fatal("transaction observer lost SQL parameter filtering")
	}
	secret := "test-secret-must-not-appear"
	if errExec := db.Exec("SELECT * FROM missing_trace_table WHERE secret = ?", secret).Error; errExec == nil {
		t.Fatal("invalid query unexpectedly succeeded")
	}
	if trace.sqlCount != 1 || strings.Contains(sqlOutput.String(), secret) || !strings.Contains(sqlOutput.String(), "secret = ?") {
		t.Fatalf("observer lost trace counts or redaction: count=%d logs=%s", trace.sqlCount, sqlOutput.String())
	}
	var summary bytes.Buffer
	standard := log.StandardLogger()
	oldOutput, oldLevel := standard.Out, standard.GetLevel()
	log.SetOutput(&summary)
	log.SetLevel(log.InfoLevel)
	t.Cleanup(func() { log.SetOutput(oldOutput); log.SetLevel(oldLevel) })
	trace.startedAt = time.Now().Add(-time.Second)
	trace.logResult(nil)
	if summary.Len() != 0 {
		t.Fatal("transaction summary ignored the configured two-second threshold")
	}
	trace.startedAt = time.Now().Add(-3 * time.Second)
	trace.lockedAt = time.Now().Add(-time.Second)
	trace.gateWait, trace.planCount = 20*time.Millisecond, 2
	trace.logResult(errors.New(secret))
	for _, expected := range []string{"operation=test", "total_ms=", "gate_acquire_ms=20.000", "gate_held_ms=", "sql_count=1", "plan_count=2", "success=false"} {
		if !strings.Contains(summary.String(), expected) {
			t.Fatalf("missing transaction diagnostic %q: %s", expected, summary.String())
		}
	}
	if strings.Contains(summary.String(), secret) {
		t.Fatal("transaction summary exposed an error's sensitive contents")
	}
}
