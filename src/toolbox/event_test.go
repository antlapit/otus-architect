package toolbox

import (
	"testing"
)

func TestDeserialize(t *testing.T) {
	type test_struct struct {
		K string `json:"k"`
	}

	eventType := "test"
	marshaller := &EventMarshaller{
		Types: map[string]interface{}{
			eventType: test_struct{},
		},
	}

	inputStruct := test_struct{
		K: "1",
	}
	sourceEventId, input, _ := marshaller.Marshall(eventType, inputStruct)

	resultEventId, _, result, _ := marshaller.UnmarshallToType(input)
	if sourceEventId != resultEventId {
		t.Errorf("EventId validation failed. Expected %v, got %v", sourceEventId, resultEventId)
	}
	output, success := result.(test_struct)
	if !success || output != inputStruct {
		t.Error("Invalid inmarshalling")
	}
}
