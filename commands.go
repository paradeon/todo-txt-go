package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// App holds config and exposes all commands.
type App struct {
	cfg Config
}

func NewApp(cfg Config) *App { return &App{cfg: cfg} }

func (app *App) today() string { return time.Now().Format("2006-01-02") }

func (app *App) readTodo() ([]Item, error) { return ReadItems(app.cfg.TodoFile) }
func (app *App) readDone() ([]Item, error) { return ReadItems(app.cfg.DoneFile) }

func (app *App) writeTodo(items []Item) error { return WriteItems(app.cfg.TodoFile, items) }
func (app *App) writeDone(items []Item) error { return WriteItems(app.cfg.DoneFile, items) }

// confirm prompts the user unless -f (force) is set.
func (app *App) confirm(prompt string) bool {
	if app.cfg.Force {
		return true
	}
	fmt.Printf("%s (y/n) ", prompt)
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(resp)) == "y"
}

// resolvePath resolves a filename relative to TodoDir if not absolute.
func (app *App) resolvePath(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return app.cfg.TodoDir + "/" + name
}

// ── add ─────────────────────────────────────────────────────────────────────

// Add adds a single new task to todo.txt.
func (app *App) Add(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: add THING I NEED TO DO +project @context")
	}
	text := strings.Join(args, " ")

	if app.cfg.DateOnAdd {
		if m := rePriority.FindString(text); m != "" {
			// Insert date after priority
			text = m + app.today() + " " + text[len(m):]
		} else {
			text = app.today() + " " + text
		}
	}

	items, err := app.readTodo()
	if err != nil {
		return err
	}

	// Effective next line number = number of non-blank trailing lines + 1, but
	// appending preserves the blank gap lines that del creates, so just use len+1.
	lineNum := len(items) + 1
	newItem := ParseItem(text, lineNum)
	items = append(items, newItem)

	if err := app.writeTodo(items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", lineNum, text)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d added.\n", lineNum)
	}
	return nil
}

// Addm adds multiple tasks from a newline-delimited string.
func (app *App) Addm(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: addm \"FIRST TASK\\nSECOND TASK\"")
	}
	text := strings.Join(args, " ")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := app.Add([]string{line}); err != nil {
			return err
		}
	}
	return nil
}

// Addto adds a line of text to any file in the todo directory.
func (app *App) Addto(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: addto DEST \"TEXT\"")
	}
	path := app.resolvePath(args[0])
	text := strings.Join(args[1:], " ")

	items, err := ReadItems(path)
	if err != nil {
		return err
	}
	lineNum := len(items) + 1
	items = append(items, ParseItem(text, lineNum))

	if err := WriteItems(path, items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", lineNum, text)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d added to %s.\n", lineNum, args[0])
	}
	return nil
}

// ── append / prepend ─────────────────────────────────────────────────────────

// Append appends text to the end of a task.
func (app *App) Append(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: append NR \"TEXT TO APPEND\"")
	}
	nr, items, idx, err := app.getItem(args[0])
	if err != nil {
		return err
	}
	text := strings.Join(args[1:], " ")
	items[idx].Raw = strings.TrimRight(items[idx].Raw, " ") + " " + text
	items[idx] = ParseItem(items[idx].Raw, nr)

	if err := app.writeTodo(items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", nr, items[idx].Raw)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d updated.\n", nr)
	}
	return nil
}

// Prepend prepends text to a task (after priority and date).
func (app *App) Prepend(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: prepend NR \"TEXT TO PREPEND\"")
	}
	nr, items, idx, err := app.getItem(args[0])
	if err != nil {
		return err
	}
	text := strings.Join(args[1:], " ")

	// Keep priority and date prefix intact; inject text before the description.
	raw := items[idx].Raw
	prefix := ""
	if m := rePriority.FindString(raw); m != "" {
		prefix += m
		raw = raw[len(m):]
	}
	if m := reDate.FindString(raw); m != "" {
		prefix += m
		raw = raw[len(m):]
	}
	updated := prefix + text + " " + raw
	items[idx] = ParseItem(updated, nr)

	if err := app.writeTodo(items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", nr, items[idx].Raw)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d updated.\n", nr)
	}
	return nil
}

