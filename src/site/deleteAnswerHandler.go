// deleteAnswerHandler.go deletes an answer to a question

package site

import (
	"encoding/json"
	"net/http"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/tasks"
)

// deleteAnswerData contains data given to us in the request.
type deleteAnswerData struct {
	AnswerID string
}

var deleteAnswerHandler = siteHandler{
	URI:         "/deleteAnswer/",
	HandlerFunc: deleteAnswerHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

// deleteAnswerHandlerFunc handles requests to create/update a like.
func deleteAnswerHandlerFunc(params *pages.HandlerParams) *pages.Result {
	u := params.U
	c := params.C
	db := params.DB

	var data deleteAnswerData
	decoder := json.NewDecoder(params.R.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode json", err).Status(http.StatusBadRequest)
	}

	// Load the existing answer
	answer, err := core.LoadAnswer(db, data.AnswerID)
	if err != nil {
		return pages.Fail("Couldn't load the existing answer", err)
	}

	err2 := db.Transaction(func(tx *database.Tx) sessions.Error {
		// Delete the answer
		statement := database.NewQuery(`
			DELETE FROM answers WHERE id=?`, data.AnswerID).ToStatement(db).WithTx(tx)
		_, err = statement.Exec()
		if err != nil {
			return sessions.NewError("Couldn't insert into DB", err)
		}

		// Update change logs
		hashmap := make(database.InsertMap)
		hashmap["pageId"] = answer.QuestionID
		hashmap["userId"] = u.ID
		hashmap["createdAt"] = database.Now()
		hashmap["type"] = core.AnswerChangeChangeLog
		hashmap["auxPageId"] = answer.AnswerPageID
		hashmap["oldSettingsValue"] = "old"
		statement = tx.DB.NewInsertStatement("changeLogs", hashmap).WithTx(tx)
		resp, err := statement.Exec()
		if err != nil {
			return sessions.NewError("Couldn't add to changeLogs", err)
		}
		changeLogID, err := resp.LastInsertId()
		if err != nil {
			return sessions.NewError("Couldn't get changeLog id", err)
		}

		// Insert updates
		var task tasks.NewUpdateTask
		task.UserID = u.ID
		task.GoToPageID = answer.AnswerPageID
		task.SubscribedToID = answer.QuestionID
		task.UpdateType = core.ChangeLogUpdateType
		task.ChangeLogID = changeLogID
		if err := tasks.Enqueue(c, &task, nil); err != nil {
			return sessions.NewError("Couldn't enqueue a task: %v", err)
		}

		return nil
	})
	if err2 != nil {
		return pages.FailWith(err2)
	}

	return pages.Success(nil)
}
