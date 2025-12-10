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
}
