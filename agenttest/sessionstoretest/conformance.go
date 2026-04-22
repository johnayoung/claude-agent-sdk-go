// Package sessionstoretest provides a reusable conformance suite for
// SessionStore implementations.
//
// Run is the entry point: call it from a *_test.go file with a Factory that
// constructs a fresh, empty store. The harness exercises every contract the
// core SDK relies on — append/load round-trip, subpath isolation, cascading
// delete, ListSessions / ListSubkeys semantics, and more.
//
// Example:
//
//	func TestMyStore_Conformance(t *testing.T) {
//	    sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
//	        return NewMyStore(...)
//	    })
//	}
package sessionstoretest

import (
	"context"
	"reflect"
	"sort"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// Factory returns a fresh, empty SessionStore. Called once per subtest so
// each contract starts from a clean slate. If setup can fail (e.g. the
// backend is unreachable), call t.Skip or t.Fatal inside the factory.
type Factory func(t *testing.T) claude.SessionStore

// Run executes the full conformance suite against stores created by factory.
// Each contract runs in its own subtest so failures are isolated.
func Run(t *testing.T, factory Factory) {
	t.Helper()

	cases := []struct {
		name string
		fn   func(*testing.T, claude.SessionStore)
	}{
		{"AppendLoadRoundTrip", testAppendLoadRoundTrip},
		{"LoadUnknownReturnsEmpty", testLoadUnknownReturnsEmpty},
		{"EmptyAppendIsNoOp", testEmptyAppendIsNoOp},
		{"CrossBatchOrdering", testCrossBatchOrdering},
		{"SubpathIsolation", testSubpathIsolation},
		{"DeleteMainCascades", testDeleteMainCascades},
		{"DeleteSubpathScoped", testDeleteSubpathScoped},
		{"ListSessionsMainOnly", testListSessionsMainOnly},
		{"ListSessionsUnknownProjectEmpty", testListSessionsUnknownProjectEmpty},
		{"ListSessionsMtimeAdvances", testListSessionsMtimeAdvances},
		{"ListSubkeysEnumerates", testListSubkeysEnumerates},
		{"ListSubkeysEmptyForNoSubagents", testListSubkeysEmptyForNoSubagents},
		{"ReappendAfterDelete", testReappendAfterDelete},
		{"EntryFieldsPreserved", testEntryFieldsPreserved},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := factory(t)
			tc.fn(t, store)
		})
	}
}

// ---- contracts ----

func testAppendLoadRoundTrip(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	key := claude.SessionKey{ProjectKey: "proj", SessionID: "s1"}
	entries := []claude.SessionStoreEntry{
		{Type: "user", UUID: "u1", Timestamp: "2026-04-22T00:00:00Z"},
		{Type: "assistant", UUID: "a1", Timestamp: "2026-04-22T00:00:01Z"},
	}
	if err := store.Append(ctx, key, entries); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := store.Load(ctx, key)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, entries) {
		t.Fatalf("round-trip mismatch:\ngot  %+v\nwant %+v", got, entries)
	}
}

func testLoadUnknownReturnsEmpty(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	got, err := store.Load(ctx, claude.SessionKey{ProjectKey: "p", SessionID: "missing"})
	if err != nil {
		t.Fatalf("Load of unknown key: unexpected error %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Load of unknown key: got %d entries, want 0", len(got))
	}
}

func testEmptyAppendIsNoOp(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	key := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	if err := store.Append(ctx, key, nil); err != nil {
		t.Fatalf("Append(nil): %v", err)
	}
	if err := store.Append(ctx, key, []claude.SessionStoreEntry{}); err != nil {
		t.Fatalf("Append(empty slice): %v", err)
	}
	// Empty appends must not create a phantom session in the index.
	got, err := store.ListSessions(ctx, "p")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty append surfaced a session: got %+v", got)
	}
}

