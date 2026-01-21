package sockt

type EventType int8

const (
	EventConnect EventType = iota + 1
	EventMessage
	EventDisconnect
)

func (e EventType) String() string {
	switch e {
	case EventConnect:
		return "connect"
	case EventMessage:
		return "message"
	case EventDisconnect:
		return "disconnect"
	default:
		return "unknown"
	}
}