// ── archive / deduplicate ────────────────────────────────────────────────────

// Archive moves done tasks from todo.txt to done.txt and removes blank lines.
func (app *App) Archive() error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}

	var remaining, archived []Item
	for _, item := range items {
		if item.Done && item.Raw != "" {
			archived = append(archived, item)
		} else {
			remaining = append(remaining, item)
		}
	}

	// Strip trailing blank lines from remaining.
	for len(remaining) > 0 && remaining[len(remaining)-1].Raw == "" {
		remaining = remaining[:len(remaining)-1]
	}
	// Renumber remaining.
	for i := range remaining {
		remaining[i].LineNum = i + 1
	}

	// Backup after blank-line removal but before removing done items,
	// matching the state sed -i.bak leaves in the original todo.txt-cli.
	if err := backupFile(app.cfg.TodoFile); err != nil {
		return err
	}
	if err := app.writeTodo(remaining); err != nil {
		return err
	}
	if len(archived) > 0 {
		if err := AppendItems(app.cfg.DoneFile, archived); err != nil {
			return err
		}
	}
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d task(s) archived.\n", len(archived))
	}
	return nil
}

// Deduplicate removes duplicate lines from todo.txt.
func (app *App) Deduplicate() error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	var result []Item
	removed := 0
	for _, item := range items {
		if item.Raw == "" {
			result = append(result, item)
			continue
		}
		if seen[item.Raw] {
			removed++
			continue
		}
		seen[item.Raw] = true
		result = append(result, item)
	}
	// Renumber.
	for i := range result {
		result[i].LineNum = i + 1
	}
	if err := app.writeTodo(result); err != nil {
		return err
	}
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d duplicate(s) removed.\n", removed)
	}
	return nil
}

// ── del ──────────────────────────────────────────────────────────────────────

// Del deletes a task or removes a specific term from a task.
func (app *App) Del(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: del NR [TERM]")
	}
	nr, items, idx, err := app.getItem(args[0])
	if err != nil {
		return err
	}

	if len(args) == 1 {
		if !app.confirm(fmt.Sprintf("Delete '%s'?", items[idx].Raw)) {
			fmt.Println("TODO: No tasks were deleted.")
			return nil
		}
		deleted := items[idx].Raw
		items[idx].Raw = ""
		if err := app.writeTodo(items); err != nil {
			return err
		}
		fmt.Printf("TODO: '%s' deleted.\n", deleted)
	} else {
		term := strings.Join(args[1:], " ")
		updated := strings.ReplaceAll(items[idx].Raw, term, "")
		// Collapse multiple spaces.
		for strings.Contains(updated, "  ") {
			updated = strings.ReplaceAll(updated, "  ", " ")
		}
		updated = strings.TrimSpace(updated)
		items[idx] = ParseItem(updated, nr)
		if err := app.writeTodo(items); err != nil {
			return err
		}
		fmt.Printf("%d %s\n", nr, items[idx].Raw)
		if app.cfg.Verbose {
			fmt.Printf("TODO: %d updated.\n", nr)
		}
	}
	return nil
}

// ── depri / pri ──────────────────────────────────────────────────────────────

// Depri removes the priority from one or more tasks.
func (app *App) Depri(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: depri NR [NR ...]")
	}
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	changed := false
	for _, arg := range args {
		nr, err := strconv.Atoi(arg)
		if err != nil || nr < 1 || nr > len(items) {
			fmt.Fprintf(os.Stderr, "TODO: %s is not a valid task number.\n", arg)
			continue
		}
		idx := nr - 1
		if items[idx].Raw == "" {
			fmt.Fprintf(os.Stderr, "TODO: %d doesn't exist.\n", nr)
			continue
		}
		if items[idx].Priority == "" {
			fmt.Printf("TODO: %d is already deprioritized.\n", nr)
			continue
		}
		items[idx].Priority = ""
		items[idx].Rebuild()
		fmt.Printf("%d %s\n", nr, items[idx].Raw)
		if app.cfg.Verbose {
			fmt.Printf("TODO: %d deprioritized.\n", nr)
		}
		changed = true
	}
	if changed {
		return app.writeTodo(items)
	}
	return nil
}

