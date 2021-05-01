package toolbox

import (
	"math/big"
	"testing"
)

func TestDeserialize(t *testing.T) {
	type test_struct struct {
		K string     `json:"k"`
		V *big.Float `json:"v"`
	}

	eventType := "test"
	marshaller := NewEventMarshaller(map[string]interface{}{
		eventType: test_struct{},
	})

	inputStruct := test_struct{
		K: "1",
		V: big.NewFloat(100.2),
	}
	sourceEventId, input, _ := marshaller.Marshall(eventType, inputStruct)

	resultEventId, _, result, _ := marshaller.UnmarshallToType(input)
	if sourceEventId != resultEventId {
		t.Errorf("EventId validation failed. Expected %v, got %v", sourceEventId, resultEventId)
	}
	output, success := result.(test_struct)
	if !success || output.K != inputStruct.K || output.V.String() != inputStruct.V.String() {
		t.Error("Invalid inmarshalling")
	}
}
