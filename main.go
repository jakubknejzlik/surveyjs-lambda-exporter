package main

import (
	"fmt"
	"os"

	cloudevents "github.com/cloudevents/sdk-go"
	handler "github.com/jakubknejzlik/cloudevents-lambda-handler"
	"github.com/novacloudcz/graphql-orm/events"
)

func receiver(e cloudevents.Event) error {
	var ormEvent events.Event
	err := e.DataAs(&ormEvent)
	if err != nil {
		return err
	}

	fmt.Println("event", ormEvent.Entity, ormEvent.Type, ormEvent.ChangedColumns())

	// if ormEvent.Type == events.EventTypeCreated && ormEvent.Entity == "Project" {
	// 	var name string
	// 	ormEvent.Change("name").NewValueAs(&name)
	// 	return sendSlackMessage(getenv("SLACK_CHANNEL", "test"), fmt.Sprintf(":tada: New project created: `%s` https://admin.reportingdokapsy.cz/projects/%s", name, ormEvent.EntityID))
	// }

	return nil
}

func main() {
	h := handler.NewCloudEventsLambdaHandler(receiver)
	h.Start()
}

func getenv(name, defaultValue string) string {
	v := os.Getenv(name)
	if v == "" {
		v = defaultValue
	}
	return v
}
