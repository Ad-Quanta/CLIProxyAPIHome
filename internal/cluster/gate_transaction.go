package cluster

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	gormlogger "gorm.io/gorm/logger"
)

// gateTransactionTrace observes one transaction without changing SQL logging or
// recording parameters. Acquisition includes the database round trip; held time
// starts on receipt of the lock and includes commit or rollback completion.
type gateTransactionTrace struct {
	gormlogger.Interface
	operation         string
	startedAt         time.Time
	lockedAt          time.Time
	gateWait          time.Duration
	rebalanceLockedAt time.Time
	rebalanceWait     time.Duration
	sqlCount          int
	planCount         int
}

// ParamsFilter keeps the application's parameter-redaction boundary when GORM
// sees this observer instead of the underlying parameterized logger.
func (t *gateTransactionTrace) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func (t *gateTransactionTrace) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	t.sqlCount++
	t.Interface.Trace(ctx, begin, fc, err)
}

func (t *gateTransactionTrace) logResult(err error) {
	finishedAt := time.Now()
	total := finishedAt.Sub(t.startedAt)
	var held time.Duration
	if !t.lockedAt.IsZero() {
		held = finishedAt.Sub(t.lockedAt)
	}
	var rebalanceHeld time.Duration
	if !t.rebalanceLockedAt.IsZero() {
		rebalanceHeld = finishedAt.Sub(t.rebalanceLockedAt)
	}
	slowThreshold := defaultDatabaseSlowQueryThreshold
	if logger, ok := t.Interface.(parameterizedGORMLogger); ok {
		if inner, okInner := logger.inner.(homeGORMLogger); okInner {
			slowThreshold = inner.slowThreshold
		}
	}
	// SQL count excludes driver-level transaction control; timings end after commit or rollback.
	const message = "cluster transaction operation=%s total_ms=%.3f gate_acquire_ms=%.3f gate_held_ms=%.3f sql_count=%d plan_count=%d success=%t rebalance_acquire_ms=%.3f rebalance_held_ms=%.3f"
	if total >= slowThreshold {
		log.Warnf(message, t.operation, float64(total)/float64(time.Millisecond), float64(t.gateWait)/float64(time.Millisecond), float64(held)/float64(time.Millisecond), t.sqlCount, t.planCount, err == nil, float64(t.rebalanceWait)/float64(time.Millisecond), float64(rebalanceHeld)/float64(time.Millisecond))
		return
	}
	log.Debugf(message, t.operation, float64(total)/float64(time.Millisecond), float64(t.gateWait)/float64(time.Millisecond), float64(held)/float64(time.Millisecond), t.sqlCount, t.planCount, err == nil, float64(t.rebalanceWait)/float64(time.Millisecond), float64(rebalanceHeld)/float64(time.Millisecond))
}
