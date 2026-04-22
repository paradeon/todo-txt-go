package main

// Targeted parity tests for the listfile / lf action.
//
// Behavioural differences from original todo.txt-cli that were fixed:
//   1. No-args: original lists all *.txt files in TODO_DIR; Go previously
//      returned an error — now fixed to enumerate files.
//   2. Summary line: original prints "N of M tasks shown"; Go previously
//      printed "N tasks shown" (missing the total) — now fixed.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── no-args: list *.txt files in TODO_DIR ────────────────────────────────────

func TestListfile_noArgs_printsHeader(t *testing.T) {
	app, _ := newTestApp(t)
	out := captureStdout(t, func() { app.Listfile(nil) })
	if !strings.Contains(out, "Files in the todo.txt directory:") {
		t.Errorf("no-args listfile should print header line, got:\n%s", out)
	}
}

func TestListfile_noArgs_onlyTxtFiles(t *testing.T) {
	app, dir := newTestApp(t)
	os.WriteFile(filepath.Join(dir, "done.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "report.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "config.cfg"), []byte(""), 0644)

	out := captureStdout(t, func() { app.Listfile(nil) })

	for _, name := range []string{"todo.txt", "done.txt", "report.txt"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in no-args output:\n%s", name, out)
		}
	}
	for _, name := range []string{"readme.md", "config.cfg"} {
		if strings.Contains(out, name) {
			t.Errorf("non-.txt file %q should not appear:\n%s", name, out)
		}
	}
}

func TestListfile_noArgs_emptyDir(t *testing.T) {
	app, _ := newTestApp(t) // only todo.txt is created by newTestApp
	out := captureStdout(t, func() { app.Listfile(nil) })
	if !strings.Contains(out, "todo.txt") {
		t.Errorf("todo.txt should appear even in a near-empty dir:\n%s", out)
	}
}

func TestListfile_noArgs_doesNotError(t *testing.T) {
	app, _ := newTestApp(t)
	captureStdout(t, func() {
		if err := app.Listfile(nil); err != nil {
			t.Errorf("no-args listfile should not return an error: %v", err)
		}
	})
}

// ── with SRC: display tasks from that file ────────────────────────────────────

func TestListfile_showsDoneTasksFromFile(t *testing.T) {
	// Unlike `list`, listfile shows everything in the file — including done items.
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "archive.txt")
	os.WriteFile(srcPath, []byte("x 2026-04-22 Completed task\nActive task\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"archive.txt"}) })
	if !strings.Contains(out, "Completed task") {
		t.Errorf("done tasks should appear in listfile output:\n%s", out)
	}
	if !strings.Contains(out, "Active task") {
		t.Errorf("active tasks should appear in listfile output:\n%s", out)
	}
}

func TestListfile_nonExistentFile_returnsError(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.Listfile([]string{"nosuchfile.txt"})
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestListfile_absolutePath(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "absolute.txt")
	os.WriteFile(srcPath, []byte("Task via abs path\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{srcPath}) })
	if !strings.Contains(out, "Task via abs path") {
		t.Errorf("absolute path should work:\n%s", out)
	}
}

func TestListfile_multipleTermsAndFilter(t *testing.T) {
	// Multiple TERM args are ANDed (same as list / listall).
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "work.txt")
	os.WriteFile(srcPath, []byte(
		"Meeting +work @office\nLunch +personal @office\nEmail +work @home\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"work.txt", "+work", "@office"}) })
	if !strings.Contains(out, "Meeting") {
		t.Errorf("AND-matched task should appear:\n%s", out)
	}
	if strings.Contains(out, "Lunch") || strings.Contains(out, "Email") {
		t.Errorf("non-AND-matched tasks should be excluded:\n%s", out)
	}
}

// ── summary line format ───────────────────────────────────────────────────────

func TestListfile_summaryFormat_noFilter(t *testing.T) {
	// "N of N tasks shown" when no filter is applied.
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	os.WriteFile(srcPath, []byte("Alpha\nBeta\nGamma\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"src.txt"}) })
	if !strings.Contains(out, "3 of 3 tasks shown") {
		t.Errorf("unfiltered summary should show '3 of 3 tasks shown':\n%s", out)
	}
}

func TestListfile_summaryFormat_withFilter(t *testing.T) {
	// "N of M tasks shown" when a filter reduces the count.
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	os.WriteFile(srcPath, []byte("Alpha @keep\nBeta @drop\nGamma @keep\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"src.txt", "@keep"}) })
	if !strings.Contains(out, "2 of 3 tasks shown") {
		t.Errorf("filtered summary should show '2 of 3 tasks shown':\n%s", out)
	}
}

func TestListfile_summaryFormat_mentionsFilename(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "mylist.txt")
	os.WriteFile(srcPath, []byte("Item\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"mylist.txt"}) })
	if !strings.Contains(out, "mylist.txt") {
		t.Errorf("summary should include the filename:\n%s", out)
	}
}

func TestListfile_summaryFormat_emptyFile(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "empty.txt")
	os.WriteFile(srcPath, []byte(""), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"empty.txt"}) })
	if !strings.Contains(out, "0 of 0 tasks shown") {
		t.Errorf("empty file summary should show '0 of 0 tasks shown':\n%s", out)
	}
}

// ── line numbers and ordering ─────────────────────────────────────────────────

func TestListfile_lineNumbersMatchFilePosition(t *testing.T) {
	app, dir := newTestApp(t)
	srcPath := filepath.Join(dir, "src.txt")
	os.WriteFile(srcPath, []byte("(C) Low\n(A) High\n(B) Mid\n"), 0644)

	out := captureStdout(t, func() { app.Listfile([]string{"src.txt"}) })
	// After priority sort: High (line 2), Mid (line 3), Low (line 1).
	posHigh := strings.Index(out, "High")
	posMid := strings.Index(out, "Mid")
	posLow := strings.Index(out, "Low")
	if posHigh > posMid || posMid > posLow {
		t.Errorf("items should be sorted by priority (A>B>C):\n%s", out)
	}
}
