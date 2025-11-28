package sockt

type Event[ConnectionID comparable] struct {
	Type    EventType
	Subject ConnectionID
	Data    []byte
}