func testCrossBatchOrdering(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	key := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	batch1 := []claude.SessionStoreEntry{
		{Type: "user", UUID: "1"}, {Type: "assistant", UUID: "2"},
	}
	batch2 := []claude.SessionStoreEntry{
		{Type: "user", UUID: "3"},
	}
	batch3 := []claude.SessionStoreEntry{
		{Type: "assistant", UUID: "4"}, {Type: "user", UUID: "5"},
	}
	for _, b := range [][]claude.SessionStoreEntry{batch1, batch2, batch3} {
		if err := store.Append(ctx, key, b); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	got, err := store.Load(ctx, key)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := append(append(append([]claude.SessionStoreEntry{}, batch1...), batch2...), batch3...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func testSubpathIsolation(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	main := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	subA := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-a"}
	subB := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-b"}

	mainEntries := []claude.SessionStoreEntry{{Type: "user", UUID: "m"}}
	aEntries := []claude.SessionStoreEntry{{Type: "user", UUID: "a1"}, {Type: "assistant", UUID: "a2"}}
	bEntries := []claude.SessionStoreEntry{{Type: "user", UUID: "b1"}}

	if err := store.Append(ctx, main, mainEntries); err != nil {
		t.Fatalf("Append main: %v", err)
	}
	if err := store.Append(ctx, subA, aEntries); err != nil {
		t.Fatalf("Append subA: %v", err)
	}
	if err := store.Append(ctx, subB, bEntries); err != nil {
		t.Fatalf("Append subB: %v", err)
	}

	for _, tc := range []struct {
		key  claude.SessionKey
		want []claude.SessionStoreEntry
	}{
		{main, mainEntries},
		{subA, aEntries},
		{subB, bEntries},
	} {
		got, err := store.Load(ctx, tc.key)
		if err != nil {
			t.Fatalf("Load %+v: %v", tc.key, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("Load %+v:\n got  %+v\nwant %+v", tc.key, got, tc.want)
		}
	}
}

func testDeleteMainCascades(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	main := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	subA := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-a"}
	subB := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-b"}
	keepMain := claude.SessionKey{ProjectKey: "p", SessionID: "other"}

	mustAppend(t, store, main, "m")
	mustAppend(t, store, subA, "a")
	mustAppend(t, store, subB, "b")
	mustAppend(t, store, keepMain, "keep")

	if err := store.Delete(ctx, main); err != nil {
		t.Fatalf("Delete main: %v", err)
	}

	for _, k := range []claude.SessionKey{main, subA, subB} {
		got, err := store.Load(ctx, k)
		if err != nil {
			t.Fatalf("Load %+v after cascade: %v", k, err)
		}
		if len(got) != 0 {
			t.Fatalf("Delete main failed to cascade %+v: %+v", k, got)
		}
	}
	got, err := store.Load(ctx, keepMain)
	if err != nil {
		t.Fatalf("Load keepMain: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unrelated session disturbed by cascade: %+v", got)
	}

	subs, err := store.ListSubkeys(ctx, main)
	if err != nil {
		t.Fatalf("ListSubkeys after cascade: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("ListSubkeys after cascade: got %+v, want empty", subs)
	}
}

func testDeleteSubpathScoped(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	main := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	subA := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-a"}
	subB := claude.SessionKey{ProjectKey: "p", SessionID: "s", Subpath: "sub-b"}

	mustAppend(t, store, main, "m")
	mustAppend(t, store, subA, "a")
	mustAppend(t, store, subB, "b")

	if err := store.Delete(ctx, subA); err != nil {
		t.Fatalf("Delete subA: %v", err)
	}

	got, err := store.Load(ctx, subA)
	if err != nil {
		t.Fatalf("Load deleted subpath: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("subpath delete didn't clear entries: %+v", got)
	}

	if loaded, err := store.Load(ctx, main); err != nil || len(loaded) != 1 {
		t.Fatalf("main clobbered by subpath delete: err=%v entries=%+v", err, loaded)
	}
	if loaded, err := store.Load(ctx, subB); err != nil || len(loaded) != 1 {
		t.Fatalf("sibling subpath clobbered: err=%v entries=%+v", err, loaded)
	}

	subs, err := store.ListSubkeys(ctx, main)
	if err != nil {
		t.Fatalf("ListSubkeys: %v", err)
	}
	if !equalUnordered(subs, []string{"sub-b"}) {
		t.Fatalf("ListSubkeys after scoped delete: got %+v, want [sub-b]", subs)
	}
}

func testListSessionsMainOnly(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()

	mustAppend(t, store, claude.SessionKey{ProjectKey: "p", SessionID: "s1"}, "x")
	mustAppend(t, store, claude.SessionKey{ProjectKey: "p", SessionID: "s2"}, "y")
	// Subagent-only session must NOT surface.
	mustAppend(t, store, claude.SessionKey{ProjectKey: "p", SessionID: "s1", Subpath: "agent"}, "z")
	// Different project must be excluded.
	mustAppend(t, store, claude.SessionKey{ProjectKey: "other", SessionID: "x"}, "w")

	got, err := store.ListSessions(ctx, "p")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	ids := make([]string, len(got))
	for i, e := range got {
		ids[i] = e.SessionID
	}
	if !equalUnordered(ids, []string{"s1", "s2"}) {
		t.Fatalf("ListSessions: got %+v, want [s1 s2]", ids)
	}
}

func testListSessionsUnknownProjectEmpty(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	got, err := store.ListSessions(ctx, "never-seen")
	if err != nil {
		t.Fatalf("ListSessions unknown project: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListSessions unknown project: got %+v", got)
	}
}

func testListSessionsMtimeAdvances(t *testing.T, store claude.SessionStore) {
	key := claude.SessionKey{ProjectKey: "p", SessionID: "s"}

	mustAppend(t, store, key, "first")
	first := mtimeOf(t, store, "p", "s")

	mustAppend(t, store, key, "second")
	second := mtimeOf(t, store, "p", "s")

	if second < first {
		t.Fatalf("mtime went backwards: first=%d second=%d", first, second)
	}
	// A strict advance is the typical case; at worst mtime is equal because
	// the store quantizes to seconds. Fail only on regression.
}

func testListSubkeysEnumerates(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	main := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	mustAppend(t, store, main, "m")
	mustAppend(t, store, withSubpath(main, "sub-a"), "a")
	mustAppend(t, store, withSubpath(main, "sub-b"), "b")
	mustAppend(t, store, withSubpath(main, "sub-c"), "c")

	subs, err := store.ListSubkeys(ctx, main)
	if err != nil {
		t.Fatalf("ListSubkeys: %v", err)
	}
	if !equalUnordered(subs, []string{"sub-a", "sub-b", "sub-c"}) {
		t.Fatalf("ListSubkeys: got %+v, want [sub-a sub-b sub-c]", subs)
	}
}

func testListSubkeysEmptyForNoSubagents(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	main := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	mustAppend(t, store, main, "m")

	subs, err := store.ListSubkeys(ctx, main)
	if err != nil {
		t.Fatalf("ListSubkeys: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("ListSubkeys for sub-less session: got %+v", subs)
	}
}

func testReappendAfterDelete(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	key := claude.SessionKey{ProjectKey: "p", SessionID: "s"}

	mustAppend(t, store, key, "first")
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	mustAppend(t, store, key, "second")

	got, err := store.Load(ctx, key)
	if err != nil {
		t.Fatalf("Load after resurrect: %v", err)
	}
	if len(got) != 1 || got[0].UUID != "second" {
		t.Fatalf("resurrected session contents wrong: %+v", got)
	}

	list, err := store.ListSessions(ctx, "p")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(list) != 1 || list[0].SessionID != "s" {
		t.Fatalf("ListSessions after resurrect: %+v", list)
	}
}

func testEntryFieldsPreserved(t *testing.T, store claude.SessionStore) {
	ctx := context.Background()
	key := claude.SessionKey{ProjectKey: "p", SessionID: "s"}
	entries := []claude.SessionStoreEntry{
		{Type: "user", UUID: "11111111-1111-1111-1111-111111111111", Timestamp: "2026-04-22T12:00:00.000Z"},
		{Type: "assistant"}, // minimal fields
		{Type: "system", UUID: "", Timestamp: "2026-04-22T12:00:01Z"},
	}
	if err := store.Append(ctx, key, entries); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := store.Load(ctx, key)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, entries) {
		t.Fatalf("entry fields not preserved:\ngot  %+v\nwant %+v", got, entries)
	}
}

// ---- helpers ----

func mustAppend(t *testing.T, store claude.SessionStore, key claude.SessionKey, uuid string) {
	t.Helper()
	err := store.Append(context.Background(), key, []claude.SessionStoreEntry{{Type: "user", UUID: uuid}})
	if err != nil {
		t.Fatalf("Append(%+v, %q): %v", key, uuid, err)
	}
}

func withSubpath(k claude.SessionKey, sub string) claude.SessionKey {
	k.Subpath = sub
	return k
}

func mtimeOf(t *testing.T, store claude.SessionStore, projectKey, sessionID string) int64 {
	t.Helper()
	list, err := store.ListSessions(context.Background(), projectKey)
	if err != nil {
		t.Fatalf("ListSessions(%q): %v", projectKey, err)
	}
	for _, e := range list {
		if e.SessionID == sessionID {
			return e.Mtime
		}
	}
	t.Fatalf("session %q not found in ListSessions(%q) result %+v", sessionID, projectKey, list)
	return 0
}

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	return reflect.DeepEqual(ac, bc)
}
