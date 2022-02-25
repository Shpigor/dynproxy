package dynproxy

type DarwinPoller struct {
	eventBufferSize int
	eventQueueSize  int
	fd              int
	lockOSThread    bool
	timeout         int
}

func openPoller(eventsBufferSize int) (Poller, error) {
	return nil, nil
}
