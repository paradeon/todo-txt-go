package main

import (
	"reflect"
	"testing"
)

// ── ParseItem ────────────────────────────────────────────────────────────────

func TestParseItem_blank(t *testing.T) {
	item := ParseItem("", 1)
	if item.Raw != "" || item.Done || item.Priority != "" || item.Description != "" {
		t.Errorf("blank line not parsed as empty item: %+v", item)
	}
}

func TestParseItem_simple(t *testing.T) {
	item := ParseItem("Call Mom", 3)
	if item.LineNum != 3 { t.Errorf("LineNum: got %d", item.LineNum) }
	if item.Done { t.Error("should not be done") }
	if item.Priority != "" { t.Errorf("Priority: got %q", item.Priority) }
	if item.CreationDate != "" { t.Errorf("CreationDate: got %q", item.CreationDate) }
	if item.Description != "Call Mom" { t.Errorf("Description: got %q", item.Description) }
	if item.Raw != "Call Mom" { t.Errorf("Raw: got %q", item.Raw) }
}

func TestParseItem_priority(t *testing.T) {
	item := ParseItem("(A) Call Mom", 1)
	if item.Priority != "A" { t.Errorf("Priority: got %q", item.Priority) }
	if item.Description != "Call Mom" { t.Errorf("Description: got %q", item.Description) }
	if item.CreationDate != "" { t.Errorf("CreationDate should be empty") }
}

func TestParseItem_priorityAndDate(t *testing.T) {
	item := ParseItem("(B) 2026-04-01 Buy milk", 1)
	if item.Priority != "B" { t.Errorf("Priority: got %q", item.Priority) }
	if item.CreationDate != "2026-04-01" { t.Errorf("CreationDate: got %q", item.CreationDate) }
	if item.Description != "Buy milk" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_creationDateOnly(t *testing.T) {
	item := ParseItem("2026-04-01 Document +TodoTxt", 1)
	if item.Priority != "" { t.Errorf("Priority should be empty") }
	if item.CreationDate != "2026-04-01" { t.Errorf("CreationDate: got %q", item.CreationDate) }
	if item.Description != "Document +TodoTxt" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_invalidPriorityPosition(t *testing.T) {
	// Priority must appear first — this should NOT be parsed as priority.
	item := ParseItem("Really call Mom (A)", 1)
	if item.Priority != "" { t.Errorf("priority in wrong position should be ignored") }
	if item.Description != "Really call Mom (A)" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_lowercasePriorityIgnored(t *testing.T) {
	item := ParseItem("(b) Do something", 1)
	if item.Priority != "" { t.Errorf("lowercase priority should be ignored, got %q", item.Priority) }
}

func TestParseItem_doneNoDate(t *testing.T) {
	item := ParseItem("x Call Mom", 1)
	if !item.Done { t.Error("should be done") }
	if item.CompletionDate != "" { t.Errorf("CompletionDate: got %q", item.CompletionDate) }
	if item.Description != "Call Mom" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_doneWithCompletionDate(t *testing.T) {
	item := ParseItem("x 2026-04-22 Call Mom", 1)
	if !item.Done { t.Error("should be done") }
	if item.CompletionDate != "2026-04-22" { t.Errorf("CompletionDate: got %q", item.CompletionDate) }
	if item.CreationDate != "" { t.Errorf("CreationDate should be empty") }
	if item.Description != "Call Mom" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_doneBothDates(t *testing.T) {
	item := ParseItem("x 2026-04-22 2026-04-01 Review PR +Repo @github", 1)
	if !item.Done { t.Error("should be done") }
	if item.CompletionDate != "2026-04-22" { t.Errorf("CompletionDate: got %q", item.CompletionDate) }
	if item.CreationDate != "2026-04-01" { t.Errorf("CreationDate: got %q", item.CreationDate) }
	if item.Description != "Review PR +Repo @github" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_donePreservesPriority(t *testing.T) {
	// The do command prepends "x date " to the raw line, so priority remains in description.
	item := ParseItem("x 2026-04-22 (A) Call Mom", 1)
	if !item.Done { t.Error("should be done") }
	if item.CompletionDate != "2026-04-22" { t.Errorf("CompletionDate: got %q", item.CompletionDate) }
	// (A) is not a date so no CreationDate
	if item.CreationDate != "" { t.Errorf("CreationDate should be empty, got %q", item.CreationDate) }
	if item.Description != "(A) Call Mom" { t.Errorf("Description: got %q", item.Description) }
}

func TestParseItem_rawPreserved(t *testing.T) {
	raw := "(A) 2026-04-01 Call Mom +Family @phone"
	item := ParseItem(raw, 5)
	if item.Raw != raw { t.Errorf("Raw: got %q, want %q", item.Raw, raw) }
}

// ── Rebuild ──────────────────────────────────────────────────────────────────

func TestRebuild_setPriority(t *testing.T) {
	item := ParseItem("Call Mom", 1)
	item.Priority = "A"
	item.Rebuild()
	if item.Raw != "(A) Call Mom" { t.Errorf("got %q", item.Raw) }
}

func TestRebuild_removePriority(t *testing.T) {
	item := ParseItem("(A) Call Mom", 1)
	item.Priority = ""
	item.Rebuild()
	if item.Raw != "Call Mom" { t.Errorf("got %q", item.Raw) }
}

func TestRebuild_removePriorityKeepsDate(t *testing.T) {
	item := ParseItem("(A) 2026-04-01 Call Mom", 1)
	item.Priority = ""
	item.Rebuild()
	if item.Raw != "2026-04-01 Call Mom" { t.Errorf("got %q", item.Raw) }
}

func TestRebuild_changePriority(t *testing.T) {
	item := ParseItem("(A) Call Mom", 1)
	item.Priority = "C"
	item.Rebuild()
	if item.Raw != "(C) Call Mom" { t.Errorf("got %q", item.Raw) }
}

func TestRebuild_done(t *testing.T) {
	item := Item{
		Done:           true,
		CompletionDate: "2026-04-22",
		CreationDate:   "2026-04-01",
		Description:    "Call Mom",
	}
	item.Rebuild()
	want := "x 2026-04-22 2026-04-01 Call Mom"
	if item.Raw != want { t.Errorf("got %q, want %q", item.Raw, want) }
}

// ── Projects / Contexts ──────────────────────────────────────────────────────

func TestProjects(t *testing.T) {
	item := ParseItem("Call Mom +Family +PeaceLoveAndHappiness @phone", 1)
	got := item.Projects()
	want := []string{"+Family", "+PeaceLoveAndHappiness"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContexts(t *testing.T) {
	item := ParseItem("Call Mom +Family @iphone @phone", 1)
	got := item.Contexts()
	want := []string{"@iphone", "@phone"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProjects_none(t *testing.T) {
	item := ParseItem("No projects here", 1)
	if len(item.Projects()) != 0 { t.Error("expected no projects") }
}

func TestContexts_none(t *testing.T) {
	item := ParseItem("No contexts here", 1)
	if len(item.Contexts()) != 0 { t.Error("expected no contexts") }
}
