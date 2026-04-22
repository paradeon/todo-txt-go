package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	force := flag.Bool("f", false, "Force mode — skip confirmation prompts")
	plain := flag.Bool("p", false, "Plain text output (no colors)")
	noColor := flag.Bool("n", false, "No colors (same as -p)")
	dateOnAdd := flag.Bool("t", false, "Prepend creation date when adding tasks")
	verbose := flag.Bool("v", false, "Verbose output")
	showVersion := flag.Bool("V", false, "Display version and exit")
	configFile := flag.String("d", "", "Use FILE as configuration file")

	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("todo.txt-go %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()

	// Load config.
	cfg := DefaultConfig()
	if *configFile != "" {
		loaded, err := LoadConfig(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config %s: %v\n", *configFile, err)
			os.Exit(1)
		}
		cfg = loaded
	} else {
		for _, p := range DefaultConfigPaths() {
			if loaded, err := LoadConfig(p); err == nil {
				cfg = loaded
				break
			}
		}
	}

	// CLI flags override config-file values.
	if *force {
		cfg.Force = true
	}
	if *plain || *noColor {
		cfg.Plain = true
	}
	if *dateOnAdd {
		cfg.DateOnAdd = true
	}
	if *verbose {
		cfg.Verbose = true
	}

	// Apply TODOTXT_DEFAULT_ACTION when no arguments are given.
	if len(args) == 0 {
		if cfg.DefaultAction != "" {
			args = strings.Fields(cfg.DefaultAction)
		} else {
			usage()
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(cfg.TodoDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating TODO_DIR: %v\n", err)
		os.Exit(1)
	}

	app := NewApp(cfg)
	action := strings.ToLower(args[0])
	rest := args[1:]

	var err error
	switch action {
	case "add", "a":
		err = app.Add(rest)
	case "addm":
		err = app.Addm(rest)
	case "addto":
		err = app.Addto(rest)
	case "append", "app":
		err = app.Append(rest)
	case "archive":
		err = app.Archive()
	case "command":
		// Re-dispatch using only built-in actions (add-ons not supported).
		if len(rest) > 0 {
			args = rest
			action = strings.ToLower(rest[0])
			rest = rest[1:]
			err = dispatch(app, action, rest)
		}
	case "deduplicate":
		err = app.Deduplicate()
	case "del", "rm":
		err = app.Del(rest)
	case "depri", "dp":
		err = app.Depri(rest)
	case "do", "done":
		err = app.Do(rest)
	case "help":
		printHelp(rest)
	case "list", "ls":
		err = app.List(rest)
	case "listall", "lsa":
		err = app.Listall(rest)
	case "listaddons":
		fmt.Println("No add-ons installed.")
	case "listcon", "lsc":
		err = app.Listcon(rest)
	case "listfile", "lf":
		err = app.Listfile(rest)
	case "listpri", "lsp":
		err = app.Listpri(rest)
	case "listproj", "lsprj":
		err = app.Listproj(rest)
	case "move", "mv":
		err = app.Move(rest)
	case "prepend", "prep":
		err = app.Prepend(rest)
	case "pri", "p":
		err = app.Pri(rest)
	case "replace":
		err = app.Replace(rest)
	case "report":
		err = app.Report()
	case "shorthelp":
		shortHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %q\n\n", action)
		shortHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dispatch(app *App, action string, args []string) error {
	switch action {
	case "add", "a":
		return app.Add(args)
	case "addm":
		return app.Addm(args)
	case "addto":
		return app.Addto(args)
	case "append", "app":
		return app.Append(args)
	case "archive":
		return app.Archive()
	case "deduplicate":
		return app.Deduplicate()
	case "del", "rm":
		return app.Del(args)
	case "depri", "dp":
		return app.Depri(args)
	case "do", "done":
		return app.Do(args)
	case "list", "ls":
		return app.List(args)
	case "listall", "lsa":
		return app.Listall(args)
	case "listcon", "lsc":
		return app.Listcon(args)
	case "listfile", "lf":
		return app.Listfile(args)
	case "listpri", "lsp":
		return app.Listpri(args)
	case "listproj", "lsprj":
		return app.Listproj(args)
	case "move", "mv":
		return app.Move(args)
	case "prepend", "prep":
		return app.Prepend(args)
	case "pri", "p":
		return app.Pri(args)
	case "replace":
		return app.Replace(args)
	case "report":
		return app.Report()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: todo [-fhpantvV] [-d todo_config] action [task_number] [task_description]

  Options:
    -d FILE  Use FILE as configuration file
    -f       Force actions without confirmation
    -h       Show this help
    -n       No colors
    -p       Plain text (no colors)
    -t       Prepend creation date when adding
    -v       Verbose output
    -V       Show version

  Environment:
    TODO_DIR     Directory containing todo files (default: ~/todo)
    TODO_FILE    Path to todo.txt
    DONE_FILE    Path to done.txt
    REPORT_FILE  Path to report.txt

  Actions (use shorthelp for a compact list):
    add|a TEXT            Add a new task
    addm TEXT             Add multiple tasks (newline-separated)
    addto DEST TEXT       Add a line to any file in TODO_DIR
    append|app NR TEXT    Append TEXT to task NR
    archive               Move done tasks to done.txt, remove blank lines
    command [ACTION]      Execute built-in action (ignores add-ons)
    deduplicate           Remove duplicate lines from todo.txt
    del|rm NR [TERM]      Delete task NR, or remove TERM from it
    depri|dp NR [NR...]   Remove priority from task(s)
    do NR [NR...]         Mark task(s) as done
    help [ACTION]         Show help
    list|ls [TERM...]     List open tasks (sorted by priority)
    listall|lsa [TERM...] List tasks from todo.txt and done.txt
    listaddons            List installed add-ons
    listcon|lsc [TERM...] List @contexts
    listfile|lf SRC       List tasks from a file in TODO_DIR
    listpri|lsp [PRI]     List prioritised tasks (e.g. A or A-C)
    listproj|lsprj        List +projects
    move|mv NR DEST [SRC] Move task between files
    prepend|prep NR TEXT  Prepend TEXT to task NR
    pri|p NR PRIORITY     Set priority (A-Z) on task NR
    replace NR TEXT       Replace task NR entirely
    report                Append open/done counts to report.txt
    shorthelp             One-line summary of each action
`)
}

func shortHelp() {
	fmt.Print(`  add|a       Add a new task
  addm        Add multiple newline-separated tasks
  addto       Add a line to any file in TODO_DIR
  append|app  Append text to a task
  archive     Move done tasks to done.txt
  command     Run a built-in action
  deduplicate Remove duplicate lines
  del|rm      Delete a task or term
  depri|dp    Remove priority from task(s)
  do          Mark task(s) as done
  list|ls     List open tasks sorted by priority
  listall|lsa List tasks from todo.txt and done.txt
  listcon|lsc List @contexts
  listfile|lf List tasks from a specific file
  listpri|lsp List prioritised tasks
  listproj    List +projects
  move|mv     Move task between files
  prepend     Prepend text to a task
  pri|p       Set priority on a task
  replace     Replace a task entirely
  report      Append open/done counts to report.txt
  shorthelp   This summary
`)
}

func printHelp(actions []string) {
	if len(actions) == 0 {
		usage()
		return
	}
	// Per-action help.
	helps := map[string]string{
		"add":         "add|a \"THING I NEED TO DO +project @context\"\n  Adds the task to todo.txt. With -t, prepends today's date.",
		"a":           "add|a \"THING I NEED TO DO\"\n  Alias for add.",
		"addm":        "addm \"FIRST TASK\\nSECOND TASK\"\n  Adds multiple tasks, one per line.",
		"addto":       "addto DEST \"TEXT\"\n  Adds TEXT to file DEST inside TODO_DIR.",
		"append":      "append|app NR \"TEXT\"\n  Appends TEXT to the task at line NR.",
		"app":         "append|app NR \"TEXT\"\n  Alias for append.",
		"archive":     "archive\n  Moves done tasks from todo.txt to done.txt and strips blank lines.",
		"command":     "command [ACTION [ARGS]]\n  Executes ACTION using only built-in commands (ignores add-ons).",
		"deduplicate": "deduplicate\n  Removes duplicate lines from todo.txt.",
		"del":         "del|rm NR [TERM]\n  Without TERM: deletes task NR. With TERM: removes TERM from task NR.",
		"rm":          "del|rm NR [TERM]\n  Alias for del.",
		"depri":       "depri|dp NR [NR ...]\n  Removes the priority from each listed task.",
		"dp":          "depri|dp NR [NR ...]\n  Alias for depri.",
		"do":          "do|done NR [NR ...]\n  Marks each listed task as done with today's completion date.",
		"done":        "do|done NR [NR ...]\n  Alias for do.",
		"list":        "list|ls [TERM ...]\n  Lists all open tasks sorted by priority. Filters by all TERMs (AND).",
		"ls":          "list|ls [TERM ...]\n  Alias for list.",
		"listall":     "listall|lsa [TERM ...]\n  Lists tasks from both todo.txt and done.txt.",
		"lsa":         "listall|lsa [TERM ...]\n  Alias for listall.",
		"listcon":     "listcon|lsc [TERM ...]\n  Lists all @context tags in todo.txt.",
		"lsc":         "listcon|lsc [TERM ...]\n  Alias for listcon.",
		"listfile":    "listfile|lf SRC [TERM ...]\n  Lists tasks from file SRC in TODO_DIR.",
		"lf":          "listfile|lf SRC [TERM ...]\n  Alias for listfile.",
		"listpri":     "listpri|lsp [PRIORITIES] [TERM ...]\n  Lists prioritised tasks. PRIORITIES can be a single letter (A) or range (A-C).",
		"lsp":         "listpri|lsp [PRIORITIES] [TERM ...]\n  Alias for listpri.",
		"listproj":    "listproj|lsprj [TERM ...]\n  Lists all +project tags in todo.txt.",
		"lsprj":       "listproj|lsprj [TERM ...]\n  Alias for listproj.",
		"move":        "move|mv NR DEST [SRC]\n  Moves task NR from SRC (default todo.txt) to DEST.",
		"mv":          "move|mv NR DEST [SRC]\n  Alias for move.",
		"prepend":     "prepend|prep NR \"TEXT\"\n  Prepends TEXT to task NR (after priority and date).",
		"prep":        "prepend|prep NR \"TEXT\"\n  Alias for prepend.",
		"pri":         "pri|p NR PRIORITY\n  Adds or replaces the priority (A-Z) on task NR.",
		"p":           "pri|p NR PRIORITY\n  Alias for pri.",
		"replace":     "replace NR \"UPDATED TODO\"\n  Replaces the entire task at line NR.",
		"report":      "report\n  Appends a dated line with open/done counts to report.txt.",
		"shorthelp":   "shorthelp\n  Displays a one-line summary for each built-in action.",
	}
	for _, a := range actions {
		if h, ok := helps[strings.ToLower(a)]; ok {
			fmt.Println(h)
		} else {
			fmt.Fprintf(os.Stderr, "No help for unknown action: %s\n", a)
		}
	}
}

