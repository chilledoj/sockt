package sockt

import "context"

type MessageSender[ConnectionID comparable] interface {
	SendTo(msg Event[ConnectionID], connId ConnectionID)
	SendToAll(msg Event[ConnectionID])
}

type EventProcessor[ConnectionID comparable] interface {
	Init(chan<- Event[ConnectionID])
	Process(msg Event[ConnectionID])
}

type ConnectionValidator[ConnectionID comparable] interface {
	CanJoin(connId ConnectionID) error
}

type Socket interface {
	Read(context.Context) ([]byte, error)
	Write(SocketMessageType, []byte) error
	Close() error
}

// EventSender provides a safe, non-blocking way to send events.
// Processors should prefer using this over writing directly to the channel
// to avoid potential deadlocks when the channel buffer is full.
type EventSender[ConnectionID comparable] interface {
	// Send sends an event to the channel. Returns true if sent, false if channel is full or closed.
	Send(event Event[ConnectionID]) bool
	// Channel returns the underlying channel for backwards compatibility.
	// Prefer using Send() to avoid blocking.
	Channel() chan<- Event[ConnectionID]
}

// SafeEventSender wraps a channel with non-blocking send behavior.
type SafeEventSender[ConnectionID comparable] struct {
	ch     chan Event[ConnectionID]
	closed bool
}

// NewSafeEventSender creates a new SafeEventSender wrapping the given channel.
func NewSafeEventSender[ConnectionID comparable](ch chan Event[ConnectionID]) *SafeEventSender[ConnectionID] {
	return &SafeEventSender[ConnectionID]{ch: ch}
}

// Send attempts to send an event without blocking.
// Returns true if the event was sent, false if the channel is full or closed.
func (s *SafeEventSender[ConnectionID]) Send(event Event[ConnectionID]) bool {
	if s.closed {
		return false
	}
	select {
	case s.ch <- event:
		return true
	default:
		return false
	}
}

// Channel returns the underlying channel for backwards compatibility.
func (s *SafeEventSender[ConnectionID]) Channel() chan<- Event[ConnectionID] {
	return s.ch
}

// SafeEventProcessor is an optional interface that processors can implement
// to receive a SafeEventSender instead of a raw channel.
// This allows for non-blocking sends that prevent deadlocks.
type SafeEventProcessor[ConnectionID comparable] interface {
	InitSafe(EventSender[ConnectionID])
	Process(msg Event[ConnectionID])
}
