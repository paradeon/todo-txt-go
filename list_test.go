package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ── isKeyValue ────────────────────────────────────────────────────────────────

func TestIsKeyValue(t *testing.T) {
	cases := []struct {
		word string
		want bool
	}{
		{"due:2026-04-22", true},
		{"t:2026-01-01", true},
		{"key:value", true},
		{"key_1:value", true},
		{"https://example.com", false}, // URL — // prefix
		{"http://example.com", false},
		{":nokey", false},   // empty key
		{"novalue:", false}, // empty value
		{"nocoion", false},
		{"(A)", false},
	}
	for _, c := range cases {
		got := isKeyValue(c.word)
		if got != c.want {
			t.Errorf("isKeyValue(%q) = %v, want %v", c.word, got, c.want)
		}
	}
}

// ── isWordChars ───────────────────────────────────────────────────────────────

func TestIsWordChars(t *testing.T) {
	if !isWordChars("abc") { t.Error("lowercase letters should pass") }
	if !isWordChars("ABC") { t.Error("uppercase letters should pass") }
	if !isWordChars("abc123") { t.Error("alphanumeric should pass") }
	if !isWordChars("a_b") { t.Error("underscore should pass") }
	if isWordChars("") { t.Error("empty string should fail") }
	if isWordChars("a-b") { t.Error("hyphen should fail") }
	if isWordChars("a b") { t.Error("space should fail") }
	if isWordChars("a.b") { t.Error("dot should fail") }
}

// ── numWidth ──────────────────────────────────────────────────────────────────