// Pri sets the priority on a task.
func (app *App) Pri(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pri NR PRIORITY")
	}
	nr, items, idx, err := app.getItem(args[0])
	if err != nil {
		return err
	}
	pri := strings.ToUpper(args[1])
	if len(pri) != 1 || pri[0] < 'A' || pri[0] > 'Z' {
		return fmt.Errorf("priority must be a single letter A-Z")
	}
	if items[idx].Done {
		return fmt.Errorf("TODO: %d is already done. Deprioritize is not possible.", nr)
	}
	items[idx].Priority = pri
	items[idx].Rebuild()

	if err := app.writeTodo(items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", nr, items[idx].Raw)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d prioritized (%s).\n", nr, pri)
	}
	return nil
}

// ── do ───────────────────────────────────────────────────────────────────────

// Do marks one or more tasks as done.
func (app *App) Do(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: do NR [NR ...]")
	}
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	changed := false
	for _, arg := range args {
		nr, err := strconv.Atoi(arg)
		if err != nil || nr < 1 || nr > len(items) {
			fmt.Fprintf(os.Stderr, "TODO: %s is not a valid task number.\n", arg)
			continue
		}
		idx := nr - 1
		if items[idx].Raw == "" {
			fmt.Fprintf(os.Stderr, "TODO: %d doesn't exist.\n", nr)
			continue
		}
		if items[idx].Done {
			fmt.Printf("TODO: %d is already done.\n", nr)
			continue
		}
		// Strip priority then prepend "x YYYY-MM-DD ", matching todo.txt-cli behavior.
		raw := items[idx].Raw
		if m := rePriority.FindString(raw); m != "" {
			raw = raw[len(m):]
		}
		items[idx].Raw = "x " + app.today() + " " + raw
		items[idx] = ParseItem(items[idx].Raw, nr)
		fmt.Printf("%d %s\n", nr, items[idx].Raw)
		if app.cfg.Verbose {
			fmt.Printf("TODO: %d marked as done.\n", nr)
		}
		changed = true
	}
	if changed {
		if err := app.writeTodo(items); err != nil {
			return err
		}
		if err := backupFile(app.cfg.TodoFile); err != nil {
			return err
		}
		if app.cfg.AutoArchive {
			return app.Archive()
		}
	}
	return nil
}

// ── replace ──────────────────────────────────────────────────────────────────

// Replace replaces an entire task line.
func (app *App) Replace(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: replace NR \"UPDATED TODO\"")
	}
	nr, items, idx, err := app.getItem(args[0])
	if err != nil {
		return err
	}
	text := strings.Join(args[1:], " ")
	items[idx] = ParseItem(text, nr)

	if err := app.writeTodo(items); err != nil {
		return err
	}
	fmt.Printf("%d %s\n", nr, items[idx].Raw)
	if app.cfg.Verbose {
		fmt.Printf("TODO: %d replaced.\n", nr)
	}
	return nil
}

// ── list variants ────────────────────────────────────────────────────────────

// List displays all open tasks, sorted by priority, filtered by optional terms.
func (app *App) List(args []string) error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	w := numWidth(len(items))

	var active []Item
	for _, item := range items {
		if !item.Done && item.Raw != "" {
			active = append(active, item)
		}
	}
	filtered := FilterItems(active, args)
	sorted := SortByPriority(filtered)

	for _, item := range sorted {
		fmt.Println(FormatItem(item, app.cfg, w))
	}
	fmt.Printf("--\nTODO: %d of %d tasks shown\n", len(sorted), len(active))
	return nil
}

