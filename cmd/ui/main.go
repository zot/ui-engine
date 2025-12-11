// Package main is the entry point for the UI server.
// Spec: deployment.md
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/zot/ui/internal/bundle"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		// Default to serve command
		runServe(os.Args[1:])
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "serve":
		runServe(args)
	case "bundle":
		runBundle(args)
	case "extract":
		runExtract(args)
	case "ls":
		runLs(args)
	case "cat":
		runCat(args)
	case "cp":
		runCp(args)
	case "create", "destroy", "update", "watch", "unwatch", "get", "getObjects", "poll":
		runProtocolCommand(command, args)
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		printVersion()
	default:
		// Check if it's a flag
		if command[0] == '-' {
			runServe(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
			printHelp()
			os.Exit(1)
		}
	}
}

func runServe(args []string) {
	cfg, err := config.Load(args)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv := server.New(cfg)

	// Start cleanup worker
	srv.StartCleanupWorker(time.Hour)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
		os.Exit(0)
	}()

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Global socket path for protocol commands
var socketPath string

func runProtocolCommand(command string, args []string) {
	// Parse --socket flag from args
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	socket := fs.String("socket", defaultSocketPath(), "Server socket path")

	// Parse known flags, leaving positional args
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--socket" && i+1 < len(args) {
			socketPath = args[i+1]
			i++ // skip next arg
		} else {
			filteredArgs = append(filteredArgs, args[i])
		}
	}
	if socketPath == "" {
		socketPath = *socket
	}
	args = filteredArgs

	var msg *protocol.Message
	var err error

	switch command {
	case "create":
		msg, err = buildCreateMessage(args)
	case "destroy":
		msg, err = buildDestroyMessage(args)
	case "update":
		msg, err = buildUpdateMessage(args)
	case "watch":
		msg, err = buildWatchMessage(args)
	case "unwatch":
		msg, err = buildUnwatchMessage(args)
	case "get":
		msg, err = buildGetMessage(args)
	case "getObjects":
		msg, err = buildGetObjectsMessage(args)
	case "poll":
		msg, err = buildPollMessage(args)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Send to server and print response
	resp, err := sendToServer(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print response as JSON
	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

func defaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\ui`
	}
	return "/tmp/ui.sock"
}

func buildCreateMessage(args []string) (*protocol.Message, error) {
	// Parse --parent, --value, --props, --nowatch, --unbound flags
	var parentID int64
	var value json.RawMessage
	var props map[string]string
	var nowatch, unbound bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parent":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--parent requires a value")
			}
			i++
			fmt.Sscanf(args[i], "%d", &parentID)
		case "--value":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--value requires a value")
			}
			i++
			value = json.RawMessage(args[i])
		case "--props":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--props requires a value")
			}
			i++
			if err := json.Unmarshal([]byte(args[i]), &props); err != nil {
				// Try key=value format
				props = parseKeyValueProps(args[i])
			}
		case "--nowatch":
			nowatch = true
		case "--unbound":
			unbound = true
		}
	}

	return protocol.NewMessage(protocol.MsgCreate, protocol.CreateMessage{
		ParentID:   parentID,
		Value:      value,
		Properties: props,
		NoWatch:    nowatch,
		Unbound:    unbound,
	})
}

func buildDestroyMessage(args []string) (*protocol.Message, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("destroy requires a variable ID")
	}

	var varID int64
	if args[0] == "--id" && len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &varID)
	} else {
		fmt.Sscanf(args[0], "%d", &varID)
	}

	return protocol.NewMessage(protocol.MsgDestroy, protocol.DestroyMessage{
		VarID: varID,
	})
}

func buildUpdateMessage(args []string) (*protocol.Message, error) {
	var varID int64
	var value json.RawMessage
	var props map[string]string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--id requires a value")
			}
			i++
			fmt.Sscanf(args[i], "%d", &varID)
		case "--value":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--value requires a value")
			}
			i++
			value = json.RawMessage(args[i])
		case "--props":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--props requires a value")
			}
			i++
			if err := json.Unmarshal([]byte(args[i]), &props); err != nil {
				props = parseKeyValueProps(args[i])
			}
		}
	}

	if varID == 0 {
		return nil, fmt.Errorf("--id is required")
	}

	return protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID:      varID,
		Value:      value,
		Properties: props,
	})
}

func buildWatchMessage(args []string) (*protocol.Message, error) {
	var varID int64
	if len(args) > 0 {
		if args[0] == "--id" && len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &varID)
		} else {
			fmt.Sscanf(args[0], "%d", &varID)
		}
	}

	if varID == 0 {
		return nil, fmt.Errorf("variable ID is required")
	}

	return protocol.NewMessage(protocol.MsgWatch, protocol.WatchMessage{
		VarID: varID,
	})
}

func buildUnwatchMessage(args []string) (*protocol.Message, error) {
	var varID int64
	if len(args) > 0 {
		if args[0] == "--id" && len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &varID)
		} else {
			fmt.Sscanf(args[0], "%d", &varID)
		}
	}

	if varID == 0 {
		return nil, fmt.Errorf("variable ID is required")
	}

	return protocol.NewMessage(protocol.MsgUnwatch, protocol.WatchMessage{
		VarID: varID,
	})
}

func buildGetMessage(args []string) (*protocol.Message, error) {
	var ids []int64
	for _, arg := range args {
		var id int64
		fmt.Sscanf(arg, "%d", &id)
		if id > 0 {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one variable ID is required")
	}

	return protocol.NewMessage(protocol.MsgGet, protocol.GetMessage{
		VarIDs: ids,
	})
}

func buildGetObjectsMessage(args []string) (*protocol.Message, error) {
	var ids []int64
	for _, arg := range args {
		var id int64
		fmt.Sscanf(arg, "%d", &id)
		if id > 0 {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one object ID is required")
	}

	return protocol.NewMessage(protocol.MsgGetObjects, protocol.GetObjectsMessage{
		ObjIDs: ids,
	})
}

func buildPollMessage(args []string) (*protocol.Message, error) {
	var wait string
	for i := 0; i < len(args); i++ {
		if args[i] == "--wait" && i+1 < len(args) {
			wait = args[i+1]
			break
		}
	}

	return protocol.NewMessage(protocol.MsgPoll, protocol.PollMessage{
		Wait: wait,
	})
}

func parseKeyValueProps(s string) map[string]string {
	// Parse format: key=value,key2=value2 or key=value key2=value2
	props := make(map[string]string)
	// Simple implementation - just support single key=value for now
	for _, part := range splitProps(s) {
		idx := -1
		for i, c := range part {
			if c == '=' {
				idx = i
				break
			}
		}
		if idx > 0 {
			key := part[:idx]
			val := ""
			if idx+1 < len(part) {
				val = part[idx+1:]
			}
			props[key] = val
		}
	}
	return props
}

func splitProps(s string) []string {
	// Split on comma or space
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' || c == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func sendToServer(msg *protocol.Message) (*protocol.Response, error) {
	// Connect to the backend socket using packet protocol
	var conn net.Conn
	var err error

	if runtime.GOOS == "windows" {
		// Named pipe on Windows
		conn, err = net.Dial("pipe", socketPath)
	} else {
		// Unix socket on POSIX
		conn, err = net.Dial("unix", socketPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server at %s: %w", socketPath, err)
	}
	defer conn.Close()

	// Encode message
	data, err := msg.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Send with length prefix (4-byte big-endian)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return nil, fmt.Errorf("failed to write length: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	// Read response with length prefix
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}
	respLen := binary.BigEndian.Uint32(lenBuf)

	respData := make([]byte, respLen)
	if _, err := io.ReadFull(conn, respData); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var resp protocol.Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Site management commands

func runBundle(args []string) {
	fs := flag.NewFlagSet("bundle", flag.ExitOnError)
	output := fs.String("o", "", "Output path for bundled binary (required)")
	source := fs.String("src", "", "Source binary to bundle (default: current executable)")
	fs.Parse(args)

	if *output == "" {
		fmt.Fprintln(os.Stderr, "Error: -o output path is required")
		fmt.Fprintln(os.Stderr, "Usage: ui bundle [-src <binary>] -o <output> <site-dir>")
		os.Exit(1)
	}

	siteDir := fs.Arg(0)
	if siteDir == "" {
		fmt.Fprintln(os.Stderr, "Error: site directory is required")
		fmt.Fprintln(os.Stderr, "Usage: ui bundle [-src <binary>] -o <output> <site-dir>")
		os.Exit(1)
	}

	// Verify site directory exists
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: site directory %s does not exist\n", siteDir)
		os.Exit(1)
	}

	// Get source binary path
	sourcePath := *source
	if sourcePath == "" {
		// Default to current executable
		var err error
		sourcePath, err = os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get executable path: %v\n", err)
			os.Exit(1)
		}
	}

	// Verify source binary exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: source binary %s does not exist\n", sourcePath)
		os.Exit(1)
	}

	// Create bundle
	if err := bundle.CreateBundle(sourcePath, siteDir, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create bundle: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created bundled binary: %s\n", *output)
}

func runExtract(args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Check if bundled
	bundled, err := bundle.IsBundled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check bundle status: %v\n", err)
		os.Exit(1)
	}

	if !bundled {
		fmt.Fprintln(os.Stderr, "Error: binary is not bundled")
		os.Exit(1)
	}

	// Extract
	if err := bundle.ExtractBundle(targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to extract bundle: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Extracted site to: %s\n", targetDir)
}

func runLs(args []string) {
	// Check if bundled
	bundled, err := bundle.IsBundled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check bundle status: %v\n", err)
		os.Exit(1)
	}

	if !bundled {
		fmt.Fprintln(os.Stderr, "Error: binary is not bundled")
		os.Exit(1)
	}

	// List files
	files, err := bundle.ListFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list files: %v\n", err)
		os.Exit(1)
	}

	for _, file := range files {
		fmt.Println(file)
	}
}

func runCat(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: file path is required")
		fmt.Fprintln(os.Stderr, "Usage: ui cat <file>")
		os.Exit(1)
	}

	// Check if bundled
	bundled, err := bundle.IsBundled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check bundle status: %v\n", err)
		os.Exit(1)
	}

	if !bundled {
		fmt.Fprintln(os.Stderr, "Error: binary is not bundled")
		os.Exit(1)
	}

	// Read file
	content, err := bundle.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read file: %v\n", err)
		os.Exit(1)
	}

	os.Stdout.Write(content)
}

func runCp(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: source and destination are required")
		fmt.Fprintln(os.Stderr, "Usage: ui cp <pattern> <dest-dir>")
		os.Exit(1)
	}

	pattern := args[0]
	destDir := args[1]

	// Check if bundled
	bundled, err := bundle.IsBundled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check bundle status: %v\n", err)
		os.Exit(1)
	}

	if !bundled {
		fmt.Fprintln(os.Stderr, "Error: binary is not bundled")
		os.Exit(1)
	}

	// List files and match pattern
	files, err := bundle.ListFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list files: %v\n", err)
		os.Exit(1)
	}

	copied := 0
	for _, file := range files {
		matched, _ := filepath.Match(pattern, filepath.Base(file))
		if !matched {
			matched, _ = filepath.Match(pattern, file)
		}
		if matched {
			content, err := bundle.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", file, err)
				continue
			}

			destPath := filepath.Join(destDir, filepath.Base(file))
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create directory: %v\n", err)
				continue
			}

			if err := os.WriteFile(destPath, content, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", destPath, err)
				continue
			}

			fmt.Printf("Copied: %s -> %s\n", file, destPath)
			copied++
		}
	}

	if copied == 0 {
		fmt.Fprintln(os.Stderr, "No files matched the pattern")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`UI Server - Remote UI Platform

Usage: ui [command] [options]

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
  --storage       Storage type: memory, sqlite, postgresql
  --storage-path  SQLite database path
  --storage-url   PostgreSQL connection URL
  --lua           Enable Lua backend (default: true)
  --lua-path      Lua scripts directory
  --session-timeout    Session expiration (default: 24h, 0=never)
  --log-level     Log level: debug, info, warn, error
  --dir           Serve from directory instead of embedded site

Site Management Examples:
  ui bundle site/ -o my-app        Create bundled binary
  ui extract extracted/            Extract bundled site
  ui ls                            List bundled files
  ui cat index.html                Show file contents
  ui cp '*.js' lib/                Copy matching files

Server Examples:
  ui serve --port 8080
  ui serve --dir my-site/

Protocol Examples:
  ui create --parent 1 --value '{"name": "Alice"}' --props 'type=Person'
  ui update --id 5 --value '{"name": "Bob"}'
  ui get 1 2 3
  ui poll --wait 30s`)
}

func printVersion() {
	fmt.Println("UI Server v0.1.0")
}
