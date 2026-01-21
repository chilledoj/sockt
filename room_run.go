package sockt

func (r *Room[RoomID, ConnectionID]) Stop() {
	// Signal all goroutines to stop
	r.cancel()

	// Close all connections to unblock any read operations
	r.mu.Lock()
	for connId, conn := range r.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				r.lg.Printf("error closing connection %v during stop: %v\n", connId, err)
			}
		}
		delete(r.connections, connId)
	}
	r.mu.Unlock()

	// Wait for all goroutines to finish
	r.wg.Wait()
}

func (r *Room[RoomID, ConnectionID]) Run() {
	r.lg.Println("run started")
loop:
	for {
		select {
		// TODO: OTHER STUFF COULD BE DONE HERE.
		case <-r.ctx.Done():
			break loop
		}
	}
	r.lg.Println("run finished")

}
