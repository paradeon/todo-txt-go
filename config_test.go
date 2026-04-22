package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ── stripQuotes ──────────────────────────────────────────────────────────────

func TestStripQuotes(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{`hello`, "hello"},
		{`""`, ""},
		{`"it's"`, "it's"},
		{`'say "hi"'`, `say "hi"`},
		{`"unclosed`, `"unclosed`},
	}
	for _, c := range cases {
		got := stripQuotes(c.in)
		if got != c.want {
			t.Errorf("stripQuotes(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── unescapeANSI ─────────────────────────────────────────────────────────────

func TestUnescapeANSI(t *testing.T) {
	// \033 → ESC (0x1b)
	got := unescapeANSI(`\033[1;33m`)
	want := "\x1b[1;33m"
	if got != want {
		t.Errorf("unescapeANSI(\\033): got %q, want %q", got, want)
	}

	// \\033 → ESC (double-backslash variant from todo.cfg comments)
	got = unescapeANSI(`\\033[0;32m`)
	want = "\x1b[0;32m"
	if got != want {
		t.Errorf("unescapeANSI(\\\\033): got %q, want %q", got, want)
	}

	// \e → ESC
	got = unescapeANSI(`\e[0m`)
	want = "\x1b[0m"
	if got != want {
		t.Errorf("unescapeANSI(\\e): got %q, want %q", got, want)
	}

	// Already an ESC byte — leave alone.
	got = unescapeANSI("\x1b[0m")
	if got != "\x1b[0m" {
		t.Errorf("unescapeANSI(ESC) altered value: %q", got)
	}

	// No escape sequences — unchanged.
	got = unescapeANSI("plain text")
	if got != "plain text" {
		t.Errorf("unescapeANSI(plain): got %q", got)
	}
}

// ── expandShellVars ──────────────────────────────────────────────────────────

func TestExpandShellVars_simple(t *testing.T) {
	vars := map[string]string{"FOO": "bar"}
	got := expandShellVars("$FOO", vars)
	if got != "bar" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_braces(t *testing.T) {
	vars := map[string]string{"FOO": "bar"}
	got := expandShellVars("${FOO}", vars)
	if got != "bar" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_defaultUsed(t *testing.T) {
	vars := map[string]string{}
	got := expandShellVars("${MISSING:-fallback}", vars)
	if got != "fallback" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_defaultNotUsed(t *testing.T) {
	vars := map[string]string{"HOME": "/home/user"}
	got := expandShellVars("${HOME:-/tmp}", vars)
	if got != "/home/user" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_defaultIsVar(t *testing.T) {
	// ${MISSING:-$FALLBACK} — default is itself a variable reference.
	vars := map[string]string{"FALLBACK": "C:\\Users\\user"}
	got := expandShellVars("${__NO_SUCH_VAR__:-$FALLBACK}", vars)
	if got != "C:\\Users\\user" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_inPath(t *testing.T) {
	vars := map[string]string{"TODO_DIR": "/tmp/todo"}
	got := expandShellVars("$TODO_DIR/todo.txt", vars)
	if got != "/tmp/todo/todo.txt" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_noExpansion(t *testing.T) {
	vars := map[string]string{}
	got := expandShellVars("no dollar signs here", vars)
	if got != "no dollar signs here" { t.Errorf("got %q", got) }
}

func TestExpandShellVars_namedColors(t *testing.T) {
	// NamedColors are seeded into LoadConfig — verify key ones resolve correctly.
	if NamedColors["YELLOW"] != Yellow { t.Error("YELLOW mismatch") }
	if NamedColors["LIGHT_BLUE"] != LightBlue { t.Error("LIGHT_BLUE mismatch") }
	if NamedColors["NONE"] != "" { t.Error("NONE should be empty string") }

	vars := map[string]string{"YELLOW": Yellow}
	got := expandShellVars("$YELLOW", vars)
	if got != Yellow { t.Errorf("got %q, want %q", got, Yellow) }
}

// ── LoadConfig ───────────────────────────────────────────────────────────────

func writeTmpConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "todo*.cfg")
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString(content); err != nil { t.Fatal(err) }
	return f.Name()
}

func TestLoadConfig_filePaths(t *testing.T) {
	path := writeTmpConfig(t, `
export TODO_DIR=/tmp/mytodo
export TODO_FILE="$TODO_DIR/todo.txt"
export DONE_FILE="$TODO_DIR/done.txt"
export REPORT_FILE="$TODO_DIR/report.txt"
`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.TodoDir != "/tmp/mytodo" { t.Errorf("TodoDir: %q", cfg.TodoDir) }
	if cfg.TodoFile != "/tmp/mytodo/todo.txt" { t.Errorf("TodoFile: %q", cfg.TodoFile) }
	if cfg.DoneFile != "/tmp/mytodo/done.txt" { t.Errorf("DoneFile: %q", cfg.DoneFile) }
	if cfg.ReportFile != "/tmp/mytodo/report.txt" { t.Errorf("ReportFile: %q", cfg.ReportFile) }
}

func TestLoadConfig_todoDirSetsDefaults(t *testing.T) {
	// Setting TODO_DIR should auto-update the three derived paths.
	path := writeTmpConfig(t, "export TODO_DIR=/tmp/auto\n")
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.TodoFile != "/tmp/auto/todo.txt" { t.Errorf("TodoFile: %q", cfg.TodoFile) }
	if cfg.DoneFile != "/tmp/auto/done.txt" { t.Errorf("DoneFile: %q", cfg.DoneFile) }
}

func TestLoadConfig_priorityColors(t *testing.T) {
	path := writeTmpConfig(t, `
export PRI_A=$YELLOW
export PRI_B=$GREEN
export PRI_C=$LIGHT_BLUE
export PRI_X=$WHITE
`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.Colors.PriA != Yellow { t.Errorf("PriA: %q", cfg.Colors.PriA) }
	if cfg.Colors.PriB != Green { t.Errorf("PriB: %q", cfg.Colors.PriB) }
	if cfg.Colors.PriC != LightBlue { t.Errorf("PriC: %q", cfg.Colors.PriC) }
	if cfg.Colors.PriX != White { t.Errorf("PriX: %q", cfg.Colors.PriX) }
}

func TestLoadConfig_tokenColors(t *testing.T) {
	path := writeTmpConfig(t, `
export COLOR_DONE=$LIGHT_GREY
export COLOR_PROJECT=$RED
export COLOR_CONTEXT=$LIGHT_CYAN
export COLOR_DATE=$LIGHT_BLUE
export COLOR_NUMBER=$DARK_GREY
export COLOR_META=$CYAN
`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.Colors.Done != LightGrey { t.Errorf("Done: %q", cfg.Colors.Done) }
	if cfg.Colors.Project != Red { t.Errorf("Project: %q", cfg.Colors.Project) }
	if cfg.Colors.Context != LightCyan { t.Errorf("Context: %q", cfg.Colors.Context) }
	if cfg.Colors.Date != LightBlue { t.Errorf("Date: %q", cfg.Colors.Date) }
	if cfg.Colors.Number != DarkGrey { t.Errorf("Number: %q", cfg.Colors.Number) }
	if cfg.Colors.Meta != Cyan { t.Errorf("Meta: %q", cfg.Colors.Meta) }
}

func TestLoadConfig_rawANSIEscapeInColor(t *testing.T) {
	// Users may write raw \033 codes instead of named variables.
	path := writeTmpConfig(t, `export PRI_A='\033[0;35m'`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.Colors.PriA != "\x1b[0;35m" {
		t.Errorf("PriA with raw escape: %q", cfg.Colors.PriA)
	}
}

func TestLoadConfig_noneDisablesColor(t *testing.T) {
	path := writeTmpConfig(t, `export PRI_A=$NONE`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.Colors.PriA != "" {
		t.Errorf("PriA with $NONE should be empty, got %q", cfg.Colors.PriA)
	}
}

func TestLoadConfig_todotxtVerbose(t *testing.T) {
	p0 := writeTmpConfig(t, "export TODOTXT_VERBOSE=0\n")
	p2 := writeTmpConfig(t, "export TODOTXT_VERBOSE=2\n")
	c0, _ := LoadConfig(p0)
	c2, _ := LoadConfig(p2)
	if c0.Verbose { t.Error("VERBOSE=0 should be false") }
	if !c2.Verbose { t.Error("VERBOSE=2 should be true") }
}

func TestLoadConfig_defaultAction(t *testing.T) {
	path := writeTmpConfig(t, `export TODOTXT_DEFAULT_ACTION=ls`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.DefaultAction != "ls" { t.Errorf("DefaultAction: %q", cfg.DefaultAction) }
}

func TestLoadConfig_commentsAndBlanksIgnored(t *testing.T) {
	path := writeTmpConfig(t, `
# This is a comment
  # Indented comment

export TODO_DIR=/tmp/commenttest

# Another comment
`)
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.TodoDir != "/tmp/commenttest" { t.Errorf("TodoDir: %q", cfg.TodoDir) }
}

func TestLoadConfig_homeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := writeTmpConfig(t, "export TODO_DIR=${HOME:-/tmp}/todohome\n")
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	want := filepath.Join(home, "todohome")
	if cfg.TodoDir != want { t.Errorf("TodoDir: got %q, want %q", cfg.TodoDir, want) }
}

func TestLoadConfig_bareAssignmentWithoutExport(t *testing.T) {
	path := writeTmpConfig(t, "TODO_DIR=/tmp/bare\n")
	cfg, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if cfg.TodoDir != "/tmp/bare" { t.Errorf("TodoDir: %q", cfg.TodoDir) }
}

func TestLoadConfig_missingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/todo.cfg")
	if err == nil { t.Error("expected error for missing file") }
}

func TestDefaultConfigPaths_primaryIsXDG(t *testing.T) {
	home, _ := os.UserHomeDir()
	paths := DefaultConfigPaths()
	if len(paths) == 0 { t.Fatal("no paths returned") }
	want := filepath.Join(home, ".config", "todo", "todo.cfg")
	if paths[0] != want {
		t.Errorf("primary path: got %q, want %q", paths[0], want)
	}
}

func TestDefaultConfigPaths_includesLegacy(t *testing.T) {
	home, _ := os.UserHomeDir()
	paths := DefaultConfigPaths()
	legacy := filepath.Join(home, ".todo.cfg")
	found := false
	for _, p := range paths {
		if p == legacy {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("legacy path %q not in %v", legacy, paths)
	}
}
