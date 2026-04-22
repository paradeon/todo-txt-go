# todo.txt-go

A Go implementation of the [todo.txt CLI](https://github.com/todotxt/todo.txt-cli), compatible with the original `todo.cfg` configuration format.

## Installation

```bash
go build -o todo .
```

Move the binary somewhere on your `$PATH`, e.g. `mv todo /usr/local/bin/`.

## Usage

```
todo [-fhpntvV] [-d config] ACTION [task_number] [task_description]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d FILE` | Use FILE as configuration file |
| `-f` | Force — skip confirmation prompts |
| `-n` / `-p` | No colors / plain text output |
| `-t` | Prepend creation date when adding tasks |
| `-v` | Verbose output |
| `-V` | Print version and exit |

### Actions

| Action | Alias | Description |
|--------|-------|-------------|
| `add TEXT` | `a` | Add a new task |
| `addm TEXT` | | Add multiple newline-separated tasks |
| `addto DEST TEXT` | | Add a line to any file in `TODO_DIR` |
| `append NR TEXT` | `app` | Append text to task NR |
| `prepend NR TEXT` | `prep` | Prepend text to task NR (after priority/date) |
| `replace NR TEXT` | | Replace task NR entirely |
| `del NR [TERM]` | `rm` | Delete task NR, or remove TERM from it |
| `do NR [NR...]` | | Mark task(s) as done |
| `pri NR PRIORITY` | `p` | Set priority (A–Z) on task NR |
| `depri NR [NR...]` | `dp` | Remove priority from task(s) |
| `list [TERM...]` | `ls` | List open tasks sorted by priority |
| `listall [TERM...]` | `lsa` | List tasks from todo.txt and done.txt |
| `listpri [PRI]` | `lsp` | List prioritised tasks (e.g. `A` or `A-C`) |
| `listcon [TERM...]` | `lsc` | List `@context` tags |
| `listproj [TERM...]` | `lsprj` | List `+project` tags |
| `listfile SRC [TERM...]` | `lf` | List tasks from a specific file |
| `archive` | | Move done tasks to done.txt, strip blank lines |
| `deduplicate` | | Remove duplicate lines from todo.txt |
| `move NR DEST [SRC]` | `mv` | Move task between files |
| `report` | | Append open/done counts to report.txt |
| `help [ACTION]` | | Show help (optionally for a specific action) |
| `shorthelp` | | One-line summary of each action |

## Configuration

todo.txt-go looks for a config file in these locations (first match wins):

1. `~/.config/todo/todo.cfg`
2. `~/.todo.cfg`
3. `~/.todo/config`
4. `~/todo/config`

Override with `-d FILE`.

The config file uses shell `export VAR=value` syntax (bare `VAR=value` also works). Variable references (`$VAR`, `${VAR}`, `${VAR:-default}`) and ANSI escape sequences (`\033`, `\e`) are supported.

### Key variables

```sh
export TODO_DIR="$HOME/todo"
export TODO_FILE="$TODO_DIR/todo.txt"
export DONE_FILE="$TODO_DIR/done.txt"
export REPORT_FILE="$TODO_DIR/report.txt"

export TODOTXT_DATE_ON_ADD=1       # prepend creation date on add
export TODOTXT_VERBOSE=2           # verbose output
export TODOTXT_DEFAULT_ACTION=ls   # run this when no action is given

# Priority colors
export PRI_A=$YELLOW
export PRI_B=$GREEN
export PRI_C=$LIGHT_BLUE
export PRI_X=$WHITE

# Token colors
export COLOR_DONE=$LIGHT_GREY
export COLOR_PROJECT=$RED
export COLOR_CONTEXT=$LIGHT_CYAN
export COLOR_DATE=$LIGHT_BLUE
export COLOR_NUMBER=$DARK_GREY
export COLOR_META=$CYAN
```

Available color names: `BLACK`, `RED`, `GREEN`, `BROWN`, `BLUE`, `PURPLE`, `CYAN`, `LIGHT_GREY`, `DARK_GREY`, `LIGHT_RED`, `LIGHT_GREEN`, `YELLOW`, `LIGHT_BLUE`, `LIGHT_PURPLE`, `LIGHT_CYAN`, `WHITE`, `NONE`.

## todo.txt format

Each line in todo.txt follows the [todo.txt format](https://github.com/todotxt/todo.txt):

```
(A) 2026-04-01 Call Mom +family @phone due:2026-04-30
x 2026-04-22 2026-04-01 Completed task +project @context
```

- `(A)` — priority, a single uppercase letter
- `2026-04-01` — creation date (optional)
- `x 2026-04-22` — completion marker and date
- `+project` — project tag
- `@context` — context tag
- `key:value` — metadata

## Development

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific test
go test -run TestAdd ./...

# Build
go build -o todo .
```
