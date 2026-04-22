package main

// ANSI color codes — names match the original todo.cfg color map.
const (
	Black       = "\033[0;30m"
	Red         = "\033[0;31m"
	Green       = "\033[0;32m"
	Brown       = "\033[0;33m"
	Blue        = "\033[0;34m"
	Purple      = "\033[0;35m"
	Cyan        = "\033[0;36m"
	LightGrey   = "\033[0;37m"
	DarkGrey    = "\033[1;30m"
	LightRed    = "\033[1;31m"
	LightGreen  = "\033[1;32m"
	Yellow      = "\033[1;33m"
	LightBlue   = "\033[1;34m"
	LightPurple = "\033[1;35m"
	LightCyan   = "\033[1;36m"
	White       = "\033[1;37m"
	Default     = "\033[0m"
	None        = "" // disables coloring
)

// NamedColors maps the config-file color names (BLACK, YELLOW, …) to ANSI codes.
// These are seeded into the variable expansion context so that config lines like
// `export PRI_A=$YELLOW` resolve correctly.
var NamedColors = map[string]string{
	"BLACK":        Black,
	"RED":          Red,
	"GREEN":        Green,
	"BROWN":        Brown,
	"BLUE":         Blue,
	"PURPLE":       Purple,
	"CYAN":         Cyan,
	"LIGHT_GREY":   LightGrey,
	"DARK_GREY":    DarkGrey,
	"LIGHT_RED":    LightRed,
	"LIGHT_GREEN":  LightGreen,
	"YELLOW":       Yellow,
	"LIGHT_BLUE":   LightBlue,
	"LIGHT_PURPLE": LightPurple,
	"LIGHT_CYAN":   LightCyan,
	"WHITE":        White,
	"DEFAULT":      Default,
	"NONE":         None,
}

// ColorScheme holds the active display colors, mirroring the PRI_* / COLOR_*
// variables from todo.cfg.  Empty string means "no coloring for this element".
type ColorScheme struct {
	PriA    string // PRI_A  — default: Yellow
	PriB    string // PRI_B  — default: Green
	PriC    string // PRI_C  — default: LightBlue
	PriX    string // PRI_X  — default for any other priority: White
	Done    string // COLOR_DONE    — default: LightGrey
	Project string // COLOR_PROJECT — default: "" (inherit priority color)
	Context string // COLOR_CONTEXT — default: "" (inherit priority color)
	Date    string // COLOR_DATE    — default: "" (inherit priority color)
	Number  string // COLOR_NUMBER  — default: "" (inherit priority color)
	Meta    string // COLOR_META    — default: "" (inherit priority color)
}

// DefaultColorScheme returns the defaults from the original todo.txt-cli.
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		PriA: Yellow,
		PriB: Green,
		PriC: LightBlue,
		PriX: White,
		Done: LightGrey,
	}
}

// applyColor wraps s with an ANSI code and a reset, unless code is empty.
func applyColor(s, code string) string {
	if code == "" {
		return s
	}
	return code + s + Default
}
