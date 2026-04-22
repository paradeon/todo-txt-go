package main

import (
	"regexp"
	"strings"
)

// Item represents a single task line.
type Item struct {
	LineNum        int
	Raw            string
	Done           bool
	Priority       string // single uppercase letter A-Z, or ""
	CompletionDate string // YYYY-MM-DD
	CreationDate   string // YYYY-MM-DD
	Description    string // text after priority and dates
}

var (
	rePriority = regexp.MustCompile(`^\(([A-Z])\) `)
	reDate     = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) `)
	reContext  = regexp.MustCompile(`(?:^|\s)@(\S+)`)
	reProject  = regexp.MustCompile(`(?:^|\s)\+(\S+)`)
)

// ParseItem parses a single line into an Item.
func ParseItem(line string, lineNum int) Item {
	item := Item{LineNum: lineNum, Raw: line}
	s := line

	// Completed task: starts with "x "
	if strings.HasPrefix(s, "x ") {
		item.Done = true
		s = s[2:]
		if m := reDate.FindString(s); m != "" {
			item.CompletionDate = strings.TrimRight(m, " ")
			s = s[len(m):]
		}
		// Optional creation date after completion date
		if m := reDate.FindString(s); m != "" {
			item.CreationDate = strings.TrimRight(m, " ")
			s = s[len(m):]
		}
		item.Description = s
		return item
	}

	// Priority: (A) through (Z)
	if m := rePriority.FindStringSubmatch(s); m != nil {
		item.Priority = m[1]
		s = s[len(m[0]):]
	}

	// Creation date
	if m := reDate.FindString(s); m != "" {
		item.CreationDate = strings.TrimRight(m, " ")
		s = s[len(m):]
	}

	item.Description = s
	return item
}

// Rebuild reconstructs Raw from the item's parsed fields.
// Use this after modifying Priority, CreationDate, or Description on an incomplete task.
// For marking done, prepend "x date " to Raw directly to preserve the full original text.
func (item *Item) Rebuild() {
	if item.Done {
		s := "x "
		if item.CompletionDate != "" {
			s += item.CompletionDate + " "
		}
		if item.CreationDate != "" {
			s += item.CreationDate + " "
		}
		s += item.Description
		item.Raw = s
		return
	}
	s := ""
	if item.Priority != "" {
		s = "(" + item.Priority + ") "
	}
	if item.CreationDate != "" {
		s += item.CreationDate + " "
	}
	s += item.Description
	item.Raw = s
}

// Contexts returns all @context tags found in the description.
func (item Item) Contexts() []string {
	var result []string
	for _, m := range reContext.FindAllStringSubmatch(item.Description, -1) {
		result = append(result, "@"+m[1])
	}
	return result
}

// Projects returns all +project tags found in the description.
func (item Item) Projects() []string {
	var result []string
	for _, m := range reProject.FindAllStringSubmatch(item.Description, -1) {
		result = append(result, "+"+m[1])
	}
	return result
}