func TestNumWidth(t *testing.T) {
	cases := []struct{ n, want int }{
		{0, 1},
		{1, 1},
		{9, 1},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
	}
	for _, c := range cases {
		got := numWidth(c.n)
		if got != c.want {
			t.Errorf("numWidth(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

// ── SortAlphabetical ──────────────────────────────────────────────────────────

func TestSortAlphabetical_order(t *testing.T) {
	items := []Item{
		ParseItem("Banana", 1),
		ParseItem("(A) Apple", 2),
		ParseItem("Cherry", 3),
	}
	sorted := SortAlphabetical(items)
	// (A) sorts before A-Z letters because '(' < 'A' in ASCII.
	if sorted[0].Raw != "(A) Apple" {
		t.Errorf("expected (A) Apple first, got %q", sorted[0].Raw)
	}
	if sorted[1].Raw != "Banana" {
		t.Errorf("expected Banana second, got %q", sorted[1].Raw)
	}
	if sorted[2].Raw != "Cherry" {
		t.Errorf("expected Cherry third, got %q", sorted[2].Raw)
	}
}

func TestSortAlphabetical_caseInsensitive(t *testing.T) {
	items := []Item{
		ParseItem("banana", 1),
		ParseItem("Apple", 2),
	}
	sorted := SortAlphabetical(items)
	if sorted[0].Raw != "Apple" {
		t.Errorf("expected Apple first (case-insensitive), got %q", sorted[0].Raw)
	}
}

func TestSortAlphabetical_doesNotMutateOriginal(t *testing.T) {
	items := []Item{ParseItem("Zzz", 1), ParseItem("Aaa", 2)}
	SortAlphabetical(items)
	if items[0].Raw != "Zzz" {
		t.Error("original slice should not be mutated")
	}
}

// ── FilterItems ───────────────────────────────────────────────────────────────

func TestFilterItems_noTerms(t *testing.T) {
	items := []Item{
		ParseItem("(A) Call Mom", 1),
		ParseItem("Buy milk", 2),
		{Raw: ""}, // blank
	}
	got := FilterItems(items, nil)
	// blank excluded, both tasks present
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterItems_singleTerm(t *testing.T) {
	items := []Item{
		ParseItem("(A) Call Mom", 1),
		ParseItem("Buy milk", 2),
		ParseItem("Call dentist", 3),
	}
	got := FilterItems(items, []string{"call"})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterItems_multipleTermsAllRequired(t *testing.T) {
	items := []Item{
		ParseItem("Call Mom +family", 1),
		ParseItem("Call dentist", 2),
		ParseItem("Email Mom", 3),
	}
	got := FilterItems(items, []string{"call", "mom"})
	if len(got) != 1 || got[0].Raw != "Call Mom +family" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestFilterItems_caseInsensitive(t *testing.T) {
	items := []Item{ParseItem("BUY MILK", 1)}
	got := FilterItems(items, []string{"milk"})
	if len(got) != 1 {
		t.Error("case-insensitive match failed")
	}
}

func TestFilterItems_blanksExcluded(t *testing.T) {
	items := []Item{{Raw: ""}, {Raw: ""}}
	got := FilterItems(items, nil)
	if len(got) != 0 {
		t.Error("blank items should be excluded")
	}
}

// ── SortByPriority ────────────────────────────────────────────────────────────

func TestSortByPriority_orderABC(t *testing.T) {
	items := []Item{
		ParseItem("(C) Third", 1),
		ParseItem("(A) First", 2),
		ParseItem("(B) Second", 3),
	}
	sorted := SortByPriority(items)
	if sorted[0].Priority != "A" || sorted[1].Priority != "B" || sorted[2].Priority != "C" {
		t.Errorf("wrong order: %v", sorted)
	}
}

func TestSortByPriority_noPriorityAfterPriority(t *testing.T) {
	items := []Item{
		ParseItem("No priority", 1),
		ParseItem("(A) Has priority", 2),
	}
	sorted := SortByPriority(items)
	if sorted[0].Priority != "A" {
		t.Error("prioritized task should come first")
	}
}

func TestSortByPriority_doneTasksLast(t *testing.T) {
	items := []Item{
		ParseItem("x 2026-04-22 Done task", 1),
		ParseItem("(A) Active task", 2),
		ParseItem("Another active task", 3),
	}
	sorted := SortByPriority(items)
	if sorted[len(sorted)-1].Raw != "x 2026-04-22 Done task" {
		t.Error("done task should sort last")
	}
}

func TestSortByPriority_preservesLineOrderForEqual(t *testing.T) {
	items := []Item{
		ParseItem("Banana", 3),
		ParseItem("Apple", 1),
		ParseItem("Cherry", 2),
	}
	sorted := SortByPriority(items)
	// All no-priority, so original line-number order should be preserved.
	if sorted[0].Raw != "Apple" || sorted[1].Raw != "Cherry" || sorted[2].Raw != "Banana" {
		t.Errorf("line order not preserved: %v", sorted)
	}
}

func TestSortByPriority_doesNotMutateOriginal(t *testing.T) {
	items := []Item{
		ParseItem("(B) B task", 1),
		ParseItem("(A) A task", 2),
	}
	SortByPriority(items)
	if items[0].Priority != "B" {
		t.Error("original slice should not be mutated")
	}
}

// ── ReadItems / WriteItems / AppendItems ──────────────────────────────────────

func writeTmpTodo(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	return path
}

func TestReadItems_basic(t *testing.T) {
	path := writeTmpTodo(t, "(A) Call Mom", "Buy milk", "x 2026-04-22 Done task")
	items, err := ReadItems(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Priority != "A" {
		t.Errorf("item 1 priority: got %q", items[0].Priority)
	}
	if items[2].Done != true {
		t.Error("item 3 should be done")
	}
	// Line numbers should start at 1.
	for i, item := range items {
		if item.LineNum != i+1 {
			t.Errorf("item %d LineNum: got %d", i, item.LineNum)
		}
	}
}

func TestReadItems_missingFile(t *testing.T) {
	items, err := ReadItems("/nonexistent/path/todo.txt")
	if err != nil {
		t.Fatal("missing file should return nil, nil")
	}
	if items != nil {
		t.Error("items should be nil for missing file")
	}
}

func TestWriteItems_roundtrip(t *testing.T) {
	path := writeTmpTodo(t, "(A) Call Mom", "Buy milk")
	items, _ := ReadItems(path)
	items[1].Priority = "B"
	items[1].Rebuild()
	if err := WriteItems(path, items); err != nil {
		t.Fatal(err)
	}
	reread, _ := ReadItems(path)
	if len(reread) != 2 {
		t.Fatalf("expected 2 items, got %d", len(reread))
	}
	if reread[1].Priority != "B" {
		t.Errorf("expected priority B after write, got %q", reread[1].Priority)
	}
}

func TestAppendItems_addsLines(t *testing.T) {
	path := writeTmpTodo(t, "Task one")
	extra := []Item{ParseItem("Task two", 2), ParseItem("Task three", 3)}
	if err := AppendItems(path, extra); err != nil {
		t.Fatal(err)
	}
	items, _ := ReadItems(path)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[2].Raw != "Task three" {
		t.Errorf("last item: got %q", items[2].Raw)
	}
}

func TestAppendItems_createsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.txt")
	items := []Item{ParseItem("New task", 1)}
	if err := AppendItems(path, items); err != nil {
		t.Fatal(err)
	}
	read, _ := ReadItems(path)
	if len(read) != 1 || read[0].Raw != "New task" {
		t.Errorf("unexpected state: %v", read)
	}
}

// ── FormatItem ────────────────────────────────────────────────────────────────

func TestFormatItem_plainMode(t *testing.T) {
	item := ParseItem("(A) Call Mom", 3)
	cfg := Config{Plain: true}
	got := FormatItem(item, cfg, 2)
	if got != "03 (A) Call Mom" {
		t.Errorf("got %q", got)
	}
}

func TestFormatItem_plainModeWidth1(t *testing.T) {
	item := ParseItem("Buy milk", 5)
	cfg := Config{Plain: true}
	got := FormatItem(item, cfg, 1)
	if got != "5 Buy milk" {
		t.Errorf("got %q", got)
	}
}

func TestFormatItem_noColorsNoColor(t *testing.T) {
	item := ParseItem("(A) Call Mom", 1)
	cfg := Config{Colors: ColorScheme{}}
	got := FormatItem(item, cfg, 1)
	// No color codes — should contain line number and raw text.
	if got != "1 (A) Call Mom" {
		t.Errorf("got %q", got)
	}
}
