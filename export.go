package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"

	"github.com/graphql-services/go-saga/graphqlorm"
)

const (
	QUERY_SURVEY_EXPORT = `
	query ($answerIDs: [ID!]) {
		result: surveyExport(
		  filter: { answerIDs: $answerIDs },
		  limit: 500
		) {
		  fields {
			key
			title
		  }
		  rows{
			answer{
			  id
			}
			values {
			  key
			  value
			  text
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
type SurveyExportRow struct {
	Values []SurveyExportRowValue
}
type SurveyExport struct {
	Fields []SurveyExportField
	Rows   []SurveyExportRow
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
	for _, field := range se.Fields {
		fmt.Println("field: ", field.Key, "=>", field.Title)
	}

	header := []string{}
	hasRowNames := meta.RowNames != nil
	if hasRowNames {
		header = append(header, "")
	}

	for _, field := range se.Fields {
		header = append(header, field.Key)
	}

	records := [][]string{
		header,
	}
	for i, row := range se.Rows {
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
			if len(meta.RowNames) > i {
				rowValues = append(rowValues, meta.RowNames[i])
			} else {
				rowValues = append(rowValues, "–")
			}
		}

		for _, field := range se.Fields {
			rowValues = append(rowValues, values[field.Key])
		}
		records = append(records, rowValues)
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
