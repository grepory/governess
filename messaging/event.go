package messaging

import (
	"encoding/json"

	"github.com/bitly/go-nsq"
)

type EventInterface interface {
	Ack()
	Nack()
	Type() string
	Body() string
}

type Event struct {
	MessageType string `json:"type"`
	MessageBody string `json:"event"`
	message     *nsq.Message
}

func NewEvent(msg *nsq.Message) (EventInterface, error) {
	e := &Event{message: msg}
	err := json.Unmarshal(msg.Body, e)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return e, nil
}

func (e *Event) Nack() {
	e.message.Requeue(0)
}

func (e *Event) Ack() {
	e.message.Finish()
}

func (e *Event) Type() string {
	return e.MessageType
}

func (e *Event) Body() string {
	return e.MessageBody
}
