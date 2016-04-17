// mergeQuestionsHandler.go merges one question into another.
package site

import (
	"encoding/json"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
)

// mergeQuestionsData is the data received from the request.
type mergeQuestionsData struct {
	QuestionId     string
	IntoQuestionId string
}

var mergeQuestionsHandler = siteHandler{
	URI:         "/mergeQuestions/",
	HandlerFunc: mergeQuestionsHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
		MinKarma:     200,
	},
}

func mergeQuestionsHandlerFunc(params *pages.HandlerParams) *pages.Result {
	u := params.U
	db := params.DB

	decoder := json.NewDecoder(params.R.Body)
	var data mergeQuestionsData
	err := decoder.Decode(&data)
	if err != nil {
		return pages.HandlerBadRequestFail("Couldn't decode json", err)
	}
	if !core.IsIdValid(data.QuestionId) || !core.IsIdValid(data.IntoQuestionId) {
		return pages.HandlerBadRequestFail("One of the ids is invalid", nil)
	}

	// Load the page
	pageMap := make(map[string]*core.Page)
	question := core.AddPageIdToMap(data.QuestionId, pageMap)
	intoQuestion := core.AddPageIdToMap(data.IntoQuestionId, pageMap)
	err = core.LoadPages(db, u, pageMap)
	if err != nil {
		return pages.HandlerErrorFail("Couldn't load questions", err)
	}
	if question == nil {
		return pages.HandlerBadRequestFail("Couldn't load the question", nil)
	}
	if intoQuestion == nil {
		return pages.HandlerBadRequestFail("Couldn't load the intoQuestion", nil)
	}

	// Begin the transaction.
	errMessage, err := db.Transaction(func(tx *database.Tx) (string, error) {
		statement := database.NewQuery(`
			UPDATE answers
			SET questionId=?`, data.IntoQuestionId).Add(`
			WHERE questionId=?`, data.QuestionId).ToTxStatement(tx)
		if _, err := statement.Exec(); err != nil {
			return "Couldn't update answers", err
		}

		statement = database.NewQuery(`
			UPDATE marks
			SET resolvedPageId=?`, data.IntoQuestionId).Add(`
			WHERE resolvedPageId=?`, data.QuestionId).ToTxStatement(tx)
		if _, err := statement.Exec(); err != nil {
			return "Couldn't update answers", err
		}
		return "", nil
	})
	if err != nil {
		return pages.HandlerErrorFail(errMessage, err)
	}

	return pages.StatusOK(nil)
}
