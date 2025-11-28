package sockt

type EventType int8

const (
	EventConnect EventType = iota + 1
	EventMessage
	EventDisconnect
)

func (e EventType) String() string {
	return [...]string{"connect", "message", "disconnect"}[e-1]
}
