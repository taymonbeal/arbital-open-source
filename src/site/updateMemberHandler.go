// updateMemberHandler.go adds a new member for a group.
package site

import (
	"encoding/json"

	"zanaduu3/src/pages"
)

// updateMemberData contains data given to us in the request.
type updateMemberData struct {
	GroupId       int64 `json:",string"`
	UserId        int64 `json:",string"`
	CanAddMembers bool
	CanAdmin      bool
}

func updateMemberHandler(params *pages.HandlerParams) *pages.Result {
	db := params.DB
	u := params.U

	decoder := json.NewDecoder(params.R.Body)
	var data updateMemberData
	err := decoder.Decode(&data)
	if err != nil {
		return pages.HandlerBadRequestFail("Couldn't decode json", err)
	}
	if data.GroupId <= 0 || data.UserId <= 0 {
		return pages.HandlerBadRequestFail("GroupId and UserId have to be set", nil)
	}

	if !u.IsLoggedIn {
		return pages.HandlerForbiddenFail("Not logged in", nil)
	}

	// Check to see if this user can add members.
	var canAdmin bool
	row := db.NewStatement(`
		SELECT canAdmin
		FROM groupMembers
		WHERE userId=? AND groupId=? AND canAddMembers
		`).QueryRow(u.Id, data.GroupId)
	found, err := row.Scan(&canAdmin)
	if err != nil {
		return pages.HandlerErrorFail("Couldn't check for a group member", err)
	} else if !found {
		return pages.HandlerForbiddenFail("You don't have the permission to add a user", nil)
	}

	// Check if the target user exists and get their permissions
	var targetCanAdmin bool
	row = db.NewStatement(`
		SELECT canAdmin
		FROM groupMembers
		WHERE userId=? AND groupId=?
		`).QueryRow(data.UserId, data.GroupId)
	found, err = row.Scan(&targetCanAdmin)
	if err != nil {
		return pages.HandlerErrorFail("Couldn't check for target group member", err)
	} else if !found {
		return pages.HandlerForbiddenFail("Target member not found", nil)
	}

	// Admin's can't change property on non-admin.
	if !canAdmin && targetCanAdmin {
		data.CanAdmin = targetCanAdmin
	}
	data.CanAddMembers = data.CanAddMembers || data.CanAdmin

	hashmap := make(map[string]interface{})
	hashmap["userId"] = data.UserId
	hashmap["groupId"] = data.GroupId
	hashmap["canAddMembers"] = data.CanAddMembers
	hashmap["canAdmin"] = data.CanAdmin
	statement := db.NewReplaceStatement("groupMembers", hashmap)
	if _, err = statement.Exec(); err != nil {
		return pages.HandlerErrorFail("Couldn't update a member", err)
	}
	return pages.StatusOK(nil)
}
