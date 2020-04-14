package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"time"

	"github.com/graphql-services/go-saga/graphqlorm"
)

const (
	QUERY_SURVEY_EXPORT = `
	query ($answerIDs: [ID!]) {
		result: surveyExport(
		  filter: { answerIDs: $answerIDs }
		) {
			items {
				survey{
					id
					name
				}
				fields {
					key
					title
				}
				rows {
					answer {
						id
						completed
						updatedAt
					}
					values {
						key
						value
					}
				}
			}
		}
	  }
	  `
)

type SurveyExportField struct {
	Key   string
	Title string
}
type SurveyExportRowValue struct {
	Key   string
	Value string
}
type SurveyExportRowAnswer struct {
	ID       string
	SurveyID string
}
type SurveyExportRow struct {
	Answer struct {
		ID        string
		Completed bool
		UpdatedAt time.Time
	}
	Values []SurveyExportRowValue
}
type SurveyExportItemSurvey struct {
	ID   string
	Name string
}
type SurveyExportItem struct {
	Survey *SurveyExportItemSurvey
	Fields []SurveyExportField
	Rows   []SurveyExportRow
}
type SurveyExport struct {
	Items []SurveyExportItem
}

type SurveyExportQuery struct {
	Result SurveyExport
}

func handleExport(ctx context.Context, client *graphqlorm.ORMClient, meta ExportMeta) (fileID string, err error) {
	surveyItems := []SurveyExportItem{}

	chunks := [][]string{}
	chunkSize := 100

	for i := 0; i < len(meta.AnswerIDs); i += chunkSize {
		end := i + chunkSize

		if end > len(meta.AnswerIDs) {
			end = len(meta.AnswerIDs)
		}

		chunks = append(chunks, meta.AnswerIDs[i:end])
	}

	for _, chunk := range chunks {
		var query SurveyExportQuery
		err = client.SendQuery(ctx, QUERY_SURVEY_EXPORT, map[string]interface{}{
			"answerIDs": chunk,
		}, &query)
		if err != nil {
			return
		}
		surveyItems = append(surveyItems, query.Result.Items...)
	}

	csv, err := buildCSV(ctx, surveyItems, meta)
	if err != nil {
		return
	}

	fmt.Println("final csv", string(csv))

	fileID, err = uploadCSVFile(ctx, csv, "survey-export.csv")

	return
}

func buildCSV(ctx context.Context, items []SurveyExportItem, meta ExportMeta) (csvContent []byte, err error) {
	records := [][]string{}

	values := map[string](map[string]string){}
	participantAnswerMap := map[string]string{}
	fields := []string{"participant"}
	fieldsMap := map[string]bool{}

	for i, answerID := range meta.AnswerIDs {
		participantID := meta.RowNames[i]
		values[participantID] = map[string]string{
			"participant": participantID,
		}
		participantAnswerMap[answerID] = participantID
	}

	for _, item := range items {
		for _, row := range item.Rows {
			for _, val := range row.Values {
				for i := 0; i < 100; i++ {
					valueKey := val.Key
					if i > 0 {
						valueKey = fmt.Sprintf("%s_%d", valueKey, i)
					}
					participantID := participantAnswerMap[row.Answer.ID]
					_, exists := values[participantID][valueKey]
					if !exists {
						values[participantID][valueKey] = val.Value
						if _, exists := fieldsMap[valueKey]; !exists {
							fields = append(fields, valueKey)
							fieldsMap[valueKey] = true
						}
						break
					} else {
						continue
					}
				}
			}
		}
	}

	records = append(records, fields)
	for _, value := range values {
		row := []string{}
		for _, field := range fields {
			row = append(row, value[field])
		}
		records = append(records, row)
	}

	buf := bytes.NewBufferString("")
	w := csv.NewWriter(buf)
	w.WriteAll(records) // calls Flush internally

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}

	csvContent = buf.Bytes()
	return
}
