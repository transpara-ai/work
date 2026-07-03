package work

// White-box unit tests for foldCache.promote() — the no-promotion-over-newer
// guarantee (CFADA1-1/adv1). The black-box tests in store_fold_cache_test.go
// cannot deterministically force the adversarial interleaving (an OLDER-head
// flight finishing AFTER a NEWER stable generation was promoted) because
// ListSummariesCached always reads a fresh headBefore per call; here the
// cache is constructed directly and promote() is called with hand-built
// UUIDv7 heads whose TimestampMS ordering is fully controlled.

import (
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// Hand-built UUIDv7 event IDs with controlled 48-bit millisecond timestamps
// (the first 12 hex digits). olderHead < newerHead by TimestampMS;
// newerHeadTie carries the SAME TimestampMS as newerHead but a different
// value, exercising promote()'s tie guard.
var (
	olderHeadID    = types.MustEventID("00000000-0001-7000-8000-0000000000aa")
	newerHeadID    = types.MustEventID("00000000-0002-7000-8000-0000000000bb")
	newerHeadTieID = types.MustEventID("00000000-0002-7000-8000-0000000000cc")
)

// stateWithHead builds a minimal fold state memoized under the given head.
func stateWithHead(head types.EventID) *taskFoldState {
	s := newTaskFoldState()
	s.stableHead = head
	s.headSet = true
	return s
}

// TestPromote_OlderHeadNeverOverwritesNewerHeldGeneration is the adversarial
// direction: the cache already holds a NEWER stable generation; a slow
// flight that folded against an OLDER head (smaller UUIDv7 TimestampMS)
// finishes late and calls promote(). The newer held state must NOT be
// overwritten.
func TestPromote_OlderHeadNeverOverwritesNewerHeldGeneration(t *testing.T) {
	fc := newFoldCache()

	newer := stateWithHead(newerHeadID)
	fc.promote(newer)
	if fc.state != newer {
		t.Fatalf("setup: newer state was not installed")
	}
	heldGen := fc.state.generation

	older := stateWithHead(olderHeadID)
	fc.promote(older)

	if fc.state != newer {
		t.Fatalf("promote(older) overwrote the newer held generation: held head = %s, want %s", fc.state.stableHead.Value(), newerHeadID.Value())
	}
	if fc.state.generation != heldGen {
		t.Fatalf("held generation counter changed from %d to %d despite refused promotion", heldGen, fc.state.generation)
	}
	if older.generation != 0 {
		t.Fatalf("refused candidate was assigned generation %d, want 0 (untouched)", older.generation)
	}
}

// TestPromote_NewerHeadPromotesOverOlder is the normal direction: a fold at
// a strictly newer head replaces an older held generation and advances the
// generation counter.
func TestPromote_NewerHeadPromotesOverOlder(t *testing.T) {
	fc := newFoldCache()

	older := stateWithHead(olderHeadID)
	fc.promote(older)
	if fc.state != older {
		t.Fatalf("setup: older state was not installed into the empty cache")
	}
	if older.generation != 1 {
		t.Fatalf("first promotion generation = %d, want 1", older.generation)
	}

	newer := stateWithHead(newerHeadID)
	fc.promote(newer)

	if fc.state != newer {
		t.Fatalf("promote(newer) did not replace the older held generation: held head = %s, want %s", fc.state.stableHead.Value(), newerHeadID.Value())
	}
	if newer.generation != older.generation+1 {
		t.Fatalf("promoted generation = %d, want %d (previous+1)", newer.generation, older.generation+1)
	}
}

// TestPromote_SameHeadRepromotionReplaces: re-promoting the SAME head (e.g.
// a duplicate flight for the same key) replaces the held state — content is
// identical by construction — and keeps the generation counter moving.
func TestPromote_SameHeadRepromotionReplaces(t *testing.T) {
	fc := newFoldCache()

	first := stateWithHead(newerHeadID)
	fc.promote(first)

	second := stateWithHead(newerHeadID)
	fc.promote(second)

	if fc.state != second {
		t.Fatalf("same-head re-promotion did not replace the held state")
	}
	if second.generation != first.generation+1 {
		t.Fatalf("re-promotion generation = %d, want %d", second.generation, first.generation+1)
	}
}

// TestPromote_TimestampTieWithDifferentHeadRefuses: two DIFFERENT heads with
// the identical millisecond timestamp cannot be proven ordered, so promote()
// must keep the already-held generation (ties fail toward the held state,
// never a spurious downgrade).
func TestPromote_TimestampTieWithDifferentHeadRefuses(t *testing.T) {
	fc := newFoldCache()

	held := stateWithHead(newerHeadID)
	fc.promote(held)

	tie := stateWithHead(newerHeadTieID)
	if tie.stableHead.TimestampMS() != held.stableHead.TimestampMS() {
		t.Fatalf("test fixture broken: heads must tie on TimestampMS (%d vs %d)", tie.stableHead.TimestampMS(), held.stableHead.TimestampMS())
	}
	fc.promote(tie)

	if fc.state != held {
		t.Fatalf("timestamp-tied different head was promoted over the held generation: held head = %s, want %s", fc.state.stableHead.Value(), newerHeadID.Value())
	}
}

// TestPromote_EmptyOrHeadlessCacheAcceptsCandidate: a nil held state, or a
// held state that never observed a head (headSet=false), always accepts the
// candidate at generation 1 — there is nothing newer to protect.
func TestPromote_EmptyOrHeadlessCacheAcceptsCandidate(t *testing.T) {
	t.Run("nil held state", func(t *testing.T) {
		fc := newFoldCache()
		candidate := stateWithHead(olderHeadID)
		fc.promote(candidate)
		if fc.state != candidate {
			t.Fatalf("candidate was not installed into the empty cache")
		}
		if candidate.generation != 1 {
			t.Fatalf("generation = %d, want 1", candidate.generation)
		}
	})

	t.Run("held state without headSet", func(t *testing.T) {
		fc := newFoldCache()
		headless := newTaskFoldState() // headSet=false
		fc.state = headless
		candidate := stateWithHead(olderHeadID)
		fc.promote(candidate)
		if fc.state != candidate {
			t.Fatalf("candidate was not installed over a headless held state")
		}
		if candidate.generation != 1 {
			t.Fatalf("generation = %d, want 1", candidate.generation)
		}
	})
}

// --- zero-head branches (codex CFAR-2) ---
//
// A held state memoized against a GENUINELY EMPTY store carries the zero
// EventID as its stableHead (with headSet=true — the documented D2
// empty-store case). types.EventID{}.TimestampMS() panics (it slices an
// empty string), so promote() must decide every zero-head pairing BEFORE
// any TimestampMS call: a zero head is treated as strictly older than any
// real head (promote the real candidate), a zero candidate never displaces
// a real held head (refuse, fail-safe), and zero-vs-zero is the same-head
// replace. These tests panic, not merely fail, on the unfixed comparison.

// TestPromote_RealHeadPromotesOverZeroHeldHead: the first append after an
// empty-store memo must be able to promote (zero held head is strictly
// older than any real head). Unfixed code panics here.
func TestPromote_RealHeadPromotesOverZeroHeldHead(t *testing.T) {
	fc := newFoldCache()

	zeroHeld := stateWithHead(types.EventID{})
	fc.promote(zeroHeld)
	if fc.state != zeroHeld {
		t.Fatalf("setup: zero-head state was not installed into the empty cache")
	}

	real := stateWithHead(olderHeadID)
	fc.promote(real)

	if fc.state != real {
		t.Fatalf("real-head candidate was not promoted over the zero held head")
	}
	if real.generation != zeroHeld.generation+1 {
		t.Fatalf("promoted generation = %d, want %d", real.generation, zeroHeld.generation+1)
	}
}

// TestPromote_ZeroCandidateNeverDisplacesRealHeldHead: a candidate folded
// against an empty store is strictly older than any real held generation —
// refuse, fail-safe. Unfixed code panics here.
func TestPromote_ZeroCandidateNeverDisplacesRealHeldHead(t *testing.T) {
	fc := newFoldCache()

	held := stateWithHead(olderHeadID)
	fc.promote(held)
	heldGen := held.generation

	zeroCandidate := stateWithHead(types.EventID{})
	fc.promote(zeroCandidate)

	if fc.state != held {
		t.Fatalf("zero-head candidate displaced the real held generation")
	}
	if fc.state.generation != heldGen {
		t.Fatalf("held generation counter changed from %d to %d despite refused promotion", heldGen, fc.state.generation)
	}
	if zeroCandidate.generation != 0 {
		t.Fatalf("refused candidate was assigned generation %d, want 0 (untouched)", zeroCandidate.generation)
	}
}

// TestPromote_ZeroOverZeroIsSameHeadReplace: two empty-store folds share
// the zero head — the existing same-head branch replaces (content is
// identical by construction) and keeps the generation counter moving.
func TestPromote_ZeroOverZeroIsSameHeadReplace(t *testing.T) {
	fc := newFoldCache()

	first := stateWithHead(types.EventID{})
	fc.promote(first)

	second := stateWithHead(types.EventID{})
	fc.promote(second)

	if fc.state != second {
		t.Fatalf("zero-over-zero re-promotion did not replace the held state")
	}
	if second.generation != first.generation+1 {
		t.Fatalf("re-promotion generation = %d, want %d", second.generation, first.generation+1)
	}
}
