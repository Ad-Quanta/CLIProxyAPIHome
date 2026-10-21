package cluster

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPIHome/internal/node"
	"gorm.io/gorm"
)

type snapshotReplacementSourceContextKey struct{}

func TestPostgresCoordinatorSnapshotReplacementSerializesHeartbeatAndClientCount(t *testing.T) {
	for _, firstSource := range []string{"heartbeat", "client-count"} {
		t.Run(firstSource, func(t *testing.T) {
			repo := newPostgresQuiescenceRepository(t)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			t.Cleanup(cancel)
			coordinator := NewCoordinator(repo, NodeIdentity{IP: "snapshot-home", Port: 8327}, CoordinatorOptions{})
			if errInitialize := coordinator.Initialize(ctx); errInitialize != nil {
				t.Fatal(errInitialize)
			}
			t.Cleanup(func() { coordinator.setMaster(false) })
			home, initialized := coordinator.HomeIncarnation()
			if !initialized {
				t.Fatal("coordinator was not initialized")
			}
			registry := node.GlobalRegistry()
			const oldIP, newIP = "192.0.2.211", "192.0.2.212"
			const oldID, newID = "snapshot-replacement-old", "snapshot-replacement-new"
			registry.AddWithNodeID(oldIP, oldID, home.StartedAt)
			t.Cleanup(func() {
				registry.RemoveWithNodeID(oldIP, oldID)
				registry.RemoveWithNodeID(newIP, newID)
			})

			deleted := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			finish := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(finish)
			secondCountUpdated := make(chan struct{})
			var countUpdatedOnce sync.Once
			var snapshotDeletes atomic.Int64
			if errRegister := repo.db.Callback().Delete().After("gorm:delete").Register("test:hold-node-snapshot-replacement", func(tx *gorm.DB) {
				if tx.Statement.Table != (CPANodeRecord{}).TableName() || tx.DryRun {
					return
				}
				snapshotDeletes.Add(1)
				if tx.Statement.Context.Value(snapshotReplacementSourceContextKey{}) != "first" {
					return
				}
				close(deleted)
				select {
				case <-release:
				case <-ctx.Done():
				}
			}); errRegister != nil {
				t.Fatal(errRegister)
			}
			if errRegister := repo.db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("test:observe-node-count-update", func(tx *gorm.DB) {
				if tx.Statement.Table == (ClusterNodeRecord{}).TableName() && tx.Statement.Context.Value(snapshotReplacementSourceContextKey{}) == "second" && tx.Error == nil {
					countUpdatedOnce.Do(func() { close(secondCountUpdated) })
				}
			}); errRegister != nil {
				t.Fatal(errRegister)
			}
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			firstCtx := context.WithValue(ctx, snapshotReplacementSourceContextKey{}, "first")
			secondCtx := context.WithValue(ctx, snapshotReplacementSourceContextKey{}, "second")
			go func() {
				if firstSource == "heartbeat" {
					firstDone <- coordinator.heartbeatAndElect(firstCtx, home)
				} else {
					firstDone <- coordinator.UpdateClientCount(firstCtx, registry.TotalCount())
				}
			}()
			select {
			case <-deleted:
			case <-ctx.Done():
				t.Fatal("first replacement did not reach DELETE:", ctx.Err())
			}
			registry.RemoveWithNodeID(oldIP, oldID)
			registry.AddWithNodeID(newIP, newID, home.StartedAt)
			go func() {
				if firstSource == "heartbeat" {
					secondDone <- coordinator.UpdateClientCount(secondCtx, registry.TotalCount())
				} else {
					secondDone <- coordinator.heartbeatAndElect(secondCtx, home)
				}
			}()
			select {
			case <-secondCountUpdated:
			case <-ctx.Done():
				t.Fatal("second caller did not update the cluster count:", ctx.Err())
			}
			// Both callers share this Repository. Its independent mutex must span DELETE through CREATE.
			if repo.cpaSnapshotMu.TryLock() {
				repo.cpaSnapshotMu.Unlock()
				t.Fatal("node snapshot replacement did not retain its independent write lock")
			}
			if got := snapshotDeletes.Load(); got != 1 {
				t.Fatalf("overlapping snapshot DELETEs = %d, want only the held writer", got)
			}
			finish()
			for _, done := range []<-chan error{firstDone, secondDone} {
				select {
				case errReplace := <-done:
					if errReplace != nil {
						t.Fatal(errReplace)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			var records []CPANodeRecord
			if errFind := repo.db.Where("home_ip = ? AND home_port = ? AND home_started_at = ? AND node_id IN ?", home.IP, home.Port, home.StartedAt, []string{oldID, newID}).Find(&records).Error; errFind != nil {
				t.Fatal(errFind)
			}
			if len(records) != 1 || records[0].NodeID != newID {
				t.Fatalf("replacement merged or lost snapshots: %#v", records)
			}
		})
	}
}