// Listall displays tasks from both todo.txt and done.txt.
func (app *App) Listall(args []string) error {
	todoItems, err := app.readTodo()
	if err != nil {
		return err
	}
	doneItems, err := app.readDone()
	if err != nil {
		return err
	}

	totalTodo := len(todoItems)
	totalDone := len(doneItems)
	// Padding is based on todo.txt line count only, matching the original.
	// Done items are all displayed as 0 so they don't affect width.
	w := numWidth(totalTodo)

	// Done items are not addressable by line number, so display them as 0.
	for i := range doneItems {
		doneItems[i].LineNum = 0
	}

	all := append(todoItems, doneItems...)

	// Include blank lines when no terms are given (matching original cat behaviour).
	// When terms are present, FilterItems naturally excludes blanks.
	var filtered []Item
	if len(args) == 0 {
		filtered = all
	} else {
		filtered = FilterItems(all, args)
	}

	sorted := SortAlphabetical(filtered)

	for _, item := range sorted {
		fmt.Println(FormatItem(item, app.cfg, w))
	}

	// Count non-blank shown items per source for the summary line.
	shownTodo, shownDone := 0, 0
	for _, item := range sorted {
		if item.Raw == "" {
			continue
		}
		if item.Done {
			shownDone++
		} else {
			shownTodo++
		}
	}
	fmt.Println("--")
	fmt.Printf("TODO: %d of %d tasks shown\n", shownTodo, totalTodo)
	fmt.Printf("DONE: %d of %d tasks shown\n", shownDone, totalDone)
	fmt.Printf("total %d of %d tasks shown\n", shownTodo+shownDone, totalTodo+totalDone)
	return nil
}

// Listcon lists all @context tags found in todo.txt.
func (app *App) Listcon(args []string) error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	set := make(map[string]bool)
	for _, item := range items {
		if item.Raw == "" {
			continue
		}
		for _, ctx := range item.Contexts() {
			set[ctx] = true
		}
	}
	tags := sortedKeys(set)
	for _, tag := range tags {
		if len(args) == 0 || containsAny(tag, args) {
			fmt.Println(tag)
		}
	}
	return nil
}

// Listfile lists lines from a file in the todo directory.
// With no arguments it lists all *.txt files in TODO_DIR, matching the
// original todo.txt-cli behaviour.
func (app *App) Listfile(args []string) error {
	if len(args) == 0 {
		entries, err := os.ReadDir(app.cfg.TodoDir)
		if err != nil {
			return err
		}
		fmt.Println("Files in the todo.txt directory:")
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
				fmt.Println(e.Name())
			}
		}
		return nil
	}
	path := app.resolvePath(args[0])
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("TODO: file %s does not exist", args[0])
	}
	items, err := ReadItems(path)
	if err != nil {
		return err
	}
	w := numWidth(len(items))
	// Total = all non-blank items in the file (before term filter).
	total := 0
	for _, item := range items {
		if item.Raw != "" {
			total++
		}
	}
	filtered := FilterItems(items, args[1:])
	sorted := SortByPriority(filtered)
	for _, item := range sorted {
		fmt.Println(FormatItem(item, app.cfg, w))
	}
	fmt.Printf("--\n%s: %d of %d tasks shown\n", args[0], len(sorted), total)
	return nil
}

// Listpri lists prioritised tasks, optionally filtered to specific priorities.
func (app *App) Listpri(args []string) error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	w := numWidth(len(items))

	priorities := map[string]bool{}
	terms := args

	if len(args) > 0 {
		first := strings.ToUpper(args[0])
		if isPrioritySpec(first) {
			terms = args[1:]
			if len(first) == 1 {
				priorities[first] = true
			} else {
				// Range like A-C
				for c := first[0]; c <= first[2]; c++ {
					priorities[string(c)] = true
				}
			}
		}
	}

	var active []Item
	for _, item := range items {
		if item.Raw == "" || item.Done {
			continue
		}
		if len(priorities) == 0 {
			if item.Priority != "" {
				active = append(active, item)
			}
		} else if priorities[item.Priority] {
			active = append(active, item)
		}
	}

	filtered := FilterItems(active, terms)
	sorted := SortByPriority(filtered)

	for _, item := range sorted {
		fmt.Println(FormatItem(item, app.cfg, w))
	}
	fmt.Printf("--\nTODO: %d of %d tasks shown\n", len(sorted), len(active))
	return nil
}

