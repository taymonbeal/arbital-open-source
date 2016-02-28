// updatePageObject.go handles request to update the value of a page object
package site

import (
	"encoding/json"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
)

// updatePageObject contains the data we get in the request.
type updatePageObject struct {
	PageId string
	Edit   int
	Object string
	Value  string
}

var updatePageObjectHandler = siteHandler{
	URI:         "/updatePageObject/",
	HandlerFunc: updatePageObjectHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

func updatePageObjectHandlerFunc(params *pages.HandlerParams) *pages.Result {
	decoder := json.NewDecoder(params.R.Body)
	var data updatePageObject
	err := decoder.Decode(&data)
	if err != nil {
		return pages.HandlerBadRequestFail("Couldn't decode json", err)
	}

	return updatePageObjectInternalHandlerFunc(params, &data)
}

func updatePageObjectInternalHandlerFunc(params *pages.HandlerParams, data *updatePageObject) *pages.Result {
	db := params.DB
	u := params.U

	if !core.IsIdValid(data.PageId) {
		return pages.HandlerBadRequestFail("Invalid page id", nil)
	}
	if data.Object == "" {
		return pages.HandlerBadRequestFail("Object alias isn't set", nil)
	}

	hashmap := make(map[string]interface{})
	hashmap["userId"] = u.Id
	hashmap["pageId"] = data.PageId
	hashmap["edit"] = data.Edit
	hashmap["object"] = data.Object
	hashmap["value"] = data.Value
	hashmap["createdAt"] = database.Now()
	hashmap["updatedAt"] = database.Now()
	statement := db.NewInsertStatement("userPageObjectPairs", hashmap, "edit", "value", "updatedAt")
	if _, err := statement.Exec(); err != nil {
		return pages.HandlerErrorFail("Couldn't update a page object", err)
	}

	return pages.StatusOK(nil)
}