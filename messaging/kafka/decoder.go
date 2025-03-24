package kafka

import (
	"encoding/json"
	"errors"

	"github.com/rluders/gofsm/fsm"
)

type EventCodec interface {
	Decode(value []byte) (fsm.Event, error)
}

type EventKafka struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
}

type JSONEventCodec struct{}

func (c *JSONEventCodec) Decode(value []byte) (fsm.Event, error) {
	var evt EventKafka
	if err := json.Unmarshal(value, &evt); err != nil {
		return nil, err
	}
	if evt.Name == "" {
		return nil, errors.New("invalid event: name is empty")
	}
	return fsm.NewBasicEvent(evt.Name, evt.Payload), nil
}
