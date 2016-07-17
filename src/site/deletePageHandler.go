// deletePageHandler.go handles requests for deleting a page.
package site

import (
	"encoding/json"
	"net/http"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/elastic"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/tasks"
)

// deletePageData is the data received from the request.
type deletePageData struct {
	PageId string

	// Set internally
	GenerateUpdate bool `json:"-"`
}

var deletePageHandler = siteHandler{
	URI:         "/deletePage/",
	HandlerFunc: deletePageHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

// deletePageHandlerFunc handles requests for deleting a page.
func deletePageHandlerFunc(params *pages.HandlerParams) *pages.Result {
	decoder := json.NewDecoder(params.R.Body)
	var data deletePageData
	err := decoder.Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode json", err).Status(http.StatusBadRequest)
	}
	if !core.IsIdValid(data.PageId) {
		return pages.Fail("PageId isn't set", nil).Status(http.StatusBadRequest)
	}
	return deletePageInternalHandlerFunc(params, &data)
}

func deletePageInternalHandlerFunc(params *pages.HandlerParams, data *deletePageData) *pages.Result {
	db := params.DB
	u := params.U

	// Load the page
	page, err := core.LoadFullEdit(db, data.PageId, u, nil)
	if err != nil {
		return pages.Fail("Couldn't load page", err)
	}
	if page.IsDeleted || page.Type == "" {
		// Looks like there is no need to delete this page.
		return pages.Success(nil)
	}
	if page.Type == core.GroupPageType || page.Type == core.DomainPageType {
		if !u.IsAdmin {
			return pages.Fail("Have to be an admin to delete a group/domain", nil).Status(http.StatusForbidden)
		}
	}
	if page.Type == core.CommentPageType && u.ID != page.PageCreatorId {
		if !u.IsAdmin {
			return pages.Fail("Have to be an admin to delete someone else's comment", nil).Status(http.StatusForbidden)
		}
	}

	// Make sure the user has the right permissions to delete this page
	if !page.Permissions.Delete.Has {
		return pages.Fail(page.Permissions.Delete.Reason, nil).Status(http.StatusBadRequest)
	}

	err2 := db.Transaction(func(tx *database.Tx) sessions.Error {
		data.GenerateUpdate = true
		return deletePageTx(tx, params, data, page)
	})
	if err2 != nil {
		return pages.FailWith(err2)
	}

	return pages.Success(nil)
}

func deletePageTx(tx *database.Tx, params *pages.HandlerParams, data *deletePageData, page *core.Page) sessions.Error {
	c := params.C

	// Clear the current edit in pages
	statement := tx.DB.NewStatement("UPDATE pages SET isLiveEdit=false WHERE pageId=? AND isLiveEdit").WithTx(tx)
	if _, err := statement.Exec(data.PageId); err != nil {
		return sessions.NewError("Couldn't update isLiveEdit for old edits", err)
	}

	// Set isDeleted in pageInfos
	statement = tx.DB.NewStatement("UPDATE pageInfos SET isDeleted=true WHERE pageId=?").WithTx(tx)
	if _, err := statement.Exec(data.PageId); err != nil {
		return sessions.NewError("Couldn't set isDeleted for deleted page", err)
	}

	// Update change log
	hashmap := make(database.InsertMap)
	hashmap["pageId"] = data.PageId
	hashmap["userId"] = params.U.ID
	hashmap["createdAt"] = database.Now()
	hashmap["type"] = core.DeletePageChangeLog
	statement = tx.DB.NewInsertStatement("changeLogs", hashmap).WithTx(tx)
	result, err := statement.Exec()
	if err != nil {
		return sessions.NewError("Couldn't update change logs", err)
	}
	changeLogId, err := result.LastInsertId()
	if err != nil {
		return sessions.NewError("Couldn't get changeLogId", err)
	}

	if data.GenerateUpdate && page.Type != core.CommentPageType {
		// Generate "delete" update for users who are subscribed to this page.
		var updateTask tasks.NewUpdateTask
		updateTask.UserId = params.U.ID
		updateTask.GoToPageId = data.PageId
		updateTask.SubscribedToId = data.PageId
		updateTask.UpdateType = core.ChangeLogUpdateType
		updateTask.ChangeLogId = changeLogId

		if err := tasks.Enqueue(c, &updateTask, nil); err != nil {
			return sessions.NewError("Couldn't enqueue changeLog task", err)
		}
	}

	// NOTE: now that we've done an undoable action, we can no longer return failure

	// Delete it from the elastic index
	if page.WasPublished {
		err := elastic.DeletePageFromIndex(c, data.PageId)
		if err != nil {
			c.Errorf("Failed to update index: %v", err)
		}
	}

	// Create a task to propagate the domain change to all children
	var task tasks.PropagateDomainTask
	task.PageId = data.PageId
	if err := tasks.Enqueue(params.C, &task, nil); err != nil {
		c.Errorf("Couldn't enqueue a task: %v", err)
	}

	return nil
}
