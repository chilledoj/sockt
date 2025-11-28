package sockt

import (
	"context"
)

type SocketMessageType = uint8

const (
	SocketMessageText SocketMessageType = iota
	SocketMessageBinary
)

type SocketConnection[ConnectionID comparable] struct {
	conn   Socket
	connID ConnectionID

	// signals
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc
}

func newSocketConnection[ConnectionID comparable](
	parentCtx context.Context,
	conn Socket,
	connID ConnectionID,
) *SocketConnection[ConnectionID] {
	ctx, cancel := context.WithCancel(parentCtx)
	return &SocketConnection[ConnectionID]{
		conn:      conn,
		connID:    connID,
		parentCtx: parentCtx,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (c *SocketConnection[ConnectionID]) Context() context.Context {
	return c.ctx
}

func (c *SocketConnection[ConnectionID]) Close() {
	c.cancel()
}

func (c *SocketConnection[ConnectionID]) Conn() Socket {
	return c.conn
}

func (c *SocketConnection[ConnectionID]) ConnId() ConnectionID {
	return c.connID
}

func (c *SocketConnection[ConnectionID]) Send(messageType SocketMessageType, bytes []byte) error {
	return c.conn.Write(messageType, bytes)
}

func (c *SocketConnection[ConnectionID]) Read(ctx context.Context) ([]byte, error) {
	return c.conn.Read(ctx)
}
