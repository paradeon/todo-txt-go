package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			break
		}
	}
	r.Close()
	return buf.String()
}

// newTestApp creates an App wired up to temp files.
func newTestApp(t *testing.T, todoLines ...string) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	todoPath := filepath.Join(dir, "todo.txt")
	donePath := filepath.Join(dir, "done.txt")
	reportPath := filepath.Join(dir, "report.txt")

	f, err := os.Create(todoPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range todoLines {
		f.WriteString(l + "\n")
	}
	f.Close()

	cfg := Config{
		TodoDir:     dir,
		TodoFile:    todoPath,
		DoneFile:    donePath,
		ReportFile:  reportPath,
		Force:       true,
		AutoArchive: false,
	}
	return NewApp(cfg), dir
}

// ── Add ───────────────────────────────────────────────────────────────────────

func TestAdd_basic(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		if err := app.Add([]string{"Call Mom"}); err != nil {
			t.Fatal(err)
		}
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 1 || items[0].Raw != "Call Mom" {
		t.Errorf("unexpected state: %v", items)
	}
}

func TestAdd_noArgs(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.Add(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestAdd_dateOnAdd(t *testing.T) {
	app, _ := newTestApp(t)
	app.cfg.DateOnAdd = true
	captureStdout(t, func() {
		app.Add([]string{"Buy milk"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	today := app.today()
	if !strings.HasPrefix(items[0].Raw, today) {
		t.Errorf("expected date prefix, got %q", items[0].Raw)
	}
}

func TestAdd_dateOnAdd_withPriority(t *testing.T) {
	app, _ := newTestApp(t)
	app.cfg.DateOnAdd = true
	captureStdout(t, func() {
		app.Add([]string{"(A)", "Fix bug"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	today := app.today()
	want := "(A) " + today + " Fix bug"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

func TestAdd_multipleItems(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Add([]string{"Task one"})
		app.Add([]string{"Task two"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

// ── Addm ──────────────────────────────────────────────────────────────────────

func TestAddm_multiline(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Addm([]string{"Task one\nTask two\nTask three"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestAddm_skipsBlanks(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		app.Addm([]string{"Task one\n\nTask two"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

// ── Append / Prepend ──────────────────────────────────────────────────────────

func TestAppend_addsText(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Append([]string{"1", "+family"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Call Mom +family" {
		t.Errorf("got %q", items[0].Raw)
	}
}

func TestPrepend_insertsBeforeDescription(t *testing.T) {
	app, _ := newTestApp(t, "(A) 2026-04-01 Call Mom")
	captureStdout(t, func() {
		app.Prepend([]string{"1", "URGENT:"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	want := "(A) 2026-04-01 URGENT: Call Mom"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

func TestPrepend_noPriorityNoDate(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Prepend([]string{"1", "Really:"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "Really: Call Mom" {
		t.Errorf("got %q", items[0].Raw)
	}
}

// ── Del ───────────────────────────────────────────────────────────────────────

func TestDel_deletesTask(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "Task two", "Task three")
	captureStdout(t, func() {
		app.Del([]string{"2"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	// Line 2 should be blank (tombstone), others intact.
	if items[1].Raw != "" {
		t.Errorf("deleted item should be blank, got %q", items[1].Raw)
	}
	if items[0].Raw != "Task one" || items[2].Raw != "Task three" {
		t.Error("other items should be untouched")
	}
}

func TestDel_removesTerm(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom +family @phone")
	captureStdout(t, func() {
		app.Del([]string{"1", "+family"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if strings.Contains(items[0].Raw, "+family") {
		t.Errorf("term should be removed, got %q", items[0].Raw)
	}
}

func TestDel_invalidNumber(t *testing.T) {
	app, _ := newTestApp(t, "Task one")
	if err := app.Del([]string{"99"}); err == nil {
		t.Error("expected error for out-of-range number")
	}
}

// ── Pri / Depri ───────────────────────────────────────────────────────────────

func TestPri_setsPriority(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Pri([]string{"1", "A"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "A" {
		t.Errorf("priority: got %q", items[0].Priority)
	}
}

func TestPri_lowercaseNormalized(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Pri([]string{"1", "b"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "B" {
		t.Errorf("priority: got %q", items[0].Priority)
	}
}

func TestPri_invalidPriority(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	if err := app.Pri([]string{"1", "1"}); err == nil {
		t.Error("expected error for non-letter priority")
	}
}

func TestDepri_removesPriority(t *testing.T) {
	app, _ := newTestApp(t, "(A) Call Mom", "(B) Buy milk")
	captureStdout(t, func() {
		app.Depri([]string{"1", "2"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Priority != "" || items[1].Priority != "" {
		t.Error("priorities should be removed")
	}
}

// ── Do ────────────────────────────────────────────────────────────────────────

func TestDo_marksTaskDone(t *testing.T) {
	app, _ := newTestApp(t, "(A) Call Mom", "Buy milk")
	captureStdout(t, func() {
		app.Do([]string{"1"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if !items[0].Done {
		t.Error("item 1 should be done")
	}
	today := app.today()
	if items[0].CompletionDate != today {
		t.Errorf("completion date: got %q, want %q", items[0].CompletionDate, today)
	}
	// Priority must be stripped.
	if strings.Contains(items[0].Raw, "(A)") {
		t.Errorf("priority should be stripped on do, got %q", items[0].Raw)
	}
	if items[1].Done {
		t.Error("item 2 should not be affected")
	}
}

func TestDo_bakContainsDoneMarkedContent(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom")
	captureStdout(t, func() {
		app.Do([]string{"1"})
	})
	bak, err := ReadItems(app.cfg.TodoFile + ".bak")
	if err != nil {
		t.Fatal("todo.txt.bak should be created after do")
	}
	if len(bak) != 1 || !bak[0].Done {
		t.Errorf("todo.txt.bak should contain the done-marked item, got %v", bak)
	}
}

func TestDo_stripsPriorityBeforeDate(t *testing.T) {
	app, _ := newTestApp(t, "(B) 2026-04-01 Buy milk")
	captureStdout(t, func() {
		app.Do([]string{"1"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	want := "x " + app.today() + " 2026-04-01 Buy milk"
	if items[0].Raw != want {
		t.Errorf("got %q, want %q", items[0].Raw, want)
	}
}

func TestDo_alreadyDone(t *testing.T) {
	app, _ := newTestApp(t, "x 2026-04-20 Already done")
	out := captureStdout(t, func() {
		app.Do([]string{"1"})
	})
	if !strings.Contains(out, "already done") {
		t.Errorf("expected 'already done' message, got: %q", out)
	}
}

func TestDo_multipleTasks(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "Task two", "Task three")
	captureStdout(t, func() {
		app.Do([]string{"1", "3"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if !items[0].Done || items[1].Done || !items[2].Done {
		t.Error("items 1 and 3 should be done, 2 should not")
	}
}

func TestDo_autoArchiveMovesDoneItemToDoneFile(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom", "Buy milk")
	app.cfg.AutoArchive = true
	captureStdout(t, func() {
		app.Do([]string{"1"})
	})
	todo, _ := ReadItems(app.cfg.TodoFile)
	done, _ := ReadItems(app.cfg.DoneFile)

	for _, item := range todo {
		if item.Done {
			t.Errorf("done item should not remain in todo.txt after auto-archive: %q", item.Raw)
		}
	}
	if len(done) != 1 || !done[0].Done {
		t.Errorf("done item should appear in done.txt: %v", done)
	}
}

// ── Replace ───────────────────────────────────────────────────────────────────

func TestReplace_replacesEntireLine(t *testing.T) {
	app, _ := newTestApp(t, "(A) Call Mom", "Buy milk")
	captureStdout(t, func() {
		app.Replace([]string{"1", "(B)", "Call Dad"})
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if items[0].Raw != "(B) Call Dad" {
		t.Errorf("got %q", items[0].Raw)
	}
	if items[1].Raw != "Buy milk" {
		t.Error("item 2 should be untouched")
	}
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestArchive_movesDoneTasks(t *testing.T) {
	app, _ := newTestApp(t, "(A) Active task", "x 2026-04-22 Done task", "Another active task")
	captureStdout(t, func() {
		app.Archive()
	})
	todo, _ := ReadItems(app.cfg.TodoFile)
	done, _ := ReadItems(app.cfg.DoneFile)

	// Only active tasks remain in todo.
	for _, item := range todo {
		if item.Done {
			t.Errorf("done item found in todo after archive: %q", item.Raw)
		}
	}
	// Done task appears in done file.
	if len(done) != 1 || !done[0].Done {
		t.Errorf("done file should have 1 done task, got: %v", done)
	}
}

func TestArchive_stripsTrailingBlanks(t *testing.T) {
	app, _ := newTestApp(t, "Active task", "x 2026-04-22 Done task")
	captureStdout(t, func() {
		app.Archive()
	})
	todo, _ := ReadItems(app.cfg.TodoFile)
	if len(todo) > 0 && todo[len(todo)-1].Raw == "" {
		t.Error("trailing blank lines should be stripped after archive")
	}
}

// ── Deduplicate ───────────────────────────────────────────────────────────────

func TestDeduplicate_removesDuplicates(t *testing.T) {
	app, _ := newTestApp(t, "Call Mom", "Buy milk", "Call Mom", "Buy milk")
	captureStdout(t, func() {
		app.Deduplicate()
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 2 {
		t.Errorf("expected 2 unique items, got %d", len(items))
	}
}

func TestDeduplicate_noDuplicates(t *testing.T) {
	app, _ := newTestApp(t, "Task A", "Task B", "Task C")
	captureStdout(t, func() {
		app.Deduplicate()
	})
	items, _ := ReadItems(app.cfg.TodoFile)
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

// ── Move ──────────────────────────────────────────────────────────────────────

func TestMove_movesTaskBetweenFiles(t *testing.T) {
	app, dir := newTestApp(t, "Task one", "Task two", "Task three")
	otherPath := filepath.Join(dir, "other.txt")

	captureStdout(t, func() {
		app.Move([]string{"2", "other.txt"})
	})

	todo, _ := ReadItems(app.cfg.TodoFile)
	other, _ := ReadItems(otherPath)

	// Task two should be gone from todo (blanked out).
	if todo[1].Raw != "" {
		t.Errorf("moved task should be blank in source, got %q", todo[1].Raw)
	}
	// Task two should appear in other file.
	if len(other) != 1 || other[0].Raw != "Task two" {
		t.Errorf("moved task not found in dest: %v", other)
	}
}

// ── Listall ───────────────────────────────────────────────────────────────────

func TestListall_doneItemsLineNumberZero(t *testing.T) {
	app, dir := newTestApp(t, "Buy milk", "Call Mom")
	// Write a done item to done.txt.
	donePath := filepath.Join(dir, "done.txt")
	os.WriteFile(donePath, []byte("x 2026-04-22 Done task\n"), 0644)

	var capturedItems []string
	out := captureStdout(t, func() {
		app.Listall(nil)
	})
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Done task") {
			capturedItems = append(capturedItems, line)
		}
	}
	if len(capturedItems) == 0 {
		t.Fatal("done task not found in listall output")
	}
	if !strings.HasPrefix(strings.TrimSpace(capturedItems[0]), "0 ") {
		t.Errorf("done item should have line number 0, got %q", capturedItems[0])
	}
}

func TestListall_summaryShowsPerFileCounts(t *testing.T) {
	app, dir := newTestApp(t, "Task one", "Task two")
	donePath := filepath.Join(dir, "done.txt")
	os.WriteFile(donePath, []byte("x 2026-04-22 Done one\nx 2026-04-22 Done two\n"), 0644)

	out := captureStdout(t, func() {
		app.Listall(nil)
	})
	if !strings.Contains(out, "TODO: 2 of 2 tasks shown") {
		t.Errorf("missing TODO count line in:\n%s", out)
	}
	if !strings.Contains(out, "DONE: 2 of 2 tasks shown") {
		t.Errorf("missing DONE count line in:\n%s", out)
	}
	if !strings.Contains(out, "total 4 of 4 tasks shown") {
		t.Errorf("missing total count line in:\n%s", out)
	}
}

func TestListall_sortedAlphabetically(t *testing.T) {
	app, _ := newTestApp(t, "Zucchini", "Apple", "Mango")

	out := captureStdout(t, func() {
		app.Listall(nil)
	})
	lines := []string{}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Apple") || strings.Contains(l, "Mango") || strings.Contains(l, "Zucchini") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 task lines, got %v", lines)
	}
	if !strings.Contains(lines[0], "Apple") || !strings.Contains(lines[1], "Mango") || !strings.Contains(lines[2], "Zucchini") {
		t.Errorf("not sorted alphabetically: %v", lines)
	}
}

func TestListall_includesBlanksWithNoTerms(t *testing.T) {
	app, _ := newTestApp(t, "Task one", "", "Task three")

	out := captureStdout(t, func() {
		app.Listall(nil)
	})
	// Blank line preserved: output should have a line with just a number.
	lineCount := 0
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(l) == "2" || l == "2 " || strings.HasSuffix(strings.TrimRight(l, " "), "2") && strings.Contains(l, " ") && !strings.Contains(l, "Task") {
			lineCount++
		}
	}
	// Simpler check: total output lines before "--" should include the blank.
	taskLines := 0
	for _, l := range strings.Split(out, "\n") {
		if l == "--" {
			break
		}
		taskLines++
	}
	if taskLines != 3 {
		t.Errorf("expected 3 lines (including blank) before --, got %d", taskLines)
	}
}

// ── isPrioritySpec ────────────────────────────────────────────────────────────

func TestIsPrioritySpec(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"A", true},
		{"Z", true},
		{"A-Z", true},
		{"A-C", true},
		{"a", false},   // lowercase
		{"1", false},   // digit
		{"AB", false},  // two letters without dash
		{"A-a", false}, // mixed case range
		{"Z-A", false}, // inverted range
		{"A-", false},  // incomplete range
		{"", false},
	}
	for _, c := range cases {
		got := isPrioritySpec(c.s)
		if got != c.want {
			t.Errorf("isPrioritySpec(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

// ── sortedKeys / containsAny ──────────────────────────────────────────────────

func TestSortedKeys(t *testing.T) {
	m := map[string]bool{"banana": true, "apple": true, "cherry": true}
	got := sortedKeys(m)
	want := []string{"apple", "banana", "cherry"}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("sortedKeys[%d] = %q, want %q", i, got[i], v)
		}
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("Hello World", []string{"world"}) {
		t.Error("case-insensitive match failed")
	}
	if !containsAny("foo bar baz", []string{"missing", "bar"}) {
		t.Error("second term should match")
	}
	if containsAny("hello", []string{"xyz", "abc"}) {
		t.Error("should not match")
	}
}
