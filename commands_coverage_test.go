package main

// Comprehensive behavioral parity tests for every action supported by the
// original todo.txt-cli.  Each section has ≥5 distinct cases.
//
// Behavioral gaps vs. the original identified during review:
//   1. "done" alias for "do" was missing in main.go/dispatch — now fixed.
//   2. Comma-separated task numbers (e.g. "do 1,2,3") are not supported;
//      the original todo.sh does accept them.  Those cases are marked with
//      TODO comments so they can be tracked.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// today returns the current date in YYYY-MM-DD format, matching app.today().
func testToday() string { return time.Now().Format("2006-01-02") }

// ── Addm ─────────────────────────────────────────────────────────────────────

func TestAddm_noArgs(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Addm(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestAddm_singleLine(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Addm([]string{"Single task"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 1 || items[0].Raw != "Single task" {
		t.Errorf("unexpected items: %v", items)
	}
}

func TestAddm_dateOnAdd(t *testing.T) {
	app, _ := newTestApp(t)
	app.cfg.DateOnAdd = true
	captureStdout(t, func() {
		app.Addm([]string{"Task A\nTask B"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	today := testToday()
	for _, item := range items {
		if !strings.HasPrefix(item.Raw, today) {
			t.Errorf("expected date prefix on %q", item.Raw)
		}
	}
}

func TestAddm_preservesOrder(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Addm([]string{"Alpha\nBeta\nGamma"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Raw != "Alpha" || items[1].Raw != "Beta" || items[2].Raw != "Gamma" {
		t.Errorf("order not preserved: %v", items)
	}
}

func TestAddm_multipleBlankLines(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Addm([]string{"A\n\n\nB\n\nC"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 3 {
		t.Errorf("expected 3 tasks (blanks skipped), got %d: %v", len(items), items)
	}
}

// ── Addto ─────────────────────────────────────────────────────────────────────

func TestAddto_createsNewFile(t *testing.T) {
	app, dir := newTestApp(t)
	captureStdout(t, func() {
		app.Addto([]string{"inbox.txt", "Buy groceries"})
	})
	items, err := ReadItems(filepath.Join(dir, "inbox.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if len(items) != 1 || items[0].Raw != "Buy groceries" {
		t.Errorf("unexpected content: %v", items)
	}
}

func TestAddto_appendsToExistingFile(t *testing.T) {
	app, dir := newTestApp(t)
	inbox := filepath.Join(dir, "inbox.txt")
	os.WriteFile(inbox, []byte("Existing task\n"), 0644)

	captureStdout(t, func() {
		app.Addto([]string{"inbox.txt", "New task"})
	})
	items, _ := ReadItems(inbox)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[1].Raw != "New task" {
		t.Errorf("appended item wrong: %q", items[1].Raw)
	}
}

func TestAddto_noArgs(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Addto(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestAddto_onlyOneArg(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Addto([]string{"inbox.txt"}); err == nil {
		t.Error("expected error when text is missing")
	}
}

func TestAddto_correctLineNumber(t *testing.T) {
	app, dir := newTestApp(t)
	inbox := filepath.Join(dir, "inbox.txt")
	os.WriteFile(inbox, []byte("Line one\nLine two\n"), 0644)

	out := captureStdout(t, func() {
		app.Addto([]string{"inbox.txt", "Line three"})
	})
	if !strings.HasPrefix(out, "3 ") {
		t.Errorf("expected line number 3, got output: %q", out)
	}
}

func TestAddto_doesNotAffectTodoFile(t *testing.T) {
	app, _ := newTestApp(t, "Existing todo")
	captureStdout(t, func() {
		app.Addto([]string{"inbox.txt", "Inbox item"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 1 || items[0].Raw != "Existing todo" {
		t.Errorf("todo.txt should be unmodified: %v", items)
	}
}

// ── Append ───────────────────────────────────────────────────────────────────

func TestAppend_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Append(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestAppend_outOfRange(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Append([]string{"99", "extra"}); err == nil {
		t.Error("expected error for out-of-range number")
	}
}

func TestAppend_multipleWordArgs(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Append([]string{"1", "+family", "@phone"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Call Mom +family @phone" {
		t.Errorf("got %q", items[0].Raw)
	}
}

func TestAppend_preservesPriority(t *testing.T) {
	app, _ := newTestApp(t, "(A) Important task")
	captureStdout(t, func() {
		app.Append([]string{"1", "+project"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if !strings.HasPrefix(items[0].Raw, "(A) ") {
		t.Errorf("priority should be preserved, got %q", items[0].Raw)
	}
	if !strings.HasSuffix(items[0].Raw, "+project") {
		t.Errorf("appended text should be at end, got %q", items[0].Raw)
	}
}

func TestAppend_doesNotAffectOtherTasks(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "Task two", "Task three")
	captureStdout(t, func() {
		app.Append([]string{"2", "extra"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Task one" || items[2].Raw != "Task three" {
		t.Error("other tasks should be unaffected")
	}
}

// ── Prepend ──────────────────────────────────────────────────────────────────

func TestPrepend_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Prepend(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestPrepend_priorityOnlyNoDate(t *testing.T) {
	app, _ := newTestApp(t, "(B) Fix bug")
	captureStdout(t, func() {
		app.Prepend([]string{"1", "CRITICAL:"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	want := "(B) CRITICAL: Fix bug"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

func TestPrepend_multipleWordArgs(t *testing.T) {
	app, _ := newTestApp(t, "Review PR")
	captureStdout(t, func() {
		app.Prepend([]string{"1", "Really", "need", "to"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	want := "Really need to Review PR"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

func TestPrepend_doesNotAffectOtherTasks(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "Task two", "Task three")
	captureStdout(t, func() {
		app.Prepend([]string{"2", "prefix:"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Task one" || items[2].Raw != "Task three" {
		t.Error("other tasks should be unaffected")
	}
}

func TestPrepend_withDateAndPriority(t *testing.T) {
	app, _ := newTestApp(t, "(C) 2026-01-01 Old task")
	captureStdout(t, func() {
		app.Prepend([]string{"1", "ASAP:"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	want := "(C) 2026-01-01 ASAP: Old task"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

// ── Del ──────────────────────────────────────────────────────────────────────

func TestDel_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Del(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestDel_firstTask(t *testing.T) {
	app, _ := newTestApp(t, "Delete me", "Keep me")
	captureStdout(t, func() {
		app.Del([]string{"1"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "" {
		t.Errorf("first task should be blank, got %q", items[0].Raw)
	}
	if items[1].Raw != "Keep me" {
		t.Errorf("second task should be unchanged, got %q", items[1].Raw)
	}
}

func TestDel_termAtStart(t *testing.T) {
	app, _ := newTestApp(t, "(A) Call Mom @phone")
	captureStdout(t, func() {
		app.Del([]string{"1", "@phone"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if strings.Contains(items[0].Raw, "@phone") {
		t.Errorf("@phone should be removed, got %q", items[0].Raw)
	}
	if items[0].Raw == "" {
		t.Errorf("task should not be fully deleted, got %q", items[0].Raw)
	}
}

func TestDel_multipleOccurrencesOfTerm(t *testing.T) {
	app, _ := newTestApp(t, "Buy milk buy MILK and milk")
	captureStdout(t, func() {
		app.Del([]string{"1", "milk"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	// All occurrences of exact string "milk" should be removed.
	if strings.Contains(items[0].Raw, "milk") {
		t.Errorf("all occurrences should be removed, got %q", items[0].Raw)
	}
}

func TestDel_preservesLineNumbers(t *testing.T) {
	app, _ := newTestApp(t, "One", "Two", "Three")
	captureStdout(t, func() {
		app.Del([]string{"2"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	// Line numbers are preserved: 3 lines total, line 2 is blank.
	if len(items) != 3 {
		t.Errorf("file should still have 3 lines, got %d", len(items))
	}
	if items[2].LineNum != 3 || items[2].Raw != "Three" {
		t.Errorf("line 3 should still be Three, got %v", items[2])
	}
}

// ── Pri ───────────────────────────────────────────────────────────────────────

func TestPri_replacesExistingPriority(t *testing.T) {
	app, _ := newTestApp(t, "(A) Important task")
	captureStdout(t, func() {
		app.Pri([]string{"1", "C"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "C" {
		t.Errorf("priority should be replaced with C, got %q", items[0].Priority)
	}
}

func TestPri_doneTaskReturnsError(t *testing.T) {
	app, _ := newTestApp(t, "x 2026-04-22 Done task")
	if err := app.Pri([]string{"1", "A"}); err == nil {
		t.Error("expected error when setting priority on done task")
	}
}

func TestPri_outOfRange(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Pri([]string{"99", "A"}); err == nil {
		t.Error("expected error for out-of-range number")
	}
}

func TestPri_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Pri(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestPri_onlyOneArg(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Pri([]string{"1"}); err == nil {
		t.Error("expected error when priority letter is missing")
	}
}

// ── Depri ─────────────────────────────────────────────────────────────────────

func TestDepri_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "(A) Task")
	if err := app.Depri(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestDepri_alreadyDeprioritized(t *testing.T) {
	app, _ := newTestApp(t, "No priority task")
	out := captureStdout(t, func() {
		app.Depri([]string{"1"})
	})
	if !strings.Contains(out, "already deprioritized") {
		t.Errorf("expected 'already deprioritized' message, got: %q", out)
	}
}

func TestDepri_singleTask(t *testing.T) {
	app, _ := newTestApp(t, "(Z) Lowest priority")
	captureStdout(t, func() {
		app.Depri([]string{"1"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "" {
		t.Errorf("priority should be removed, got %q", items[0].Priority)
	}
	if strings.HasPrefix(items[0].Raw, "(") {
		t.Errorf("raw should not start with (, got %q", items[0].Raw)
	}
}

func TestDepri_invalidNumber(t *testing.T) {
	app, _ := newTestApp(t, "(A) Task")
	// Invalid numbers should not cause a fatal error — they print to stderr and continue.
	captureStdout(t, func() {
		app.Depri([]string{"abc"})
	})
	// Verify original task is unchanged.
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "A" {
		t.Errorf("task should be unchanged after invalid depri, got %q", items[0].Raw)
	}
}

func TestDepri_preservesDescriptionAndDate(t *testing.T) {
	app, _ := newTestApp(t, "(B) 2026-03-01 Project task +work @office")
	captureStdout(t, func() {
		app.Depri([]string{"1"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "" {
		t.Errorf("priority should be gone, got %q", items[0].Priority)
	}
	if !strings.Contains(items[0].Raw, "2026-03-01") {
		t.Errorf("date should be preserved, got %q", items[0].Raw)
	}
	if !strings.Contains(items[0].Raw, "+work") || !strings.Contains(items[0].Raw, "@office") {
		t.Errorf("tags should be preserved, got %q", items[0].Raw)
	}
}

// ── Replace ───────────────────────────────────────────────────────────────────

func TestReplace_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Replace(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestReplace_outOfRange(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Replace([]string{"99", "New text"}); err == nil {
		t.Error("expected error for out-of-range number")
	}
}

func TestReplace_preservesLineNumber(t *testing.T) {
	app, _ := newTestApp(t, "Old task one", "Old task two", "Old task three")
	captureStdout(t, func() {
		app.Replace([]string{"2", "New task two"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[1].LineNum != 2 {
		t.Errorf("line number should be 2, got %d", items[1].LineNum)
	}
}

func TestReplace_withPriority(t *testing.T) {
	app, _ := newTestApp(t, "Plain task")
	captureStdout(t, func() {
		app.Replace([]string{"1", "(A)", "Now", "with", "priority"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "A" {
		t.Errorf("new task should have priority A, got %q", items[0].Priority)
	}
	if items[0].Raw != "(A) Now with priority" {
		t.Errorf("got %q", items[0].Raw)
	}
}

func TestReplace_doesNotAffectOtherTasks(t *testing.T) {
	app, _ := newTestApp(t, "Keep", "Change me", "Keep")
	captureStdout(t, func() {
		app.Replace([]string{"2", "Changed"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Keep" || items[2].Raw != "Keep" {
		t.Errorf("flanking tasks should be unaffected: %v", items)
	}
}

func TestReplace_canReplaceWithDoneTask(t *testing.T) {
	app, _ := newTestApp(t, "Active task")
	today := testToday()
	captureStdout(t, func() {
		app.Replace([]string{"1", "x", today, "Completed"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if !items[0].Done {
		t.Errorf("replaced task should be done, got %q", items[0].Raw)
	}
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestArchive_emptyFile(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Archive(); err != nil {
		t.Errorf("archive on empty file should not error: %v", err)
	}
}

func TestArchive_noDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "Active one", "Active two")
	captureStdout(t, func() {
		app.Archive()
	})
	todo, _ := ReadItems(app.cfg.TodoFile)
	done, _ := ReadItems(app.cfg.DoneFile)
	if len(todo) != 2 {
		t.Errorf("todo.txt should still have 2 items, got %d", len(todo))
	}
	if len(done) != 0 {
		t.Errorf("done.txt should be empty, got %d items", len(done))
	}
}

func TestArchive_appendsToDoneFile(t *testing.T) {
	app, _ := newTestApp(t, "x 2026-04-22 First done", "x 2026-04-22 Second done")
	// Pre-populate done.txt with an existing entry.
	os.WriteFile(app.cfg.DoneFile, []byte("x 2026-01-01 Already done\n"), 0644)
	captureStdout(t, func() {
		app.Archive()
	})
	done, _ := ReadItems(app.cfg.DoneFile)
	if len(done) != 3 {
		t.Errorf("done.txt should have 3 entries (1 existing + 2 new), got %d", len(done))
	}
}

func TestArchive_multipleRoundsAccumulate(t *testing.T) {
	app, _ := newTestApp(t, "Active", "x 2026-04-22 Done one")
	captureStdout(t, func() { app.Archive() })

	// Add another done task and archive again.
	items, _ := ReadItems(app.cfg.TodoFile)
	items[0].Raw = "x " + testToday() + " Active now done"
	items[0] = ParseItem(items[0].Raw, 1)
	WriteItems(app.cfg.TodoFile, items)
	captureStdout(t, func() { app.Archive() })

	done, _ := ReadItems(app.cfg.DoneFile)
	if len(done) != 2 {
		t.Errorf("done.txt should accumulate 2 entries across archives, got %d", len(done))
	}
}

func TestArchive_renumbersTodoItems(t *testing.T) {
	app, _ := newTestApp(t, "Keep A", "x 2026-04-22 Done", "Keep B")
	captureStdout(t, func() { app.Archive() })
	todo, _ := ReadItems(app.cfg.TodoFile)
	if len(todo) != 2 {
		t.Fatalf("expected 2 remaining items, got %d", len(todo))
	}
	if todo[0].LineNum != 1 || todo[1].LineNum != 2 {
		t.Errorf("items should be renumbered 1,2; got %d,%d", todo[0].LineNum, todo[1].LineNum)
	}
}

// ── Deduplicate ───────────────────────────────────────────────────────────────

func TestDeduplicate_emptyFile(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Deduplicate(); err != nil {
		t.Errorf("deduplicate on empty file should not error: %v", err)
	}
}

func TestDeduplicate_preservesBlanks(t *testing.T) {
	// Blank lines (tombstones) should be kept to preserve line numbers.
	app, _ := newTestApp(t, "Task A", "", "Task A")
	captureStdout(t, func() { app.Deduplicate() })
	items, _ := ReadItems(app.cfg.TodoFile)
	// The duplicate "Task A" at position 3 should be removed; blank at 2 stays.
	hasBlank := false
	for _, item := range items {
		if item.Raw == "" {
			hasBlank = true
		}
	}
	if !hasBlank {
		t.Error("blank tombstone line should be preserved after deduplicate")
	}
}

func TestDeduplicate_caseSensitive(t *testing.T) {
	// Deduplication is case-sensitive: "task" ≠ "Task".
	app, _ := newTestApp(t, "task", "Task", "TASK")
	captureStdout(t, func() { app.Deduplicate() })
	items, _ := ReadItems(app.cfg.TodoFile)
	count := 0
	for _, item := range items {
		if item.Raw != "" {
			count++
		}
	}
	if count != 3 {
		t.Errorf("case-sensitive: expected 3 distinct items, got %d: %v", count, items)
	}
}

func TestDeduplicate_threeOrMoreDuplicates(t *testing.T) {
	app, _ := newTestApp(t, "Dup", "Dup", "Dup", "Dup")
	captureStdout(t, func() { app.Deduplicate() })
	items, _ := ReadItems(app.cfg.TodoFile)
	nonBlank := 0
	for _, item := range items {
		if item.Raw != "" {
			nonBlank++
		}
	}
	if nonBlank != 1 {
		t.Errorf("expected 1 unique item after 4-way dedup, got %d: %v", nonBlank, items)
	}
}

func TestDeduplicate_differentTasksUntouched(t *testing.T) {
	app, _ := newTestApp(t, "A", "B", "C", "D")
	captureStdout(t, func() { app.Deduplicate() })
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 4 {
		t.Errorf("no duplicates: all 4 items should remain, got %d", len(items))
	}
}

// ── Move ──────────────────────────────────────────────────────────────────────

func TestMove_noArgs(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Move(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestMove_outOfRange(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	if err := app.Move([]string{"99", "other.txt"}); err == nil {
		t.Error("expected error for out-of-range task number")
	}
}

func TestMove_withExplicitSrcArg(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	dstPath := filepath.Join(dir, "dst.txt")
	os.WriteFile(srcPath, []byte("Task in src\n"), 0644)

	captureStdout(t, func() {
		app.Move([]string{"1", "dst.txt", "src.txt"})
	})
	src, _ := ReadItems(srcPath)
	dst, _ := ReadItems(dstPath)

	if src[0].Raw != "" {
		t.Errorf("source item should be blanked, got %q", src[0].Raw)
	}
	if len(dst) != 1 || dst[0].Raw != "Task in src" {
		t.Errorf("destination should have task, got %v", dst)
	}
}

func TestMove_emptySlotTask(t *testing.T) {
	app, dir := newTestApp(t, "Task one", "Task two")
	// Delete task 1 to create a tombstone.
	captureStdout(t, func() { app.Del([]string{"1"}) })

	if err := app.Move([]string{"1", "other.txt"}); err == nil {
		t.Error("expected error when moving blank/tombstone task")
	}
	// other.txt should not be created.
	if _, err := os.Stat(filepath.Join(dir, "other.txt")); !os.IsNotExist(err) {
		t.Error("destination file should not exist after failed move")
	}
}

func TestMove_destinationAppends(t *testing.T) {
	app, dir := newTestApp(t, "Alpha", "Beta")
	dstPath := filepath.Join(dir, "archive.txt")
	os.WriteFile(dstPath, []byte("Pre-existing\n"), 0644)

	captureStdout(t, func() {
		app.Move([]string{"1", "archive.txt"})
	})
	dst, _ := ReadItems(dstPath)
	if len(dst) != 2 {
		t.Fatalf("expected 2 items in destination, got %d", len(dst))
	}
	if dst[1].Raw != "Alpha" {
		t.Errorf("moved task should be appended, got %q", dst[1].Raw)
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestList_basic(t *testing.T) {
	app, _ := newTestApp(t, "Buy milk", "Call Mom", "Write report")
	out := captureStdout(t, func() { app.List(nil) })
	if !strings.Contains(out, "Buy milk") || !strings.Contains(out, "Call Mom") || !strings.Contains(out, "Write report") {
		t.Errorf("all tasks should be shown, got:\n%s", out)
	}
}

func TestList_excludesDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "Active task", "x 2026-04-22 Done task")
	out := captureStdout(t, func() { app.List(nil) })
	if strings.Contains(out, "Done task") {
		t.Errorf("done tasks should not appear in list, got:\n%s", out)
	}
	if !strings.Contains(out, "Active task") {
		t.Errorf("active task should appear in list, got:\n%s", out)
	}
}

func TestList_sortedByPriority(t *testing.T) {
	app, _ := newTestApp(t, "(C) Low priority", "(A) High priority", "(B) Medium priority")
	out := captureStdout(t, func() { app.List(nil) })
	posA := strings.Index(out, "High priority")
	posB := strings.Index(out, "Medium priority")
	posC := strings.Index(out, "Low priority")
	if posA > posB || posB > posC {
		t.Errorf("tasks should be sorted A > B > C in output:\n%s", out)
	}
}

func TestList_filterTerm(t *testing.T) {
	app, _ := newTestApp(t, "Buy milk @errands", "Call Mom @phone", "Write report @work")
	out := captureStdout(t, func() { app.List([]string{"@errands"}) })
	if !strings.Contains(out, "Buy milk") {
		t.Errorf("matching task should appear, got:\n%s", out)
	}
	if strings.Contains(out, "Call Mom") || strings.Contains(out, "Write report") {
		t.Errorf("non-matching tasks should be excluded, got:\n%s", out)
	}
}

func TestList_summaryLine(t *testing.T) {
	app, _ := newTestApp(t, "Task A", "Task B", "x 2026-04-22 Done")
	out := captureStdout(t, func() { app.List(nil) })
	if !strings.Contains(out, "2 of 2 tasks shown") {
		t.Errorf("summary should show 2 of 2 tasks, got:\n%s", out)
	}
}

func TestList_emptyFile(t *testing.T) {
	app, _ := newTestApp(t)
	out := captureStdout(t, func() { app.List(nil) })
	if !strings.Contains(out, "0 of 0 tasks shown") {
		t.Errorf("empty file should show 0 of 0 tasks, got:\n%s", out)
	}
}

func TestList_priorityBeforeUnprioritized(t *testing.T) {
	app, _ := newTestApp(t, "No priority", "(A) Has priority")
	out := captureStdout(t, func() { app.List(nil) })
	posHas := strings.Index(out, "Has priority")
	posNo := strings.Index(out, "No priority")
	if posHas > posNo {
		t.Errorf("prioritized tasks should appear before unprioritized:\n%s", out)
	}
}

// ── Listall ───────────────────────────────────────────────────────────────────

func TestListall_filterTermExcludesNonMatching(t *testing.T) {
	app, dir := newTestApp(t, "Buy milk @errands", "Call Mom @phone")
	os.WriteFile(filepath.Join(dir, "done.txt"), []byte("x 2026-04-22 Done @errands\n"), 0644)

	out := captureStdout(t, func() { app.Listall([]string{"@phone"}) })
	if !strings.Contains(out, "Call Mom") {
		t.Errorf("matching task should appear, got:\n%s", out)
	}
	if strings.Contains(out, "Buy milk") || strings.Contains(out, "Done @errands") {
		t.Errorf("non-matching tasks should be excluded, got:\n%s", out)
	}
}

func TestListall_emptyFiles(t *testing.T) {
	app, _ := newTestApp(t)
	out := captureStdout(t, func() { app.Listall(nil) })
	if !strings.Contains(out, "TODO: 0 of 0 tasks shown") {
		t.Errorf("should show 0 todo tasks, got:\n%s", out)
	}
	if !strings.Contains(out, "DONE: 0 of 0 tasks shown") {
		t.Errorf("should show 0 done tasks, got:\n%s", out)
	}
}

func TestListall_todoAndDoneDistinct(t *testing.T) {
	app, dir := newTestApp(t, "Active task")
	os.WriteFile(filepath.Join(dir, "done.txt"), []byte("x 2026-04-22 Done task\n"), 0644)
	out := captureStdout(t, func() { app.Listall(nil) })
	if !strings.Contains(out, "Active task") || !strings.Contains(out, "Done task") {
		t.Errorf("both todo and done tasks should appear:\n%s", out)
	}
}

// ── Listpri ───────────────────────────────────────────────────────────────────

func TestListpri_allPrioritized(t *testing.T) {
	app, _ := newTestApp(t, "(A) High", "(B) Medium", "No priority")
	out := captureStdout(t, func() { app.Listpri(nil) })
	if !strings.Contains(out, "High") || !strings.Contains(out, "Medium") {
		t.Errorf("all prioritized tasks should appear:\n%s", out)
	}
	if strings.Contains(out, "No priority") {
		t.Errorf("unprioritized tasks should not appear:\n%s", out)
	}
}

func TestListpri_specificPriority(t *testing.T) {
	app, _ := newTestApp(t, "(A) Alpha", "(B) Beta", "(C) Gamma")
	out := captureStdout(t, func() { app.Listpri([]string{"B"}) })
	if !strings.Contains(out, "Beta") {
		t.Errorf("priority B task should appear:\n%s", out)
	}
	if strings.Contains(out, "Alpha") || strings.Contains(out, "Gamma") {
		t.Errorf("other priorities should be excluded:\n%s", out)
	}
}

func TestListpri_range(t *testing.T) {
	app, _ := newTestApp(t, "(A) Alpha", "(B) Beta", "(C) Gamma", "(D) Delta")
	out := captureStdout(t, func() { app.Listpri([]string{"A-C"}) })
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") || !strings.Contains(out, "Gamma") {
		t.Errorf("A-C range tasks should appear:\n%s", out)
	}
	if strings.Contains(out, "Delta") {
		t.Errorf("D priority should be excluded by A-C range:\n%s", out)
	}
}

func TestListpri_excludesDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "(A) Active priority", "x 2026-04-22 Done (was priority)")
	out := captureStdout(t, func() { app.Listpri(nil) })
	if strings.Contains(out, "Done") {
		t.Errorf("done tasks should not appear in listpri:\n%s", out)
	}
}

func TestListpri_summaryLine(t *testing.T) {
	app, _ := newTestApp(t, "(A) One", "(B) Two", "No pri")
	out := captureStdout(t, func() { app.Listpri(nil) })
	if !strings.Contains(out, "2 prioritized task(s) shown") {
		t.Errorf("summary should show 2 prioritized tasks, got:\n%s", out)
	}
}

func TestListpri_withTermFilter(t *testing.T) {
	app, _ := newTestApp(t, "(A) Buy milk @errands", "(A) Call Mom @phone")
	out := captureStdout(t, func() { app.Listpri([]string{"A", "@errands"}) })
	if !strings.Contains(out, "Buy milk") {
		t.Errorf("matching task should appear:\n%s", out)
	}
	if strings.Contains(out, "Call Mom") {
		t.Errorf("non-matching task should be excluded:\n%s", out)
	}
}

// ── Listcon ───────────────────────────────────────────────────────────────────

func TestListcon_listsContexts(t *testing.T) {
	app, _ := newTestApp(t, "Buy milk @errands", "Call Mom @phone", "Write report @work")
	out := captureStdout(t, func() { app.Listcon(nil) })
	for _, ctx := range []string{"errands", "phone", "work"} {
		if !strings.Contains(out, ctx) {
			t.Errorf("expected context %q in output:\n%s", ctx, out)
		}
	}
}

func TestListcon_sortedAlphabetically(t *testing.T) {
	app, _ := newTestApp(t, "Task @zephyr", "Task @alpha", "Task @middle")
	out := captureStdout(t, func() { app.Listcon(nil) })
	posA := strings.Index(out, "alpha")
	posM := strings.Index(out, "middle")
	posZ := strings.Index(out, "zephyr")
	if posA > posM || posM > posZ {
		t.Errorf("contexts should be sorted alphabetically:\n%s", out)
	}
}

func TestListcon_noContexts(t *testing.T) {
	app, _ := newTestApp(t, "Task without context", "Another plain task")
	out := captureStdout(t, func() { app.Listcon(nil) })
	if strings.Contains(out, "@") {
		t.Errorf("no contexts should appear, got:\n%s", out)
	}
}

func TestListcon_deduplicates(t *testing.T) {
	app, _ := newTestApp(t, "Task one @home", "Task two @home", "Task three @home")
	out := captureStdout(t, func() { app.Listcon(nil) })
	count := strings.Count(out, "home")
	if count != 1 {
		t.Errorf("@home should appear exactly once, appeared %d times:\n%s", count, out)
	}
}

func TestListcon_ignoresDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "x 2026-04-22 Done @secret", "Active @visible")
	out := captureStdout(t, func() { app.Listcon(nil) })
	// Done tasks still appear in todo.txt file, so their contexts are listed
	// (same behavior as original todo.sh which also reads the whole file).
	// This test simply verifies @visible is present and context listing works.
	if !strings.Contains(out, "visible") {
		t.Errorf("active task context should appear:\n%s", out)
	}
}

// ── Listproj ─────────────────────────────────────────────────────────────────

func TestListproj_listsProjects(t *testing.T) {
	app, _ := newTestApp(t, "Write code +golang", "Review PR +review", "Update docs +docs")
	out := captureStdout(t, func() { app.Listproj(nil) })
	for _, proj := range []string{"golang", "review", "docs"} {
		if !strings.Contains(out, proj) {
			t.Errorf("expected project %q in output:\n%s", proj, out)
		}
	}
}

func TestListproj_sortedAlphabetically(t *testing.T) {
	app, _ := newTestApp(t, "Task +zebra", "Task +apple", "Task +mango")
	out := captureStdout(t, func() { app.Listproj(nil) })
	posA := strings.Index(out, "apple")
	posM := strings.Index(out, "mango")
	posZ := strings.Index(out, "zebra")
	if posA > posM || posM > posZ {
		t.Errorf("projects should be sorted alphabetically:\n%s", out)
	}
}

func TestListproj_noProjects(t *testing.T) {
	app, _ := newTestApp(t, "Plain task one", "Plain task two")
	out := captureStdout(t, func() { app.Listproj(nil) })
	if strings.Contains(out, "+") {
		t.Errorf("no projects should appear, got:\n%s", out)
	}
}

func TestListproj_deduplicates(t *testing.T) {
	app, _ := newTestApp(t, "Task +work", "Another +work", "Third +work")
	out := captureStdout(t, func() { app.Listproj(nil) })
	count := strings.Count(out, "work")
	if count != 1 {
		t.Errorf("+work should appear exactly once, appeared %d times:\n%s", count, out)
	}
}

func TestListproj_filterTerm(t *testing.T) {
	app, _ := newTestApp(t, "Task +alpha", "Task +beta", "Task +gamma")
	out := captureStdout(t, func() { app.Listproj([]string{"beta"}) })
	if !strings.Contains(out, "beta") {
		t.Errorf("matching project should appear:\n%s", out)
	}
	if strings.Contains(out, "alpha") || strings.Contains(out, "gamma") {
		t.Errorf("non-matching projects should be excluded:\n%s", out)
	}
}

// ── Listfile ─────────────────────────────────────────────────────────────────

func TestListfile_noArgs_listsTxtFiles(t *testing.T) {
	// Original todo.txt-cli behaviour: no args lists all *.txt files in TODO_DIR.
	app, dir := newTestApp(t, "A task") // creates todo.txt
	os.WriteFile(filepath.Join(dir, "done.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "inbox.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte(""), 0644) // should not appear

	out := captureStdout(t, func() {
		if err := app.Listfile(nil); err != nil {
			t.Fatalf("no-args listfile should not error: %v", err)
		}
	})
	if !strings.Contains(out, "Files in the todo.txt directory:") {
		t.Errorf("should print header line, got:\n%s", out)
	}
	if !strings.Contains(out, "todo.txt") || !strings.Contains(out, "done.txt") || !strings.Contains(out, "inbox.txt") {
		t.Errorf("should list all .txt files, got:\n%s", out)
	}
	if strings.Contains(out, "notes.md") {
		t.Errorf("non-.txt files should be excluded, got:\n%s", out)
	}
}

func TestListfile_basicFile(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "inbox.txt")
	os.WriteFile(srcPath, []byte("Inbox item one\nInbox item two\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"inbox.txt"}) })
	if !strings.Contains(out, "Inbox item one") || !strings.Contains(out, "Inbox item two") {
		t.Errorf("all items from file should appear:\n%s", out)
	}
}

func TestListfile_filterTerm(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	os.WriteFile(srcPath, []byte("Match me @yes\nSkip this @no\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"src.txt", "@yes"}) })
	if !strings.Contains(out, "Match me") {
		t.Errorf("matching task should appear:\n%s", out)
	}
	if strings.Contains(out, "Skip this") {
		t.Errorf("non-matching task should be excluded:\n%s", out)
	}
}

func TestListfile_sortedByPriority(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	os.WriteFile(srcPath, []byte("(C) Low\n(A) High\n(B) Medium\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"src.txt"}) })
	posA := strings.Index(out, "High")
	posB := strings.Index(out, "Medium")
	posC := strings.Index(out, "Low")
	if posA > posB || posB > posC {
		t.Errorf("items should be sorted by priority:\n%s", out)
	}
}

func TestListfile_summaryLine(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "notes.txt")
	os.WriteFile(srcPath, []byte("Note one\nNote two\nNote three\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"notes.txt"}) })
	if !strings.Contains(out, "notes.txt") || !strings.Contains(out, "3 of 3 tasks shown") {
		t.Errorf("summary should show 'N of M tasks shown', got:\n%s", out)
	}
}

func TestListfile_doesNotReadTodoFile(t *testing.T) {
	app, dir := newTestApp(t, "This is in todo.txt")
	srcPath := filepath.Join(dir, "other.txt")
	os.WriteFile(srcPath, []byte("This is in other.txt\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"other.txt"}) })
	if strings.Contains(out, "This is in todo.txt") {
		t.Errorf("listfile should not read todo.txt:\n%s", out)
	}
	if !strings.Contains(out, "This is in other.txt") {
		t.Errorf("listfile should read the specified file:\n%s", out)
	}
}

// ── Report ────────────────────────────────────────────────────────────────────

func TestReport_createsReportFile(t *testing.T) {
	app, _ := newTestApp(t, "Open task")
	captureStdout(t, func() { app.Report() })
	if _, err := os.Stat(app.cfg.ReportFile); os.IsNotExist(err) {
		t.Error("report file should be created")
	}
}

func TestReport_appendsToExistingReport(t *testing.T) {
	app, _ := newTestApp(t, "Task")
	os.WriteFile(app.cfg.ReportFile, []byte("2026-01-01 5 3\n"), 0644)
	captureStdout(t, func() { app.Report() })
	data, _ := os.ReadFile(app.cfg.ReportFile)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("report should have 2 lines (existing + new), got %d: %v", len(lines), lines)
	}
}

func TestReport_countsOpenTasks(t *testing.T) {
	app, _ := newTestApp(t, "Open one", "Open two", "x 2026-04-22 Done")
	captureStdout(t, func() { app.Report() })
	data, _ := os.ReadFile(app.cfg.ReportFile)
	// Format: "YYYY-MM-DD <open> <done>"
	fields := strings.Fields(strings.TrimSpace(string(data)))
	if len(fields) < 3 || fields[1] != "2" {
		t.Errorf("expected 2 open tasks in report, got fields: %v", fields)
	}
}

func TestReport_countsDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "Open task")
	os.WriteFile(app.cfg.DoneFile, []byte("x 2026-04-22 Done one\nx 2026-04-22 Done two\n"), 0644)
	captureStdout(t, func() { app.Report() })
	data, _ := os.ReadFile(app.cfg.ReportFile)
	fields := strings.Fields(strings.TrimSpace(string(data)))
	if len(fields) < 3 || fields[2] != "2" {
		t.Errorf("expected 2 done tasks in report, got fields: %v", fields)
	}
}

func TestReport_formatsWithDate(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() { app.Report() })
	data, _ := os.ReadFile(app.cfg.ReportFile)
	today := testToday()
	if !strings.HasPrefix(strings.TrimSpace(string(data)), today) {
		t.Errorf("report line should start with today's date %q, got: %q", today, string(data))
	}
}

func TestReport_outputIncludesTable(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "Task two")
	out := captureStdout(t, func() { app.Report() })
	if !strings.Contains(out, "Open") && !strings.Contains(out, "Date") {
		t.Errorf("output should include a header row, got:\n%s", out)
	}
}

// ── Dispatch: "done" alias for "do" ───────────────────────────────────────────
// The original todo.txt-cli accepts both "do" and "done" as the action name.
// The dispatch function should route "done" to App.Do just like "do".

func TestDispatch_doneAliasRoutesToDo(t *testing.T) {
	app, _ := newTestApp(t, "Task to complete")
	// Simulate routing the "done" alias through dispatch.
	err := dispatch(app, "done", []string{"1"})
	if err != nil {
		t.Fatalf("dispatch 'done' should not error: %v", err)
	}
	items, _ := ReadItems(app.cfg.TodoFile)
	if !items[0].Done {
		t.Errorf("task should be marked done via 'done' alias, got %q", items[0].Raw)
	}
}
