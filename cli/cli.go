// Package cli provides the command-line interface for remote-ui.
// It exports Run() and RunWithHooks() to allow extension by wrapper projects.
package cli

import (
	"fmt"
	"os"
)

// Hooks allows extending the CLI with additional commands.
type Hooks struct {
	// BeforeDispatch is called before command dispatch.
	// Return (handled=true, exitCode) to skip normal dispatch.
	BeforeDispatch func(command string, args []string) (handled bool, exitCode int)

	// CustomHelp returns additional help text to append.
	CustomHelp func() string

	// CustomVersion returns version info to append (optional).
	CustomVersion func() string
}

// Run executes the CLI with the given arguments.
// Returns exit code (0 = success, non-zero = error).
func Run(args []string) int {
	return RunWithHooks(args, nil)
}

// RunWithHooks executes CLI with extension hooks.
func RunWithHooks(args []string, hooks *Hooks) int {
	if len(args) < 1 {
		return runServe(args)
	}

	command := args[0]
	cmdArgs := args[1:]

	// Let hooks intercept first
	if hooks != nil && hooks.BeforeDispatch != nil {
		if handled, code := hooks.BeforeDispatch(command, cmdArgs); handled {
			return code
		}
	}

	switch command {
	case "serve":
		return runServe(cmdArgs)
	case "bundle":
		return runBundle(cmdArgs)
	case "extract":
		return runExtract(cmdArgs)
	case "ls":
		return runLs(cmdArgs)
	case "cat":
		return runCat(cmdArgs)
	case "cp":
		return runCp(cmdArgs)
	case "create", "destroy", "update", "watch", "unwatch", "get", "getObjects", "poll":
		return runProtocolCommand(command, cmdArgs)
	case "help", "-h", "--help":
		printHelp(hooks)
		return 0
	case "version", "-v", "--version":
		printVersion(hooks)
		return 0
	default:
		// Check if it's a flag (starts with -)
		if len(command) > 0 && command[0] == '-' {
			return runServe(args)
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp(hooks)
		return 1
	}
}

func printHelp(hooks *Hooks) {
	fmt.Println(`UI Engine Server

Usage: ui-engine [command] [options]

Server Commands:
  serve           Start the UI server (default)

Site Management:
  bundle          Create binary with custom site bundled
  extract         Extract bundled site to filesystem
  ls              List files in bundled site
  cat             Display contents of a bundled file
  cp              Copy files from bundled site

Protocol Commands:
  create          Create a new variable
  destroy         Destroy a variable
  update          Update a variable
  watch           Watch a variable
  unwatch         Stop watching a variable
  get             Get variable values
  getObjects      Get object values
  poll            Poll for pending responses

Server Options:
  --host          Browser listen address (default: 0.0.0.0)
  --port          Browser listen port (default: 8080)
  --socket        Backend API socket path
  --lua           Enable Lua backend (default: true)
  --lua-path      Lua scripts directory
  --session-timeout    Session expiration (default: 24h, 0=never)
  --log-level     Log level: debug, info, warn, error
  --dir           Serve from directory instead of embedded site

Site Management Examples:
  ui-engine bundle site/ -o my-app        Create bundled binary
  ui-engine extract extracted/            Extract bundled site
  ui-engine ls                            List bundled files
  ui-engine cat index.html                Show file contents
  ui-engine cp '*.js' lib/                Copy matching files

Server Examples:
  ui-engine serve --port 8080
  ui-engine serve --dir my-site/

Protocol Examples:
  ui-engine create --parent 1 --value '{"name": "Alice"}' --props 'type=Person'
  ui-engine update --id 5 --value '{"name": "Bob"}'
  ui-engine get 1 2 3
  ui-engine poll --wait 30s`)

	if hooks != nil && hooks.CustomHelp != nil {
		fmt.Println(hooks.CustomHelp())
	}
}

func printVersion(hooks *Hooks) {
	fmt.Println("UI Engine v0.1.0")
	if hooks != nil && hooks.CustomVersion != nil {
		fmt.Println(hooks.CustomVersion())
	}
}