// Listproj lists all +project tags found in todo.txt.
func (app *App) Listproj(args []string) error {
	items, err := app.readTodo()
	if err != nil {
		return err
	}
	set := make(map[string]bool)
	for _, item := range items {
		if item.Raw == "" {
			continue
		}
		for _, proj := range item.Projects() {
			set[proj] = true
		}
	}
	tags := sortedKeys(set)
	for _, tag := range tags {
		if len(args) == 0 || containsAny(tag, args) {
			fmt.Println(tag)
		}
	}
	return nil
}

// ── move ─────────────────────────────────────────────────────────────────────

// Move moves a task from SRC file to DEST file.
func (app *App) Move(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: move NR DEST [SRC]")
	}
	nr, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task number: %s", args[0])
	}
	dest := args[1]
	srcName := "todo.txt"
	if len(args) > 2 {
		srcName = args[2]
	}

	srcPath := app.resolvePath(srcName)
	destPath := app.resolvePath(dest)

	srcItems, err := ReadItems(srcPath)
	if err != nil {
		return err
	}
	if nr < 1 || nr > len(srcItems) {
		return fmt.Errorf("item %d doesn't exist in %s", nr, srcName)
	}
	item := srcItems[nr-1]
	if item.Raw == "" {
		return fmt.Errorf("item %d is empty in %s", nr, srcName)
	}

	destItems, err := ReadItems(destPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Blank out from source.
	srcItems[nr-1].Raw = ""
	// Append to dest.
	item.LineNum = len(destItems) + 1
	destItems = append(destItems, item)

	if err := WriteItems(srcPath, srcItems); err != nil {
		return err
	}
	if err := WriteItems(destPath, destItems); err != nil {
		return err
	}
	fmt.Printf("TODO: %d moved from '%s' to '%s'.\n", nr, srcName, dest)
	return nil
}

// ── report ───────────────────────────────────────────────────────────────────

// Report appends open/done counts to report.txt.
func (app *App) Report() error {
	todoItems, err := app.readTodo()
	if err != nil {
		return err
	}
	doneItems, err := app.readDone()
	if err != nil {
		return err
	}

	openCount := 0
	for _, item := range todoItems {
		if item.Raw != "" && !item.Done {
			openCount++
		}
	}
	doneCount := 0
	for _, item := range doneItems {
		if item.Raw != "" {
			doneCount++
		}
	}

	line := fmt.Sprintf("%s %d %d\n", app.today(), openCount, doneCount)
	f, err := os.OpenFile(app.cfg.ReportFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return err
	}

	fmt.Printf("TODO: Report file updated (%s)\n", app.cfg.ReportFile)
	fmt.Printf("Date\t\tOpen\tDone\n%s\t%d\t%d\n", app.today(), openCount, doneCount)
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// getItem parses a task number and returns (nr, items, idx, err).
func (app *App) getItem(numStr string) (int, []Item, int, error) {
	nr, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("invalid task number: %s", numStr)
	}
	items, err := app.readTodo()
	if err != nil {
		return 0, nil, 0, err
	}
	if nr < 1 || nr > len(items) {
		return 0, nil, 0, fmt.Errorf("TODO: %d doesn't exist", nr)
	}
	if items[nr-1].Raw == "" {
		return 0, nil, 0, fmt.Errorf("TODO: %d doesn't exist", nr)
	}
	return nr, items, nr - 1, nil
}

// isPrioritySpec returns true if s looks like "A" or "A-Z".
func isPrioritySpec(s string) bool {
	if len(s) == 1 {
		return s[0] >= 'A' && s[0] <= 'Z'
	}
	if len(s) == 3 && s[1] == '-' {
		return s[0] >= 'A' && s[0] <= 'Z' && s[2] >= 'A' && s[2] <= 'Z' && s[0] <= s[2]
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func containsAny(s string, terms []string) bool {
	lower := strings.ToLower(s)
	for _, t := range terms {
		if strings.Contains(lower, strings.ToLower(t)) {
			return true
		}
	}
	return false
}

// backupFile copies src to src.bak, matching the todo.txt-cli sed -i.bak behavior.
func backupFile(src string) error {
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()

	out, err := os.Create(src + ".bak")
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.ReadFrom(in)
	return err
}
