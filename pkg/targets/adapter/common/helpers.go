package common

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nknorg/nkn-sdk-go"
)

func ConvertNKNMessageToCloudevent(message nkn.Message) (cloudevents.Event, error) {
	var cloudEvent cloudevents.Event
	err := cloudEvent.DataAs(message.Data)
	if err != nil {
		return cloudevents.NewEvent(), err
	}

	return cloudEvent, nil
}
