package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/graphql-services/go-saga/graphqlorm"
	handler "github.com/jakubknejzlik/cloudevents-lambda-handler"
	"github.com/novacloudcz/graphql-orm/events"
)

type ExportMeta struct {
	AnswerIDs      []string
	RowNames       []string
	PublicSurveyId *string
}

func receiver(e cloudevents.Event) error {
	ctx := context.Background()

	var ormEvent events.Event
	err := e.DataAs(&ormEvent)
	if err != nil {
		return err
	}

	fmt.Println("event", ormEvent.Entity, ormEvent.Type, ormEvent.ChangedColumns())

	ormURL := os.Getenv("GRAPHQL_ORM_URL")
	if ormURL == "" {
		return fmt.Errorf("Missing required GRAPHQL_ORM_URL envvar")
	}
	ormClient := graphqlorm.NewClient(ormURL)

	err = updateExportState(ctx, ormClient, ormEvent, "PROCESSING", 0, nil)
	if err != nil {
		return err
	}

	var metadata string
	err = ormEvent.Change("metadata").NewValueAs(&metadata)
	if err != nil {
		return err
	}

	var meta ExportMeta
	err = json.Unmarshal([]byte(metadata), &meta)
	if err != nil {
		return err
	}

	fileID, err := handleExport(ctx, ormClient, meta, func(progress float32) {
		updateExportState(ctx, ormClient, ormEvent, "PROCESSING", progress, nil)
	})
	if err != nil {
		fmt.Println("error processing", err)
		err = updateExport(ctx, ormClient, ormEvent, map[string]interface{}{
			"state":            "ERROR",
			"errorDescription": err.Error(),
		})
		if err != nil {
			return err
		}
		return nil
	}

	return updateExportState(ctx, ormClient, ormEvent, "COMPLETED", 1, &fileID)
}

func updateExportState(ctx context.Context, c *graphqlorm.ORMClient, ormEvent events.Event, state string, progress float32, fileID *string) (err error) {
	input := map[string]interface{}{
		"state":    state,
		"progress": progress,
	}

	if fileID != nil {
		input["fileId"] = *fileID
	}
	return updateExport(ctx, c, ormEvent, input)
}
func updateExport(ctx context.Context, c *graphqlorm.ORMClient, ormEvent events.Event, input map[string]interface{}) (err error) {
	fmt.Println("update entity input", input)
	_, err = c.UpdateEntity(ctx, graphqlorm.UpdateEntityOptions{
		Entity:   ormEvent.Entity,
		EntityID: ormEvent.EntityID,
		Input:    input,
	})
	return
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
