package toolbox

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

type BaseEvent struct {
	EventId   string      `json:"@id"`
	EventType string      `json:"@type"`
	EventData interface{} `json:"@data"`
}

type EventMarshaller struct {
	Types map[string]interface{}
}

type EventHandler func(id string, eventType string, data interface{})

func (this *EventMarshaller) UnmarshallToType(input string) (string, string, interface{}, error) {
	var msg json.RawMessage
	env := BaseEvent{
		EventData: &msg,
	}

	buf := []byte(input)
	if err := json.Unmarshal(buf, &env); err != nil {
		return "", "", nil, err
	}

	var eventData interface{}
	if err := json.Unmarshal(msg, &eventData); err != nil {
		return env.EventId, env.EventType, nil, err
	} else {
		var realType = this.Types[env.EventType]
		err := mapstructure.Decode(eventData, &realType)
		if err == nil {
			return env.EventId, env.EventType, realType, nil
		} else {
			return env.EventId, env.EventType, nil, err
		}
	}
}

func (this *EventMarshaller) Marshall(eventType string, data interface{}) (string, string, error) {
	event := BaseEvent{
		EventId:   uuid.New().String(),
		EventType: eventType,
		EventData: data,
	}
	id := event.EventId
	res, err := json.Marshal(event)
	if err == nil {
		return id, string(res), nil
	} else {
		return "", "", err
	}
}
