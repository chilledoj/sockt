package sockt

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockSocket implements the Socket interface for testing
type MockSocket struct {
	mu           sync.Mutex
	readChan     chan []byte
	readErr      error
	writtenData  [][]byte
	writeErr     error
	closed       atomic.Bool
	closeErr     error
	readBlocking bool
}

func NewMockSocket() *MockSocket {
	return &MockSocket{
		readChan:    make(chan []byte, 10),
		writtenData: make([][]byte, 0),
	}
}

func (m *MockSocket) Read(ctx context.Context) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	select {
	case data := <-m.readChan:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MockSocket) Write(msgType SocketMessageType, data []byte) error {
	if m.closed.Load() {
		return errors.New("socket closed")
	}
	if m.writeErr != nil {
		return m.writeErr
	}
	m.mu.Lock()
	m.writtenData = append(m.writtenData, data)
	m.mu.Unlock()
	return nil
}

func (m *MockSocket) Close() error {
	m.closed.Store(true)
	return m.closeErr
}

func (m *MockSocket) IsClosed() bool {
	return m.closed.Load()
}

func (m *MockSocket) GetWrittenData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.writtenData))
	copy(result, m.writtenData)
	return result
}

func (m *MockSocket) SendData(data []byte) {
	m.readChan <- data
}

// MockProcessor implements EventProcessor for testing
type MockProcessor struct {
	mu       sync.Mutex
	events   []Event[string]
	sendChan chan<- Event[string]
}

func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		events: make([]Event[string], 0),
	}
}

func (p *MockProcessor) Init(ch chan<- Event[string]) {
	p.sendChan = ch
}

func (p *MockProcessor) Process(event Event[string]) {
	p.mu.Lock()
	p.events = append(p.events, event)
	p.mu.Unlock()
}

func (p *MockProcessor) GetEvents() []Event[string] {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]Event[string], len(p.events))
	copy(result, p.events)
	return result
}

func (p *MockProcessor) SendEvent(event Event[string]) {
	if p.sendChan != nil {
		p.sendChan <- event
	}
}

func (p *MockProcessor) EventCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func (p *MockProcessor) WaitForEvents(count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.EventCount() >= count {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// Tests

func TestNewRoom(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()

	room := NewRoom(ctx, "test-room", processor)

	if room == nil {
		t.Fatal("NewRoom returned nil")
	}
	if room.ID != "test-room" {
		t.Errorf("expected room ID 'test-room', got '%s'", room.ID)
	}
	if room.Status() != Open {
		t.Errorf("expected status Open, got %v", room.Status())
	}
	if room.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections, got %d", room.ConnectionCount())
	}

	room.Stop()
}

func TestRoomSetStatus(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	room.SetStatus(Locked)
	if room.Status() != Locked {
		t.Errorf("expected status Locked, got %v", room.Status())
	}

	room.SetStatus(Inactive)
	if room.Status() != Inactive {
		t.Errorf("expected status Inactive, got %v", room.Status())
	}
}

func TestAddConnection(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	socket := NewMockSocket()
	err := room.AddConnection(socket, "conn-1")
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if room.ConnectionCount() != 1 {
		t.Errorf("expected 1 connection, got %d", room.ConnectionCount())
	}

	// Wait for connect event
	if !processor.WaitForEvents(1, 100*time.Millisecond) {
		t.Fatal("timed out waiting for connect event")
	}

	events := processor.GetEvents()
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}
	if events[0].Type != EventConnect {
		t.Errorf("expected EventConnect, got %v", events[0].Type)
	}
	if events[0].Subject != "conn-1" {
		t.Errorf("expected subject 'conn-1', got '%s'", events[0].Subject)
	}
}

func TestAddMultipleConnections(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	for i := 0; i < 5; i++ {
		socket := NewMockSocket()
		connID := string(rune('a' + i))
		err := room.AddConnection(socket, connID)
		if err != nil {
			t.Fatalf("AddConnection failed for %s: %v", connID, err)
		}
	}

	if room.ConnectionCount() != 5 {
		t.Errorf("expected 5 connections, got %d", room.ConnectionCount())
	}
}

func TestRemoveConnection(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	socket := NewMockSocket()
	room.AddConnection(socket, "conn-1")

	// Wait for connect event
	processor.WaitForEvents(1, 100*time.Millisecond)

	room.RemoveConnection("conn-1")

	if room.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections after removal, got %d", room.ConnectionCount())
	}

	if !socket.IsClosed() {
		t.Error("expected socket to be closed after RemoveConnection")
	}

	// Wait for disconnect event
	if !processor.WaitForEvents(2, 100*time.Millisecond) {
		t.Fatal("timed out waiting for disconnect event")
	}

	events := processor.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Type != EventDisconnect {
		t.Errorf("expected EventDisconnect, got %v", events[1].Type)
	}
}

func TestRemoveNonexistentConnection(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	// Should not panic or send disconnect event
	room.RemoveConnection("nonexistent")

	time.Sleep(50 * time.Millisecond)
	if processor.EventCount() != 0 {
		t.Errorf("expected no events for nonexistent connection, got %d", processor.EventCount())
	}
}

