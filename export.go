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
						text
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
	Text  string
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
	var query SurveyExportQuery
	err = client.SendQuery(ctx, QUERY_SURVEY_EXPORT, map[string]interface{}{
		"answerIDs": meta.AnswerIDs,
	}, &query)
	if err != nil {
		return
	}

	csv, err := buildCSV(ctx, query.Result, meta)
	if err != nil {
		return
	}

	fmt.Println("final csv", string(csv))

	fileID, err = uploadCSVFile(ctx, csv, "survey-export.csv")

	return
}

func buildCSV(ctx context.Context, se SurveyExport, meta ExportMeta) (csvContent []byte, err error) {
	records := [][]string{}

	var rowNamesMap *map[string]string
	if meta.RowNames != nil {
		rowNamesMap = &map[string]string{}
		for i, answerID := range meta.AnswerIDs {
			(*rowNamesMap)[answerID] = meta.RowNames[i]
		}
	}

	for _, item := range se.Items {
		_records, _err := buildCSVRow(ctx, item, rowNamesMap)
		if _err != nil {
			err = _err
			return
		}
		records = append(records, _records...)
		records = append(records, []string{})
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
func buildCSVRow(ctx context.Context, item SurveyExportItem, rowNamesMap *map[string]string) (records [][]string, err error) {
	for _, field := range item.Fields {
		fmt.Println("field: ", field.Key, "=>", field.Title)
	}

	header := []string{}
	hasRowNames := rowNamesMap != nil
	if hasRowNames {
		header = append(header, item.Survey.Name)
	}

	header = append(header, "completed", "last update")

	for _, field := range item.Fields {
		header = append(header, field.Key)
	}

	records = [][]string{
		header,
	}
	for _, row := range item.Rows {
		values := map[string]string{}

		for _, val := range row.Values {
			if val.Text != "" {
				values[val.Key] = val.Text
			} else {
				values[val.Key] = val.Value
			}
		}
		rowValues := []string{}

		if hasRowNames {
			rowName, ok := (*rowNamesMap)[row.Answer.ID]
			if ok {
				rowValues = append(rowValues, rowName)
			} else {
				rowValues = append(rowValues, "–")
			}
		}

		if row.Answer.Completed {
			rowValues = append(rowValues, "✓")
		} else {
			rowValues = append(rowValues, "✗")
		}
		rowValues = append(rowValues, row.Answer.UpdatedAt.Format("2006-01-02 15:04:05"))

		for _, field := range item.Fields {
			rowValues = append(rowValues, values[field.Key])
		}
		records = append(records, rowValues)
	}

	return
}
