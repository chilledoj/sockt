package sockt

// RoomStatus represents the current operational state of a Room.
// It determines which players can join the room and how player connections are managed.
type RoomStatus int8

const (
	// Inactive indicates the room is not operational and no players can join.
	// This status is typically used when a room is being shut down or hasn't been initialized yet.
	Inactive RoomStatus = iota - 1

	// Open indicates the room is accepting new player connections.
	// Any player can join an open room, subject to other room-specific rules.
	Open

	// Locked indicates the room is operational but not accepting new players.
	// Only players who were previously connected can rejoin a locked room.
	// When a room is locked, disconnected players may be cleaned up immediately.
	Locked
)

// String returns a human-readable representation of the RoomStatus.
// This is useful for logging and debugging purposes.
func (r RoomStatus) String() string {
	switch r {
	case Inactive:
		return "Inactive"
	case Open:
		return "Open"
	case Locked:
		return "Locked"
	default:
		return "Unknown"
	}
}
