package main

import (
	"context"

	"github.com/graphql-services/go-saga/graphqlorm"
)

const (
	QUERY_SURVEY_ASSIGNMENTS_EXPORT = `
	query publicSurveyAssignments($filter: PublicSurveyAssignmentFilterType) {
		result: publicSurveyAssignments(filter:$filter,limit:9999) {
			items {
				id answerId participantId
			} 
		} 
	}
	`
)

type SurveyAssignmentExportItem struct {
	ID            string
	AnswerID      *string
	ParticipantID *string
}
type SurveyAssignmentExport struct {
	Items []SurveyAssignmentExportItem
}

type SurveyAssignmentExportQuery struct {
	Result SurveyAssignmentExport
}

func fetchAnswerIDsByPublicSurveID(ctx context.Context, client *graphqlorm.ORMClient, publicSurveyID string) (ids []string, participants []string, err error) {
	var query SurveyAssignmentExportQuery
	err = client.SendQuery(ctx, QUERY_SURVEY_ASSIGNMENTS_EXPORT, map[string]interface{}{
		"filter": map[string]interface{}{
			"participant": map[string]interface{}{
				"publicSurveyId": publicSurveyID,
			},
		},
	}, &query)

	if err != nil {
		return
	}

	ids = []string{}
	participants = []string{}
	for _, item := range query.Result.Items {
		if item.AnswerID != nil {
			if item.ParticipantID != nil {
				participants = append(participants, *item.ParticipantID)
			} else {
				participants = append(participants, *item.AnswerID)
			}
			ids = append(ids, *item.AnswerID)
		}

	}

	return
}
