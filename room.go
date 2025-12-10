package sockt

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
)

type Room[RoomID comparable, ConnectionID comparable] struct {
	ID RoomID

	// signals
	ctx    context.Context
	cancel context.CancelFunc

	// Data
	mu          sync.RWMutex
	status      RoomStatus
	connections map[ConnectionID]*SocketConnection[ConnectionID]
	sendChan    chan Event[ConnectionID]

	// Processor
	processor EventProcessor[ConnectionID]

	// Logger
	lg *log.Logger
}

func NewRoom[RoomID comparable, ConnectionID comparable](
	parentCtx context.Context,
	id RoomID,
	processor EventProcessor[ConnectionID],
) *Room[RoomID, ConnectionID] {
	ctx, cancel := context.WithCancel(parentCtx)
	r := &Room[RoomID, ConnectionID]{
		ID:          id,
		ctx:         ctx,
		cancel:      cancel,
		status:      Open,
		connections: make(map[ConnectionID]*SocketConnection[ConnectionID]),
		processor:   processor,
		sendChan:    make(chan Event[ConnectionID], 100),
		lg:          log.New(os.Stdout, "[room] ", log.LstdFlags),
	}

	go r.writeLoop()
	processor.Init(r.sendChan)

	return r
}

func (r *Room[RoomID, ConnectionID]) Status() RoomStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *Room[RoomID, ConnectionID]) SetStatus(status RoomStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
}

func (r *Room[RoomID, ConnectionID]) AddConnection(conn Socket, connId ConnectionID) error {

	validator, ok := r.processor.(ConnectionValidator[ConnectionID])
	if ok {
		if err := validator.CanJoin(connId); err != nil {
			return err
		}
	}

	newConn := newSocketConnection[ConnectionID](r.ctx, conn, connId)

	r.mu.Lock()
	r.connections[connId] = newConn
	r.mu.Unlock()

	// read + write loop here for connection.
	go r.readLoop(conn, connId)

	r.processor.Process(Event[ConnectionID]{
		Type:    EventConnect,
		Subject: connId,
	})

	return nil
}

func (r *Room[RoomID, ConnectionID]) RemoveConnection(connId ConnectionID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.connections, connId)
	r.processor.Process(Event[ConnectionID]{
		Type:    EventDisconnect,
		Subject: connId,
	})

}

func (r *Room[RoomID, ConnectionID]) readLoop(conn Socket, connId ConnectionID) {
	r.lg.Println("read loop started")
loop:
	for {
		msg, err := conn.Read(r.ctx)
		if err != nil {
			break loop
		}
		r.processor.Process(Event[ConnectionID]{
			Type:    EventMessage,
			Subject: connId,
			Data:    msg,
		})
		if r.ctx.Err() != nil {
			break loop
		}
	}

	if r.ctx.Err() != nil {
		r.lg.Printf("read loop ctx Err: %v\n", r.ctx.Err())
	}

	r.processor.Process(Event[ConnectionID]{
		Type:    EventDisconnect,
		Subject: connId,
	})
	r.lg.Println("read loop finished")
}

func (r *Room[RoomID, ConnectionID]) writeLoop() {
	r.lg.Println("write loop started")
loop:
	for {
		select {
		case msg := <-r.sendChan:
			r.lg.Printf("write loop got msg: %v\n", msg)
			// send
			r.mu.RLock()
			conn, ok := r.connections[msg.Subject]
			if !ok {
				fmt.Printf("write loop: connection not found for subject: %v\n", msg.Subject)
				r.mu.RUnlock()
				continue loop
			}
			r.mu.RUnlock()
			if err := conn.Send(SocketMessageBinary, msg.Data); err != nil {
				fmt.Printf("write loop: error sending message: %v\n", err)
			}
		case <-r.ctx.Done():
			break loop
		}
	}
	r.lg.Println("write loop finished")
}
