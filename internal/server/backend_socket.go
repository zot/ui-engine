// CRC: crc-BackendSocket.md, crc-ProtocolDetector.md, crc-PacketProtocol.md
// Spec: deployment.md
package server

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/zot/ui/internal/protocol"
)

// BackendSocket handles the backend API socket.
type BackendSocket struct {
	socketPath  string
	listener    net.Listener
	handler     *protocol.Handler
	httpHandler *HTTPEndpoint
	connections map[string]net.Conn
	verbosity   int
	closed      bool
	mu          sync.RWMutex
}

// NewBackendSocket creates a new backend socket handler.
func NewBackendSocket(socketPath string, handler *protocol.Handler, httpHandler *HTTPEndpoint) *BackendSocket {
	return &BackendSocket{
		socketPath:  socketPath,
		handler:     handler,
		httpHandler: httpHandler,
		connections: make(map[string]net.Conn),
	}
}

// SetVerbosity sets the verbosity level for connection logging.
func (bs *BackendSocket) SetVerbosity(level int) {
	bs.verbosity = level
}

// DefaultSocketPath returns the platform-specific default socket path.
func DefaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\ui`
	}
	return "/tmp/ui.sock"
}

// Listen starts listening on the backend socket.
func (bs *BackendSocket) Listen() error {
	// Remove existing socket file on Unix
	if runtime.GOOS != "windows" {
		os.Remove(bs.socketPath)
	}

	// Create listener
	var err error
	if runtime.GOOS == "windows" {
		// Windows named pipe
		// Note: For full Windows support, would need npipe package
		// For now, fall back to TCP on Windows
		bs.listener, err = net.Listen("tcp", "127.0.0.1:0")
	} else {
		bs.listener, err = net.Listen("unix", bs.socketPath)
	}
	if err != nil {
		return err
	}

	// Accept connections
	go bs.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections.
func (bs *BackendSocket) acceptLoop() {
	for {
		conn, err := bs.listener.Accept()
		if err != nil {
			bs.mu.RLock()
			closed := bs.closed
			bs.mu.RUnlock()
			if closed {
				return // Socket closed gracefully
			}
			log.Printf("Accept error: %v", err)
			continue
		}

		go bs.handleConnection(conn)
	}
}

// handleConnection handles a new backend connection.
func (bs *BackendSocket) handleConnection(conn net.Conn) {
	connID := "backend-" + conn.RemoteAddr().String()

	bs.mu.Lock()
	bs.connections[connID] = conn
	bs.mu.Unlock()

	// Log connection event (verbosity level 1)
	if bs.verbosity >= 1 {
		log.Printf("[v1] Backend connected: %s", connID)
	}

	defer func() {
		bs.mu.Lock()
		delete(bs.connections, connID)
		bs.mu.Unlock()
		conn.Close()
		// Log disconnection event (verbosity level 1)
		if bs.verbosity >= 1 {
			log.Printf("[v1] Backend disconnected: %s", connID)
		}
	}()

	// Detect protocol by peeking first 4 bytes
	reader := bufio.NewReader(conn)
	peek, err := reader.Peek(4)
	if err != nil {
		if err != io.EOF {
			log.Printf("Peek error: %v", err)
		}
		return
	}

	if isHTTPPrefix(peek) {
		bs.handleHTTPConnection(reader, conn)
	} else {
		bs.handlePacketConnection(reader, conn, connID)
	}
}

// isHTTPPrefix checks if the peek bytes indicate HTTP protocol.
func isHTTPPrefix(peek []byte) bool {
	if len(peek) < 4 {
		return false
	}

	// Check for HTTP method prefixes
	prefixes := []string{"GET ", "POST", "PUT ", "DELE", "HEAD", "PATC", "OPTI"}
	s := string(peek)
	for _, prefix := range prefixes {
		if s == prefix || (len(s) >= len(prefix) && s[:len(prefix)] == prefix) {
			return true
		}
	}
	return false
}

// handleHTTPConnection handles an HTTP connection over the socket.
func (bs *BackendSocket) handleHTTPConnection(reader *bufio.Reader, conn net.Conn) {
	// For HTTP over socket, we'd need to implement a basic HTTP parser
	// or use net/http with a custom listener
	// For now, return an error response
	conn.Write([]byte("HTTP/1.1 501 Not Implemented\r\n\r\n"))
}

// handlePacketConnection handles a packet-protocol connection.
func (bs *BackendSocket) handlePacketConnection(reader *bufio.Reader, conn net.Conn, connID string) {
	for {
		// Read 4-byte length prefix
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			if err != io.EOF {
				log.Printf("Read length error: %v", err)
			}
			return
		}

		length := binary.BigEndian.Uint32(lenBuf)
		if length > 10*1024*1024 { // 10MB max
			log.Printf("Message too large: %d bytes", length)
			return
		}

		// Read JSON payload
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			log.Printf("Read payload error: %v", err)
			return
		}

		// Parse and handle message
		msg, err := protocol.ParseMessage(payload)
		if err != nil {
			log.Printf("Parse message error: %v", err)
			bs.writePacketError(conn, "Invalid message format")
			continue
		}

		resp, err := bs.handler.HandleMessage(connID, msg)
		if err != nil {
			bs.writePacketError(conn, err.Error())
			continue
		}

		// Write response
		bs.writePacketResponse(conn, resp)
	}
}

// writePacketResponse writes a packet-protocol response.
func (bs *BackendSocket) writePacketResponse(conn net.Conn, resp *protocol.Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	// Write length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}

	// Write payload
	_, err = conn.Write(data)
	return err
}

// writePacketError writes an error response.
func (bs *BackendSocket) writePacketError(conn net.Conn, message string) error {
	resp := &protocol.Response{Error: message}
	return bs.writePacketResponse(conn, resp)
}

// Broadcast sends a message to all connected backends.
func (bs *BackendSocket) Broadcast(msg *protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	bs.mu.RLock()
	defer bs.mu.RUnlock()

	for _, conn := range bs.connections {
		conn.Write(lenBuf)
		conn.Write(data)
	}
	return nil
}

// Close closes the socket and all connections.
func (bs *BackendSocket) Close() error {
	bs.mu.Lock()
	bs.closed = true
	bs.mu.Unlock()

	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, conn := range bs.connections {
		conn.Close()
	}
	bs.connections = make(map[string]net.Conn)

	if bs.listener != nil {
		err := bs.listener.Close()
		bs.listener = nil

		// Remove socket file on Unix
		if runtime.GOOS != "windows" {
			os.Remove(bs.socketPath)
		}

		return err
	}
	return nil
}

// GetSocketPath returns the socket path.
func (bs *BackendSocket) GetSocketPath() string {
	return bs.socketPath
}
