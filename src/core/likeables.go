// Contains helpers for likeable things, like pages and changeLogs.
package core

import (
	"fmt"

	"zanaduu3/src/database"
)

const (
	// Possible likeable types
	ChangeLogLikeableType      = "changeLog"
	PageLikeableType           = "page"
	RedLinkLikeableType        = "redLink"
	ContentRequestLikeableType = "contentRequest"
)

// Get the likeableId of the given likeable. If it doesn't have one, create one for it.
// Returns the likeableId of the likeable.
func GetOrCreateLikeableId(tx *database.Tx, likeableType string, id string) (int64, error) {
	// Figure out which table to look into
	tableName, idField, err := GetTableAndIdFieldForLikeable(likeableType)
	if err != nil {
		return 0, err
	}

	// Look up the likeable
	var likeableId int64
	row := tx.DB.NewStatement(`
		SELECT likeableId
		FROM ` + tableName + `
		WHERE ` + idField + `=?`).WithTx(tx).QueryRow(id)
	_, err = row.Scan(&likeableId)
	if err != nil {
		return 0, fmt.Errorf("Couldn't look up likeableId: %v", err)
	}

	// If it already has a likeableId, return that
	if likeableId != 0 {
		return likeableId, nil
	}

	// Otherwise, insert a new likeableId
	result, err := tx.DB.NewInsertStatement("likeableIds", make(database.InsertMap)).WithTx(tx).Exec()
	if err != nil {
		return 0, fmt.Errorf("Couldn't insert new likeableId", err)
	}
	likeableId, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("Couldn't retrieve new likeableId", err)
	}

	// Update the likeable with the likeableId
	hashmap := make(database.InsertMap)
	hashmap[idField] = id
	hashmap["likeableId"] = likeableId
	result, err = tx.DB.NewInsertStatement(tableName, hashmap, "likeableId").WithTx(tx).Exec()
	if err != nil {
		return 0, err
	}

	return likeableId, nil
}

// Get the name of the table and id field for the given likeableType.
func GetTableAndIdFieldForLikeable(likeableType string) (string, string, error) {
	switch likeableType {
	case PageLikeableType:
		return "pageInfos", "pageId", nil
	case ChangeLogLikeableType:
		return "changeLogs", "id", nil
	case RedLinkLikeableType:
		return "redLinks", "alias", nil
	case ContentRequestLikeableType:
		return "contentRequests", "id", nil
	default:
		return "", "", fmt.Errorf("invalid likeableType")
	}
}

// Check if the given likeableType is valid.
func IsValidLikeableType(likeableType string) bool {
	switch likeableType {
	case PageLikeableType, ChangeLogLikeableType, RedLinkLikeableType, ContentRequestLikeableType:
		return true
	default:
		return false
	}
}
