package sockt

type EventType int8

const (
	EventConnect EventType = iota + 1
	EventMessage
	EventDisconnect
)

func (e EventType) String() string {
	if e < 1 || e > 3 {
		return "unknown"
	}
	return [...]string{"connect", "message", "disconnect"}[e-1]
}
