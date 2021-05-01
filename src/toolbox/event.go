package toolbox

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"math/big"
	"reflect"
)

type BaseEvent struct {
	EventId   string      `json:"@id"`
	EventType string      `json:"@type"`
	EventData interface{} `json:"@data"`
}

type EventMarshaller struct {
	Types map[string]interface{}
}

func NewEventMarshaller(types map[string]interface{}) *EventMarshaller {
	return &EventMarshaller{Types: types}
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
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				ToBigFloatHookFunc(),
			),
			Result: &realType,
		})
		err = decoder.Decode(eventData)
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

func ToBigFloatHookFunc() mapstructure.DecodeHookFunc {
	return func(
		from reflect.Type,
		to reflect.Type,
		data interface{}) (interface{}, error) {
		if to != reflect.TypeOf(big.Float{}) {
			return data, nil
		}

		switch from.Kind() {
		case reflect.String:
			r, _, err := big.NewFloat(0).Parse(data.(string), 10)
			return r, err
		case reflect.Float64:
			return big.NewFloat(data.(float64)), nil
		case reflect.Int64:
			return big.NewFloat(float64(data.(int64))), nil
		default:
			return data, nil
		}
		// Convert it by parsing
	}
}

type EventProcessor interface {
	ProcessEvent(id string, eventType string, data interface{})
}
