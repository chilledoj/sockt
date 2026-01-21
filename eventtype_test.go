package sockt

import "testing"

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		name     string
		event    EventType
		expected string
	}{
		{
			name:     "EventConnect",
			event:    EventConnect,
			expected: "connect",
		},
		{
			name:     "EventMessage",
			event:    EventMessage,
			expected: "message",
		},
		{
			name:     "EventDisconnect",
			event:    EventDisconnect,
			expected: "disconnect",
		},
		{
			name:     "Unknown negative",
			event:    EventType(-1),
			expected: "unknown",
		},
		{
			name:     "Unknown zero",
			event:    EventType(0),
			expected: "unknown",
		},
		{
			name:     "Unknown high value",
			event:    EventType(99),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.String(); got != tt.expected {
				t.Errorf("EventType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventTypeValues(t *testing.T) {
	// Ensure the iota values are as expected
	if EventConnect != 1 {
		t.Errorf("Expected EventConnect to be 1, got %d", EventConnect)
	}
	if EventMessage != 2 {
		t.Errorf("Expected EventMessage to be 2, got %d", EventMessage)
	}
	if EventDisconnect != 3 {
		t.Errorf("Expected EventDisconnect to be 3, got %d", EventDisconnect)
	}
}
