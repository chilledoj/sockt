package sockt

func (r *Room[RoomID, ConnectionID]) Stop() {
	r.cancel()
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
