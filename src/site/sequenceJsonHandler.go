// sequenceJsonHandler.go returns the sequence of pages needed for understanding a page
package site

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
)

var sequenceHandler = siteHandler{
	URI:         "/json/sequence/",
	HandlerFunc: sequenceJsonHandler,
}

type sequenceJsonData struct {
	PageId int64 `json:",string"`
}

type sequencePart struct {
	// I want to understand PageId
	PageId int64 `json:"pageId,string"`
	// To understand it, I will read TaughtById
	TaughtById int64 `json:"taughtById,string"`
	// To understand TaughtById, I need to meet th following Requirements
	Requirements []*sequencePart `json:"requirements"`
}

func sequenceJsonHandler(params *pages.HandlerParams) *pages.Result {
	u := params.U
	db := params.DB

	// Decode data
	var data sequenceJsonData
	err := json.NewDecoder(params.R.Body).Decode(&data)
	if err != nil {
		return pages.HandlerBadRequestFail("Couldn't decode request", err)
	}
	if data.PageId < 0 {
		return pages.HandlerBadRequestFail("Need a valid pageId", nil)
	}

	returnData := newHandlerData(true)
	returnData.User = u

	// Check if the user already has this requirement
	hasMastery := false
	row := database.NewQuery(`
		SELECT ifnull(max(has),false)
		FROM userMasteryPairs
		WHERE userId=?`, u.Id).Add(`AND masteryId=?`, data.PageId).ToStatement(db).QueryRow()
	_, err = row.Scan(&hasMastery)
	if err != nil {
		return pages.HandlerErrorFail("Error while checking if already knows", err)
	}

	// Track which requirements we need to process
	requirementIds := make([]interface{}, 0)
	if !hasMastery {
		requirementIds = append(requirementIds, data.PageId)
	}

	// Create the sequence root
	sequence := &sequencePart{PageId: data.PageId}
	sequenceMap := make(map[int64]*sequencePart)
	sequenceMap[data.PageId] = sequence

	// What to load for the pages
	loadOptions := (&core.PageLoadOptions{}).Add(core.TitlePlusLoadOptions)
	core.AddPageToMap(data.PageId, returnData.PageMap, loadOptions)

	// Recursively find which pages the user has to read
	for maxCount := 0; len(requirementIds) > 0 && maxCount < 20; maxCount++ {
		// Track which taughtBy ids we load, so we can load requirements for them
		taughtByIds := make([]interface{}, 0)

		// Load which pages teach the requirements
		rows := database.NewQuery(`
			SELECT pp.parentId,pp.childId
			FROM pagePairs AS pp
			WHERE pp.parentId IN`).AddArgsGroup(requirementIds).Add(`
				AND pp.type=?`, core.SubjectPagePairType).Add(`
			GROUP BY 1`).ToStatement(db).Query()
		err = rows.Process(func(db *database.DB, rows *database.Rows) error {
			var parentId, childId int64
			err := rows.Scan(&parentId, &childId)
			if err != nil {
				return fmt.Errorf("Failed to scan: %v", err)
			}
			sequenceMap[parentId].TaughtById = childId
			taughtByIds = append(taughtByIds, childId)
			core.AddPageToMap(childId, returnData.PageMap, loadOptions)
			return nil
		})
		if err != nil {
			return pages.HandlerErrorFail("Error while loading subjects", err)
		}
		if len(taughtByIds) <= 0 {
			break
		}

		// Load the requirements for the subjects
		requirementIds = make([]interface{}, 0)
		rows = database.NewQuery(`
			SELECT pp.parentId,pp.childId,mp.has
			FROM pagePairs AS pp
			LEFT JOIN userMasteryPairs AS mp
			ON (pp.parentId=mp.masteryId AND mp.userId=?)`, u.Id).Add(`
			WHERE pp.childId IN`).AddArgsGroup(taughtByIds).Add(`
				AND pp.type=?`, core.RequirementPagePairType).ToStatement(db).Query()
		err = rows.Process(func(db *database.DB, rows *database.Rows) error {
			var parentId, childId int64
			var has sql.NullBool
			err := rows.Scan(&parentId, &childId, &has)
			if err != nil {
				return fmt.Errorf("Failed to scan: %v", err)
			}
			if has.Valid && has.Bool {
				return nil
			}

			for _, part := range sequenceMap {
				if part.TaughtById != childId {
					continue
				}
				requirementIds = append(requirementIds, parentId)
				requirementPart, ok := sequenceMap[parentId]
				if !ok {
					requirementPart = &sequencePart{PageId: parentId}
				}
				part.Requirements = append(part.Requirements, requirementPart)
				sequenceMap[parentId] = requirementPart
				core.AddPageToMap(parentId, returnData.PageMap, loadOptions)
			}
			return nil
		})
		if err != nil {
			return pages.HandlerErrorFail("Error while loading requirements", err)
		}
	}

	// Load pages
	err = core.ExecuteLoadPipeline(db, u, returnData.PageMap, returnData.UserMap, returnData.MasteryMap)
	if err != nil {
		return pages.HandlerErrorFail("Pipeline error", err)
	}

	returnData.ResultMap["sequence"] = sequence
	return pages.StatusOK(returnData.toJson())
}