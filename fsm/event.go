package fsm

type BasicEvent struct {
	name    string
	payload any
}

func NewBasicEvent(name string, payload any) *BasicEvent {
	return &BasicEvent{name: name, payload: payload}
}

func (e *BasicEvent) Name() string {
	return e.name
}

func (e *BasicEvent) Payload() any {
	return e.payload
}
