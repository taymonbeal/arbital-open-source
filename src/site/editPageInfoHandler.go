// editPageInfoHandler.go contains the handler for editing pageInfo data.

package site

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/tasks"
)

// editPageInfoData contains parameters passed in.
type editPageInfoData struct {
	PageID           string
	Type             string
	HasVote          bool
	VoteType         string
	VotesAnonymous   bool
	SeeDomainID      string
	EditDomainID     string
	SubmitToDomainID string
	Alias            string // if empty, leave the current one
	SortChildrenBy   string
	IndirectTeacher  bool
	IsEditorComment  bool
	ExternalUrl      string
}

var editPageInfoHandler = siteHandler{
	URI:         "/editPageInfo/",
	HandlerFunc: editPageInfoHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

// editPageInfoHandlerFunc handles requests to create a new page.
func editPageInfoHandlerFunc(params *pages.HandlerParams) *pages.Result {
	c := params.C
	db := params.DB
	u := params.U
	handlerData := core.NewHandlerData(u)

	// Decode data
	var data editPageInfoData
	decoder := json.NewDecoder(params.R.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode json", err).Status(http.StatusBadRequest)
	}

	if !core.IsIDValid(data.PageID) {
		return pages.Fail("No pageId specified", nil).Status(http.StatusBadRequest)
	}

	// Load the published page.
	editLoadOptions := &core.LoadEditOptions{
		LoadNonliveEdit: true,
		PreferLiveEdit:  true,
	}
	oldPage, err := core.LoadFullEdit(db, data.PageID, u, handlerData.DomainMap, editLoadOptions)
	if err != nil {
		return pages.Fail("Couldn't load the old page", err)
	} else if oldPage == nil {
		return pages.Fail("Couldn't find the old page", err)
	}

	// Fix some data.
	if data.Type == core.CommentPageType {
		data.EditDomainID = u.MyDomainID()
	}
	if oldPage.WasPublished {
		if (data.Type == core.WikiPageType || data.Type == core.QuestionPageType) &&
			(oldPage.Type == core.WikiPageType || oldPage.Type == core.QuestionPageType) {
			// Allow type changing from wiki <-> question
		} else {
			// Don't allow type changing
			data.Type = oldPage.Type
		}
	}

	// Error checking.
	// Check the group settings
	if oldPage.SeeDomainID != data.SeeDomainID && oldPage.WasPublished {
		return pages.Fail("Editing this page in incorrect private group", nil).Status(http.StatusBadRequest)
	}
	// Check validity of most options. (We are super permissive with autosaves.)
	data.Type, err = core.CorrectPageType(data.Type)
	if err != nil {
		return pages.Fail(err.Error(), nil).Status(http.StatusBadRequest)
	}
	if data.SortChildrenBy != core.LikesChildSortingOption &&
		data.SortChildrenBy != core.RecentFirstChildSortingOption &&
		data.SortChildrenBy != core.OldestFirstChildSortingOption &&
		data.SortChildrenBy != core.AlphabeticalChildSortingOption {
		return pages.Fail("Invalid sort children value", nil).Status(http.StatusBadRequest)
	}
	if data.VoteType != "" && data.VoteType != core.ProbabilityVoteType && data.VoteType != core.ApprovalVoteType {
		return pages.Fail("Invalid vote type value", nil).Status(http.StatusBadRequest)
	}

	// Data correction. Rewrite the data structure so that we can just use it
	// in a straight-forward way to populate the database.
	// Can't change certain parameters after the page has been published.
	var hasVote bool
	if oldPage.WasPublished && oldPage.VoteType != "" {
		hasVote = data.HasVote
		data.VoteType = oldPage.VoteType
	} else {
		hasVote = data.VoteType != ""
	}
	// Can't un-anonymize votes
	if oldPage.VotesAnonymous {
		data.VotesAnonymous = oldPage.VotesAnonymous
	}
	// Enforce SortChildrenBy
	if data.Type == core.CommentPageType {
		data.SortChildrenBy = core.RecentFirstChildSortingOption
	} else if data.Type == core.QuestionPageType {
		data.SortChildrenBy = core.LikesChildSortingOption
	}

	// Make sure alias is valid
	if strings.ToLower(data.Alias) == "www" {
		return pages.Fail("Alias can't be 'www'", nil).Status(http.StatusBadRequest)
	} else if data.Type == core.GroupPageType {
		data.Alias = oldPage.Alias
	} else if data.Alias == "" {
		data.Alias = data.PageID
	} else if data.Alias != data.PageID {
		// Check if the alias matches the strict regexp
		if !core.StrictAliasRegexp.MatchString(data.Alias) {
			return pages.Fail("Invalid alias. An aliases can only contain letters, digits, and underscores. It also cannot start with a digit.", nil)
		}

		// Prefix alias with the group alias, if appropriate
		if core.IsIntIDValid(data.SeeDomainID) && data.Type != core.GroupPageType {
			if data.SeeDomainID != params.PrivateDomain.ID {
				return pages.Fail("Editing outside of the correct private domain", nil)
			}
			data.Alias = fmt.Sprintf("%s.%s", params.PrivateDomain.Alias, data.Alias)
		}

		// Check if another page is already using the alias
		var existingPageID string
		row := database.NewQuery(`
			SELECT pageId
			FROM pageInfos AS pi
			WHERE pageId!=?`, data.PageID).Add(`
				AND alias=?`, data.Alias).Add(`
				AND`).AddPart(core.PageInfosFilter(u)).ToStatement(db).QueryRow()
		exists, err := row.Scan(&existingPageID)
		if err != nil {
			return pages.Fail("Failed on looking for conflicting alias", err)
		} else if exists {
			return pages.Fail(fmt.Sprintf("Alias '%s' is already in use by: %s", data.Alias, existingPageID), nil)
		}

		// Check if there is a redirect from this alias
		// var aliasRedirect string
		// row := database.NewQuery(`
		// 	SELECT newAlias
		// 	FROM aliasRedirects
		// 	WHERE oldAlias=?`).AddPart(core.PageInfosFilter(u)).ToStatement(db).QueryRow()
		// exists, err := row.Scan(&aliasRedirect)
		// if err != nil {
		// 	return pages.Fail("Failed on looking for conflicting alias", err)
		// } else if exists {
		// 	return pages.Fail(fmt.Sprintf("Alias '%s' is already in use by: %s", data.Alias, existingPageID), nil)
		// }
	}

	// Check if something is actually different from live edit
	// NOTE: we do this as the last step before writing data, just so we can be sure
	// exactly what date we'll be writing
	if !oldPage.IsDeleted {
		if data.Alias == oldPage.Alias &&
			data.SortChildrenBy == oldPage.SortChildrenBy &&
			data.HasVote == oldPage.HasVote &&
			data.VoteType == oldPage.VoteType &&
			data.VotesAnonymous == oldPage.VotesAnonymous &&
			data.Type == oldPage.Type &&
			data.SeeDomainID == oldPage.SeeDomainID &&
			data.EditDomainID == oldPage.EditDomainID &&
			data.SubmitToDomainID == oldPage.SubmitToDomainID &&
			data.IndirectTeacher == oldPage.IndirectTeacher &&
			data.IsEditorComment == oldPage.IsEditorComment &&
			data.ExternalUrl == oldPage.ExternalUrl {
			return pages.Success(nil)
		}
	}

	// Make sure the user has the right permissions to edit this page
	// NOTE: check permissions AFTER checking if any data will be changed, becase we
	// don't want to flag the user for not having correct permissions, when they are
	// not actually changing anything
	if !oldPage.Permissions.Edit.Has {
		return pages.Fail("Can't edit: "+oldPage.Permissions.Edit.Reason, nil).Status(http.StatusBadRequest)
	}

	var changeLogIDs []int64

	// Begin the transaction.
	err2 := db.Transaction(func(tx *database.Tx) sessions.Error {
		// Update pageInfos
		hashmap := make(database.InsertMap)
		hashmap["pageId"] = data.PageID
		hashmap["alias"] = data.Alias
		hashmap["sortChildrenBy"] = data.SortChildrenBy
		hashmap["hasVote"] = hasVote
		hashmap["voteType"] = data.VoteType
		hashmap["votesAnonymous"] = data.VotesAnonymous
		hashmap["type"] = data.Type
		hashmap["seeDomainID"] = data.SeeDomainID
		hashmap["editDomainID"] = data.EditDomainID
		hashmap["submitToDomainId"] = data.SubmitToDomainID
		hashmap["indirectTeacher"] = data.IndirectTeacher
		hashmap["isEditorComment"] = data.IsEditorComment
		hashmap["externalUrl"] = data.ExternalUrl
		statement := tx.DB.NewInsertStatement("pageInfos", hashmap, hashmap.GetKeys()...).WithTx(tx)
		if _, err = statement.Exec(); err != nil {
			return sessions.NewError("Couldn't update pageInfos", err)
		}

		// Update change logs
		if oldPage.WasPublished {
			updateChangeLog := func(changeType string, auxPageID string, oldSettingsValue string, newSettingsValue string) (int64, sessions.Error) {

				hashmap = make(database.InsertMap)
				hashmap["pageId"] = data.PageID
				hashmap["userId"] = u.ID
				hashmap["createdAt"] = database.Now()
				hashmap["type"] = changeType
				hashmap["auxPageId"] = auxPageID
				hashmap["oldSettingsValue"] = oldSettingsValue
				hashmap["newSettingsValue"] = newSettingsValue
				statement = tx.DB.NewInsertStatement("changeLogs", hashmap).WithTx(tx)
				result, err := statement.Exec()
				if err != nil {
					return 0, sessions.NewError(fmt.Sprintf("Couldn't insert new child change log for %s", changeType), err)
				}
				changeLogID, err := result.LastInsertId()
				if err != nil {
					return 0, sessions.NewError(fmt.Sprintf("Couldn't insert new child change log for %s", changeType), err)
				}
				return changeLogID, nil
			}

			if data.Alias != oldPage.Alias {
				changeLogID, err2 := updateChangeLog(core.NewAliasChangeLog, "", oldPage.Alias, data.Alias)
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)

				// Create and alias redirect
				hashmap = make(database.InsertMap)
				hashmap["oldAlias"] = oldPage.Alias
				hashmap["newAlias"] = data.Alias
				statement = tx.DB.NewInsertStatement("aliasRedirects", hashmap, "newAlias").WithTx(tx)
				_, err10 := statement.Exec()
				if err10 != nil {
					return sessions.NewError(fmt.Sprintf("Couldn't insert new "), err10)
				}
			}
			if data.ExternalUrl != oldPage.ExternalUrl {
				changeLogID, err2 := updateChangeLog(core.NewAliasChangeLog, "", oldPage.ExternalUrl, data.ExternalUrl)
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
			if data.SortChildrenBy != oldPage.SortChildrenBy {
				changeLogID, err2 := updateChangeLog(core.NewSortChildrenByChangeLog, "", oldPage.SortChildrenBy, data.SortChildrenBy)
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
			if hasVote != oldPage.HasVote {
				changeType := core.TurnOnVoteChangeLog
				if !hasVote {
					changeType = core.TurnOffVoteChangeLog
				}
				changeLogID, err2 := updateChangeLog(changeType, "", strconv.FormatBool(oldPage.HasVote), strconv.FormatBool(hasVote))
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
			if data.VoteType != oldPage.VoteType {
				changeLogID, err2 := updateChangeLog(core.SetVoteTypeChangeLog, "", oldPage.VoteType, data.VoteType)
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
			if data.VotesAnonymous != oldPage.VotesAnonymous {
				changeLogID, err2 := updateChangeLog(core.SetVoteTypeChangeLog, "", "not anonymous", "anonymous")
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
			if data.EditDomainID != oldPage.EditDomainID {
				changeLogID, err2 := updateChangeLog(core.NewEditGroupChangeLog, data.EditDomainID, oldPage.EditDomainID, data.EditDomainID)
				if err2 != nil {
					return err2
				}
				changeLogIDs = append(changeLogIDs, changeLogID)
			}
		}
		return nil
	})
	if err2 != nil {
		return pages.FailWith(err2)
	}

	// === Once the transaction has succeeded, we can't really fail on anything
	// else. So we print out errors, but don't return an error. ===

	// Update elastic search index.
	if oldPage.WasPublished {
		var task tasks.UpdateElasticPageTask
		task.PageID = data.PageID
		if err := tasks.Enqueue(c, &task, nil); err != nil {
			c.Errorf("Couldn't enqueue a task: %v", err)
		}
	}

	// Generate "edit" update for users who are subscribed to this page.
	if oldPage.WasPublished {
		for _, changeLogID := range changeLogIDs {
			var task tasks.NewUpdateTask
			task.UserID = u.ID
			task.GoToPageID = data.PageID
			task.SubscribedToID = data.PageID
			task.UpdateType = core.ChangeLogUpdateType
			task.ChangeLogID = changeLogID
			if err := tasks.Enqueue(c, &task, nil); err != nil {
				c.Errorf("Couldn't enqueue a task: %v", err)
			}
		}
	}

	return pages.Success(nil)
}