func TestSendTo(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	socket := NewMockSocket()
	room.AddConnection(socket, "conn-1")
	processor.WaitForEvents(1, 100*time.Millisecond)

	msg := Event[string]{
		Type:    EventMessage,
		Subject: "conn-1",
		Data:    []byte("hello"),
	}
	room.SendTo(msg, "conn-1")

	time.Sleep(50 * time.Millisecond)
	written := socket.GetWrittenData()
	if len(written) != 1 {
		t.Fatalf("expected 1 written message, got %d", len(written))
	}
	if string(written[0]) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(written[0]))
	}
}

func TestSendToNonexistentConnection(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	msg := Event[string]{
		Type: EventMessage,
		Data: []byte("hello"),
	}
	// Should not panic
	room.SendTo(msg, "nonexistent")
}

func TestSendToAll(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	sockets := make([]*MockSocket, 3)
	for i := 0; i < 3; i++ {
		sockets[i] = NewMockSocket()
		connID := string(rune('a' + i))
		room.AddConnection(sockets[i], connID)
	}
	processor.WaitForEvents(3, 100*time.Millisecond)

	msg := Event[string]{
		Type: EventMessage,
		Data: []byte("broadcast"),
	}
	room.SendToAll(msg)

	time.Sleep(50 * time.Millisecond)
	for i, socket := range sockets {
		written := socket.GetWrittenData()
		if len(written) != 1 {
			t.Errorf("socket %d: expected 1 written message, got %d", i, len(written))
			continue
		}
		if string(written[0]) != "broadcast" {
			t.Errorf("socket %d: expected 'broadcast', got '%s'", i, string(written[0]))
		}
	}
}

func TestRoomStop(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)

	sockets := make([]*MockSocket, 3)
	for i := 0; i < 3; i++ {
		sockets[i] = NewMockSocket()
		connID := string(rune('a' + i))
		room.AddConnection(sockets[i], connID)
	}
	processor.WaitForEvents(3, 100*time.Millisecond)

	room.Stop()

	// All sockets should be closed
	for i, socket := range sockets {
		if !socket.IsClosed() {
			t.Errorf("socket %d should be closed after Stop", i)
		}
	}

	if room.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections after Stop, got %d", room.ConnectionCount())
	}
}

func TestMessageProcessing(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	socket := NewMockSocket()
	room.AddConnection(socket, "conn-1")
	processor.WaitForEvents(1, 100*time.Millisecond)

	// Send message through socket
	socket.SendData([]byte("test message"))

	// Wait for message event
	if !processor.WaitForEvents(2, 100*time.Millisecond) {
		t.Fatal("timed out waiting for message event")
	}

	events := processor.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[1].Type != EventMessage {
		t.Errorf("expected EventMessage, got %v", events[1].Type)
	}
	if string(events[1].Data) != "test message" {
		t.Errorf("expected 'test message', got '%s'", string(events[1].Data))
	}
}

func TestWriteLoopSendsMessages(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	socket := NewMockSocket()
	room.AddConnection(socket, "conn-1")
	processor.WaitForEvents(1, 100*time.Millisecond)

	// Processor sends message through the channel
	processor.SendEvent(Event[string]{
		Type:    EventMessage,
		Subject: "conn-1",
		Data:    []byte("from processor"),
	})

	time.Sleep(50 * time.Millisecond)
	written := socket.GetWrittenData()
	if len(written) != 1 {
		t.Fatalf("expected 1 written message, got %d", len(written))
	}
	if string(written[0]) != "from processor" {
		t.Errorf("expected 'from processor', got '%s'", string(written[0]))
	}
}

// Concurrency tests

func TestConcurrentAddRemove(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOps := 100

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				socket := NewMockSocket()
				connID := string(rune('a'+id)) + string(rune('0'+j%10))
				room.AddConnection(socket, connID)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent removes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				connID := string(rune('a'+id)) + string(rune('0'+j%10))
				room.RemoveConnection(connID)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentSendToAll(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	// Add connections
	numConns := 5
	for i := 0; i < numConns; i++ {
		socket := NewMockSocket()
		connID := string(rune('a' + i))
		room.AddConnection(socket, connID)
	}
	processor.WaitForEvents(numConns, 100*time.Millisecond)

	// Concurrent SendToAll
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := Event[string]{
				Type: EventMessage,
				Data: []byte("concurrent message"),
			}
			room.SendToAll(msg)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentConnectionCount(t *testing.T) {
	ctx := context.Background()
	processor := NewMockProcessor()
	room := NewRoom(ctx, "test-room", processor)
	defer room.Stop()

	var wg sync.WaitGroup

	// Concurrent adds and count reads
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			socket := NewMockSocket()
			connID := string(rune('a' + id))
			room.AddConnection(socket, connID)
		}(i)
		go func() {
			defer wg.Done()
			_ = room.ConnectionCount()
		}()
	}

	wg.Wait()
}
