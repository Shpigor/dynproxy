package dynproxy

type EventRouter interface {
	Process(key string, event *Event) error
}

var eventRouter EventRouter
