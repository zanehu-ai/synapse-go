// Package outbox implements Synapse's Reliable Events Outbox pattern:
// business writes enqueue an outbox row in the same DB transaction; an async
// dispatcher picks up pending rows and delivers them to the message bus.
//
// # Transactional Inserter
//
// The core abstraction is [Inserter]:
//
//	type Inserter interface {
//	    InsertTx(ctx context.Context, tx *gorm.DB, evt any) error
//	}
//
// [RepoInserter] is the production implementation — it calls GORM Create on
// the caller-supplied transaction handle. [MockInserter] is a test helper
// that records events in-memory and supports failure injection.
//
// # Migration path
//
// Before (best-effort, violates CLAUDE.md hard constraint #7):
//
//	// Build the outbox row ...
//	evt, err := eventsSvc.BuildOutbox(reliable_events.PublishInput{...})
//
//	// Commit business write.
//	if err := db.Save(&order).Error; err != nil { return err }
//
//	// Outbox insert is OUTSIDE the business tx — if the process crashes
//	// between here and the insert the event is silently lost forever.
//	if err := repo.InsertOutbox(ctx, evt); err != nil { log.Error(err) }
//
// After (transactional, CLAUDE.md hard constraint #7 satisfied):
//
//	inserter := outbox.NewRepoInserter()
//
//	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
//	    if err := tx.Save(&order).Error; err != nil {
//	        return err
//	    }
//	    evt, err := eventsSvc.BuildOutbox(reliable_events.PublishInput{...})
//	    if err != nil {
//	        return err
//	    }
//	    // Outbox row commits or rolls back together with the business write.
//	    return inserter.InsertTx(ctx, tx, evt)
//	})
//
// # Anti-scope
//
// This package exposes the platform-side interface only. Migration of
// individual funding_* services (funding_payment, funding_withdrawal,
// funding_refund, funding_ledger) is deferred to per-service follow-up PRs
// after their respective in-flight PRs (#381/#384/#385/#386) merge. The
// recommended adoption order is:
//
//  1. funding_payment (canonical, highest-value path)
//  2. funding_withdrawal
//  3. funding_refund
//  4. funding_ledger
//
// # In-process store
//
// [MemoryStore] and the package-level [Enqueue] / [Dispatch] helpers remain
// unchanged for tests and lightweight in-process producers that do not have a
// DB handle. They are not transactional with respect to a GORM session.
package outbox
