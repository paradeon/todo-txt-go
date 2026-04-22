package main

// Tests for the help / shorthelp actions, verified against original todo.txt-cli behavior.
//
// Behavioral differences from the original todo.txt-cli noted here:
//   - Original paginates full help through $PAGER and writes to stdout.
//     Go writes to stderr directly (no pager). Stdout-capture tests for the
//     no-args case therefore capture stderr instead.
//   - "help done" was missing until the "done" alias was added to the help map.

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr redirects os.Stderr for the duration of fn and returns the output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	out, _ := io.ReadAll(r)
	r.Close()
	return string(out)
}

// ── help (no args) ────────────────────────────────────────────────────────────

func TestHelp_noArgs_showsAllActions(t *testing.T) {
	out := captureStderr(t, func() { printHelp(nil) })
	for _, action := range []string{"add", "addm", "addto", "append", "archive",
		"deduplicate", "del", "depri", "do", "list", "listall", "listcon",
		"listfile", "listpri", "listproj", "move", "prepend", "pri", "replace", "report"} {
		if !strings.Contains(out, action) {
			t.Errorf("help should mention action %q, not found in output:\n%s", action, out)
		}
	}
}

func TestHelp_noArgs_showsOptions(t *testing.T) {
	out := captureStderr(t, func() { printHelp(nil) })
	for _, flag := range []string{"-f", "-p", "-t", "-v", "-V", "-d"} {
		if !strings.Contains(out, flag) {
			t.Errorf("help should document flag %q:\n%s", flag, out)
		}
	}
}

func TestHelp_noArgs_showsEnvironmentVars(t *testing.T) {
	out := captureStderr(t, func() { printHelp(nil) })
	for _, env := range []string{"TODO_DIR", "TODO_FILE", "DONE_FILE", "REPORT_FILE"} {
		if !strings.Contains(out, env) {
			t.Errorf("help should document env var %q:\n%s", env, out)
		}
	}
}

func TestHelp_noArgs_mentionsDoneAlias(t *testing.T) {
	out := captureStderr(t, func() { printHelp(nil) })
	if !strings.Contains(out, "do") {
		t.Errorf("help should mention the do action:\n%s", out)
	}
}

func TestHelp_noArgs_mentionsAliases(t *testing.T) {
	out := captureStderr(t, func() { printHelp(nil) })
	// Spot-check a few aliases that must appear.
	for _, alias := range []string{"ls", "lsa", "lsc", "lsp", "lsprj", "lf", "rm", "dp", "app", "prep", "mv"} {
		if !strings.Contains(out, alias) {
			t.Errorf("help should mention alias %q:\n%s", alias, out)
		}
	}
}

// ── help ACTION ───────────────────────────────────────────────────────────────

func TestHelp_specificAction_add(t *testing.T) {
	out := captureStdout(t, func() { printHelp([]string{"add"}) })
	if !strings.Contains(out, "add") {
		t.Errorf("help add should describe the add action:\n%s", out)
	}
}

func TestHelp_specificAction_caseInsensitive(t *testing.T) {
	// The original accepts the action name in any case.
	lower := captureStdout(t, func() { printHelp([]string{"add"}) })
	upper := captureStdout(t, func() { printHelp([]string{"ADD"}) })
	if lower != upper {
		t.Errorf("help should be case-insensitive:\nlower=%q\nupper=%q", lower, upper)
	}
}

func TestHelp_specificAction_unknownAction(t *testing.T) {
	out := captureStderr(t, func() {
		captureStdout(t, func() { printHelp([]string{"nosuchaction"}) })
	})
	if !strings.Contains(out, "No help") && !strings.Contains(out, "unknown") {
		t.Errorf("unknown action should produce an error message, got:\n%s", out)
	}
}

func TestHelp_specificAction_aliasReturnsHelp(t *testing.T) {
	// Aliases should resolve to their help text.
	cases := []string{"a", "app", "rm", "dp", "ls", "lsa", "lsc", "lf", "lsp", "lsprj", "mv", "prep", "p"}
	for _, alias := range cases {
		out := captureStdout(t, func() { printHelp([]string{alias}) })
		if strings.TrimSpace(out) == "" {
			t.Errorf("help %q should return text, got empty output", alias)
		}
	}
}

func TestHelp_specificAction_doneAlias(t *testing.T) {
	// "done" is an alias for "do" — help should recognise it.
	out := captureStdout(t, func() { printHelp([]string{"done"}) })
	if strings.TrimSpace(out) == "" {
		t.Errorf("help done should return text (alias for do), got empty output")
	}
}

func TestHelp_multipleActions(t *testing.T) {
	out := captureStdout(t, func() { printHelp([]string{"add", "del", "do"}) })
	for _, action := range []string{"add", "del", "do"} {
		if !strings.Contains(out, action) {
			t.Errorf("help with multiple actions should include %q:\n%s", action, out)
		}
	}
}

// ── shorthelp ─────────────────────────────────────────────────────────────────

func TestShortHelp_listsEveryAction(t *testing.T) {
	out := captureStdout(t, func() { shortHelp() })
	for _, action := range []string{"add", "addm", "addto", "append", "archive",
		"deduplicate", "del", "depri", "do", "list", "listall", "listcon",
		"listfile", "listpri", "listproj", "move", "prepend", "pri", "replace", "report", "shorthelp"} {
		if !strings.Contains(out, action) {
			t.Errorf("shorthelp should mention %q:\n%s", action, out)
		}
	}
}

func TestShortHelp_isCompact(t *testing.T) {
	// Each action should be on its own line (one-liner format).
	out := captureStdout(t, func() { shortHelp() })
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 15 {
		t.Errorf("shorthelp should have at least 15 lines, got %d", len(lines))
	}
	// Every line should be short (< 80 chars).
	for i, line := range lines {
		if len(line) > 80 {
			t.Errorf("shorthelp line %d is too long (%d chars): %q", i+1, len(line), line)
		}
	}
}

func TestShortHelp_includesAliases(t *testing.T) {
	out := captureStdout(t, func() { shortHelp() })
	// Key aliases should appear inline.
	for _, alias := range []string{"ls", "lsa", "lsc", "rm", "dp", "app", "mv"} {
		if !strings.Contains(out, alias) {
			t.Errorf("shorthelp should mention alias %q:\n%s", alias, out)
		}
	}
}

func TestShortHelp_writesToStdout(t *testing.T) {
	// shorthelp should write to stdout (same as original), not stderr.
	stdout := captureStdout(t, func() { shortHelp() })
	if strings.TrimSpace(stdout) == "" {
		t.Error("shorthelp should write to stdout")
	}
}

func TestShortHelp_noBlankLines(t *testing.T) {
	out := captureStdout(t, func() { shortHelp() })
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			t.Errorf("shorthelp should have no blank lines, found one at line %d", i+1)
		}
	}
}
