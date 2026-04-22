package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ReadItems reads all items from a file.
// Blank / empty lines are preserved as items with Raw == "".
func ReadItems(path string) ([]Item, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var items []Item
	scanner := bufio.NewScanner(f)
	lineNum := 1
	for scanner.Scan() {
		items = append(items, ParseItem(scanner.Text(), lineNum))
		lineNum++
	}
	return items, scanner.Err()
}

// WriteItems writes items back to a file, one per line.
func WriteItems(path string, items []Item) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, item := range items {
		fmt.Fprintln(w, item.Raw)
	}
	return w.Flush()
}

// AppendItems appends items to an existing file (creates if absent).
func AppendItems(path string, items []Item) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, item := range items {
		fmt.Fprintln(w, item.Raw)
	}
	return w.Flush()
}

// FilterItems filters items by search terms (all terms must match, case-insensitive).
// Blank items are always excluded.
func FilterItems(items []Item, terms []string) []Item {
	var result []Item
	for _, item := range items {
		if item.Raw == "" {
			continue
		}
		match := true
		lower := strings.ToLower(item.Raw)
		for _, t := range terms {
			if !strings.Contains(lower, strings.ToLower(t)) {
				match = false
				break
			}
		}
		if match {
			result = append(result, item)
		}
	}
	return result
}

// SortByPriority sorts: prioritised tasks first (A < B < …), then unprioritised,
// then done tasks. Secondary sort preserves original line order.
func SortByPriority(items []Item) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if a.Priority != "" && b.Priority != "" {
			return a.Priority < b.Priority
		}
		if a.Priority != "" {
			return true
		}
		if b.Priority != "" {
			return false
		}
		return a.LineNum < b.LineNum
	})
	return out
}

// FormatItem formats an item for terminal display using cfg's color scheme.
// width is the digit count for line number padding (based on total lines in file).
func FormatItem(item Item, cfg Config, width int) string {
	numFmt := "%0" + strconv.Itoa(width) + "d"
	numStr := fmt.Sprintf(numFmt, item.LineNum)
	text := item.Raw

	if cfg.Plain {
		return numStr + " " + text
	}

	cs := cfg.Colors
	numOut := applyColor(numStr, itemBaseColor(item, cs))
	return numOut + " " + colorizeItem(item, text, cs)
}

// colorizeItem applies the color scheme to a single task's text.
// It wraps the whole line with the priority/done base color, then overlays
// per-token colors for projects, contexts, dates, and key:value pairs when
// the corresponding COLOR_* variables are set.
func colorizeItem(item Item, text string, cs ColorScheme) string {
	baseColor := itemBaseColor(item, cs)

	hasTokenColors := cs.Project != "" || cs.Context != "" ||
		cs.Date != "" || cs.Meta != ""

	if !hasTokenColors {
		return applyColor(text, baseColor)
	}

	return colorizeTokens(text, baseColor, cs)
}

// itemBaseColor returns the priority/done base color for an item.
func itemBaseColor(item Item, cs ColorScheme) string {
	if item.Done {
		return cs.Done
	}
	switch item.Priority {
	case "A":
		return cs.PriA
	case "B":
		return cs.PriB
	case "C":
		return cs.PriC
	default:
		if item.Priority != "" {
			return cs.PriX
		}
		return ""
	}
}

// colorizeTokens scans text word by word, applying per-token colors for
// +projects, @contexts, YYYY-MM-DD dates, and key:value pairs, then
// restoring the base color between tokens.
func colorizeTokens(text, baseColor string, cs ColorScheme) string {
	var b strings.Builder
	if baseColor != "" {
		b.WriteString(baseColor)
	}

	i := 0
	for i < len(text) {
		// Collect leading spaces / separators.
		j := i
		for j < len(text) && text[j] == ' ' {
			j++
		}
		b.WriteString(text[i:j])
		i = j
		if i >= len(text) {
			break
		}

		// Find end of word.
		k := i
		for k < len(text) && text[k] != ' ' {
			k++
		}
		word := text[i:k]
		i = k

		tokenColor := tokenColor(word, cs)
		if tokenColor != "" {
			if baseColor != "" {
				b.WriteString(Default)
			}
			b.WriteString(tokenColor)
			b.WriteString(word)
			b.WriteString(Default)
			if baseColor != "" {
				b.WriteString(baseColor)
			}
		} else {
			b.WriteString(word)
		}
	}

	if baseColor != "" {
		b.WriteString(Default)
	}
	return b.String()
}

// tokenColor returns the color code for a single word token, or "" if none applies.
func tokenColor(word string, cs ColorScheme) string {
	if strings.HasPrefix(word, "+") && cs.Project != "" {
		return cs.Project
	}
	if strings.HasPrefix(word, "@") && cs.Context != "" {
		return cs.Context
	}
	if cs.Date != "" && reDate.MatchString(word+" ") {
		return cs.Date
	}
	if cs.Meta != "" && isKeyValue(word) {
		return cs.Meta
	}
	return ""
}

// isKeyValue returns true for word:value tokens (but not URLs or bare colons).
func isKeyValue(word string) bool {
	idx := strings.IndexByte(word, ':')
	if idx <= 0 || idx == len(word)-1 {
		return false
	}
	key := word[:idx]
	val := word[idx+1:]
	// Key must be all word chars; value must be non-empty and not look like //URL.
	return isWordChars(key) && val != "" && !strings.HasPrefix(val, "//")
}

func isWordChars(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return len(s) > 0
}

// numWidth returns the number of decimal digits needed to represent n.
func numWidth(n int) int {
	if n < 10 {
		return 1
	}
	w := 0
	for n > 0 {
		w++
		n /= 10
	}
	return w
}
