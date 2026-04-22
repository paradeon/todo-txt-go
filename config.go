package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all runtime configuration.
type Config struct {
	TodoDir       string
	TodoFile      string
	DoneFile      string
	ReportFile    string
	DateOnAdd     bool   // -t / TODOTXT_DATE_ON_ADD
	AutoArchive   bool   // TODOTXT_AUTO_ARCHIVE (default true)
	Force         bool   // -f
	Plain         bool   // -p / -n
	Verbose       bool   // -v / TODOTXT_VERBOSE=2
	DefaultAction string // TODOTXT_DEFAULT_ACTION
	Colors        ColorScheme
}

// DefaultConfig builds a Config from environment variables with sensible fallbacks.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	todoDir := filepath.Join(home, "todo")
	if d := os.Getenv("TODO_DIR"); d != "" {
		todoDir = d
	}
	cfg := Config{
		TodoDir:     todoDir,
		TodoFile:    filepath.Join(todoDir, "todo.txt"),
		DoneFile:    filepath.Join(todoDir, "done.txt"),
		ReportFile:  filepath.Join(todoDir, "report.txt"),
		AutoArchive: true,
		Colors:      DefaultColorScheme(),
	}
	if f := os.Getenv("TODO_FILE"); f != "" {
		cfg.TodoFile = f
	}
	if f := os.Getenv("DONE_FILE"); f != "" {
		cfg.DoneFile = f
	}
	if f := os.Getenv("REPORT_FILE"); f != "" {
		cfg.ReportFile = f
	}
	return cfg
}

// DefaultConfigPaths returns config file search locations in priority order.
// ~/.config/todo/todo.cfg is checked first (XDG-style), then legacy paths.
func DefaultConfigPaths() []string {
	home, _ := os.UserHomeDir()
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	return []string{
		filepath.Join(xdgConfig, "todo", "todo.cfg"), // primary default
		filepath.Join(home, ".todo.cfg"),              // original todo.txt-cli default
		filepath.Join(home, ".todo", "config"),
		filepath.Join(home, "todo", "config"),
	}
}

// LoadConfig reads a todo.cfg-format file (shell export VAR=value syntax).
//
// Supports:
//   - `export VAR=value` and bare `VAR=value` assignments
//   - Double- and single-quoted values
//   - Variable expansion: $VAR  ${VAR}  ${VAR:-default}
//   - All color-map names (YELLOW, LIGHT_BLUE, …) as pre-defined variables
//   - ANSI escape sequences written as \033 or \e
//   - All TODO_* / TODOTXT_* / PRI_* / COLOR_* knobs from the original
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	// Seed the expansion context with built-in color names + current environment.
	vars := make(map[string]string, len(NamedColors)+10)
	for k, v := range NamedColors {
		vars[k] = v
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		rawVal := strings.TrimSpace(line[eqIdx+1:])

		// Strip one layer of surrounding quotes before expanding.
		rawVal = stripQuotes(rawVal)

		// Expand $VAR / ${VAR} / ${VAR:-default} using accumulated vars + env.
		val := expandShellVars(rawVal, vars)

		// Convert \033 / \e escape sequences to actual ESC bytes.
		val = unescapeANSI(val)

		// Accumulate so later lines can reference this variable.
		vars[key] = val

		// Apply to config struct.
		switch key {
		case "TODO_DIR":
			cfg.TodoDir = val
			cfg.TodoFile = filepath.Join(val, "todo.txt")
			cfg.DoneFile = filepath.Join(val, "done.txt")
			cfg.ReportFile = filepath.Join(val, "report.txt")
		case "TODO_FILE":
			cfg.TodoFile = val
		case "DONE_FILE":
			cfg.DoneFile = val
		case "REPORT_FILE":
			cfg.ReportFile = val
		case "TODOTXT_VERBOSE":
			cfg.Verbose = val == "2"
		case "TODOTXT_DATE_ON_ADD":
			cfg.DateOnAdd = val == "1"
		case "TODOTXT_AUTO_ARCHIVE":
			cfg.AutoArchive = val != "0"
		case "TODOTXT_DEFAULT_ACTION":
			cfg.DefaultAction = val
		case "PRI_A":
			cfg.Colors.PriA = val
		case "PRI_B":
			cfg.Colors.PriB = val
		case "PRI_C":
			cfg.Colors.PriC = val
		case "PRI_X":
			cfg.Colors.PriX = val
		case "COLOR_DONE":
			cfg.Colors.Done = val
		case "COLOR_PROJECT":
			cfg.Colors.Project = val
		case "COLOR_CONTEXT":
			cfg.Colors.Context = val
		case "COLOR_DATE":
			cfg.Colors.Date = val
		case "COLOR_NUMBER":
			cfg.Colors.Number = val
		case "COLOR_META":
			cfg.Colors.Meta = val
		}
	}
	return cfg, scanner.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

// stripQuotes removes one layer of matching surrounding double or single quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// expandShellVars expands $VAR, ${VAR}, and ${VAR:-default} in s.
// It consults vars first (accumulated config vars + built-in color names),
// then falls back to os.Getenv.
func expandShellVars(s string, vars map[string]string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		i++ // consume '$'
		if i >= len(s) {
			b.WriteByte('$')
			break
		}

		if s[i] == '{' {
			// ${VAR} or ${VAR:-default}
			i++ // consume '{'
			end := strings.IndexByte(s[i:], '}')
			if end < 0 {
				b.WriteString("${")
				b.WriteString(s[i:])
				break
			}
			expr := s[i : i+end]
			i += end + 1 // consume up to and including '}'

			if sep := strings.Index(expr, ":-"); sep >= 0 {
				name := expr[:sep]
				def := expr[sep+2:]
				if v := shellLookup(name, vars); v != "" {
					b.WriteString(v)
				} else {
					b.WriteString(expandShellVars(def, vars))
				}
			} else {
				b.WriteString(shellLookup(expr, vars))
			}
		} else {
			// $IDENTIFIER
			j := i
			for j < len(s) && isIdentChar(s[j]) {
				j++
			}
			if j == i {
				b.WriteByte('$')
				continue
			}
			b.WriteString(shellLookup(s[i:j], vars))
			i = j
		}
	}
	return b.String()
}

func shellLookup(name string, vars map[string]string) string {
	if v, ok := vars[name]; ok {
		return v
	}
	return os.Getenv(name)
}

func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// unescapeANSI converts \\033, \033, and \e to the ESC byte (0x1b).
func unescapeANSI(s string) string {
	// Handle double-backslash variant (e.g. from config comments: \\033[1;33m).
	s = strings.ReplaceAll(s, "\\\\033", "\x1b")
	// Handle single-backslash octal (\033).
	s = strings.ReplaceAll(s, "\\033", "\x1b")
	// Handle \e shorthand.
	s = strings.ReplaceAll(s, "\\e", "\x1b")
	return s
}
