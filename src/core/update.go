// updates.go contains all the updates stuff
package core

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"time"

	"appengine/urlfetch"

	"zanaduu3/src/database"
	"zanaduu3/src/sessions"

	"github.com/dustin/go-humanize"
)

const (
	// Various types of updates a user can get.
	TopLevelCommentUpdateType        = "topLevelComment"
	ReplyUpdateType                  = "reply"
	ChangeLogUpdateType              = "changeLog"
	PageEditUpdateType               = "pageEdit"
	NewPageByUserUpdateType          = "newPageByUser"
	PageToDomainSubmissionUpdateType = "pageToDomainSubmission"
	PageToDomainAcceptedUpdateType   = "pageToDomainAccepted"

	// there's no deleteLens because there's no way to undo the association between a lens and its parent page
	// (other than deleting the lens page)
	NewLensUpdateType = "newLens"

	AtMentionUpdateType             = "atMention"
	AddedToGroupUpdateType          = "addedToGroup"
	RemovedFromGroupUpdateType      = "removedFromGroup"
	InviteReceivedUpdateType        = "inviteReceived"
	NewMarkUpdateType               = "newMark"
	ResolvedThreadUpdateType        = "resolvedThread"
	ResolvedMarkUpdateType          = "resolvedMark"
	AnsweredMarkUpdateType          = "answeredMark"
	QuestionMergedUpdateType        = "questionMerged"
	QuestionMergedReverseUpdateType = "questionMergedReverse"
)

// UpdateRow is a row from updates table
type UpdateRow struct {
	Id                   string
	UserId               string
	ByUserId             string
	CreatedAt            string
	Type                 string
	GroupByPageId        string
	GroupByUserId        string
	Seen                 bool
	SubscribedToId       string
	GoToPageId           string
	MarkId               string
	IsGroupByObjectAlive bool
	IsGoToPageAlive      bool
	ChangeLog            *ChangeLog
}

// UpdateGroupKey is what we group updateEntries by
type UpdateGroupKey struct {
	GroupByPageId        string `json:"groupByPageId"`
	GroupByUserId        string `json:"groupByUserId"`
	Seen                 bool   `json:"seen"`
	IsGroupByObjectAlive bool   `json:"isGroupByObjectAlive"`
}

// UpdateEntry corresponds to one update entry we'll display.
type UpdateEntry struct {
	Id              string `json:"id"`
	UserId          string `json:"userId"`
	ByUserId        string `json:"byUserId"`
	Type            string `json:"type"`
	Repeated        int    `json:"repeated"`
	SubscribedToId  string `json:"subscribedToId"`
	GoToPageId      string `json:"goToPageId"`
	IsGoToPageAlive bool   `json:"isGoToPageAlive"`
	MarkId          string `json:"markId"`
	// Optional changeLog associated with this update
	ChangeLog *ChangeLog `json:"changeLog"`
	// True if the user has gone to the GoToPage
	IsVisited bool   `json:"isVisited"`
	CreatedAt string `json:"createdAt"`
}

// UpdateGroup is a collection of updates groupped by the context page.
type UpdateGroup struct {
	Key *UpdateGroupKey `json:"key"`
	// The date of the most recent update
	MostRecentDate string         `json:"mostRecentDate"`
	Updates        []*UpdateEntry `json:"updates"`
}

// UpdateData is all the data collected by LoadUpdateEmail()
type UpdateData struct {
	UpdateCount        int
	UpdateRows         []*UpdateRow
	UpdateGroups       []*UpdateGroup
	UpdateEmailAddress string
	UpdateEmailText    string
}

// LoadUpdateRows loads all the updates for the given user, populating the
// given maps.
func LoadUpdateRows(db *database.DB, u *CurrentUser, resultData *CommonHandlerData, forEmail bool, updateTypes []string, limit int) ([]*UpdateRow, error) {
	emailFilter := database.NewQuery("")
	if forEmail {
		emailFilter = database.NewQuery("AND NOT updates.seen AND NOT updates.emailed")
	}

	updateTypeFilter := database.NewQuery("")
	if len(updateTypes) > 0 {
		updateTypeFilter = database.NewQuery("AND updates.type IN").AddArgsGroupStr(updateTypes)
	}

	if limit <= 0 {
		limit = 100
	}

	// Create group loading options
	groupLoadOptions := (&PageLoadOptions{IsSubscribed: true}).Add(TitlePlusIncludeDeletedLoadOptions)
	goToPageLoadOptions := (&PageLoadOptions{
		Parents: true, // to show comment threads properly
	}).Add(TitlePlusIncludeDeletedLoadOptions)

	updateRows := make([]*UpdateRow, 0, 0)
	changeLogs := make([]*ChangeLog, 0)

	rows := database.NewQuery(`
		SELECT updates.id,updates.userId,updates.byUserId,updates.createdAt,updates.type,updates.seen,
			updates.groupByPageId,updates.groupByUserId,updates.subscribedToId,updates.goToPageId,updates.markId,
			(
				SELECT !isDeleted
				FROM`).AddPart(PageInfosTableWithOptions(u, &PageInfosOptions{Deleted: true})).Add(`AS pi
				WHERE pageId IN (updates.groupByPageId, updates.groupByUserId)
			) AS isGroupByObjectAlive,
			COALESCE((
				SELECT !isDeleted
				FROM`).AddPart(PageInfosTableWithOptions(u, &PageInfosOptions{Deleted: true})).Add(`AS pi
				WHERE updates.goToPageId = pageId), false
			) AS isGoToPageAlive,
			COALESCE(changeLogs.id, 0),
			COALESCE(changeLogs.pageId, 0),
			COALESCE(changeLogs.type, ''),
			COALESCE(changeLogs.oldSettingsValue, ''),
			COALESCE(changeLogs.newSettingsValue, ''),
			COALESCE(changeLogs.edit, 0)
		FROM updates
		LEFT JOIN changeLogs
		ON (updates.changeLogId = changeLogs.id)
		WHERE updates.userId=?`, u.Id).AddPart(emailFilter).AddPart(updateTypeFilter).Add(`
			AND NOT updates.dismissed
		HAVING isGoToPageAlive OR updates.type IN`).AddArgsGroupStr(getOkayToShowWhenGoToPageIsDeletedUpdateTypes()).Add(`
		ORDER BY updates.createdAt DESC
		LIMIT ?`, limit).ToStatement(db).Query()
	err := rows.Process(func(db *database.DB, rows *database.Rows) error {
		var row UpdateRow
		var changeLog ChangeLog
		err := rows.Scan(&row.Id, &row.UserId, &row.ByUserId, &row.CreatedAt, &row.Type,
			&row.Seen, &row.GroupByPageId, &row.GroupByUserId, &row.SubscribedToId,
			&row.GoToPageId, &row.MarkId, &row.IsGroupByObjectAlive, &row.IsGoToPageAlive,
			&changeLog.Id, &changeLog.PageId, &changeLog.Type, &changeLog.OldSettingsValue,
			&changeLog.NewSettingsValue, &changeLog.Edit)
		if err != nil {
			return fmt.Errorf("failed to scan an update: %v", err)
		}
		row.ChangeLog = &changeLog
		AddPageToMap(row.GoToPageId, resultData.PageMap, goToPageLoadOptions)
		AddPageToMap(row.GroupByPageId, resultData.PageMap, groupLoadOptions)
		AddPageToMap(row.SubscribedToId, resultData.PageMap, groupLoadOptions)
		if row.Type == PageToDomainSubmissionUpdateType {
			AddPageToMap(row.GoToPageId, resultData.PageMap, &PageLoadOptions{SubmittedTo: true})
		}

		resultData.UserMap[row.UserId] = &User{Id: row.UserId}
		if IsIdValid(row.ByUserId) {
			resultData.UserMap[row.ByUserId] = &User{Id: row.ByUserId}
		}
		if IsIdValid(row.GroupByUserId) {
			resultData.UserMap[row.GroupByUserId] = &User{Id: row.GroupByUserId}
		}
		if row.MarkId == "0" {
			row.MarkId = ""
		} else {
			resultData.AddMark(row.MarkId)
		}
		if row.ChangeLog.Id != 0 {
			changeLogs = append(changeLogs, row.ChangeLog)
		}

		updateRows = append(updateRows, &row)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error while loading updates: %v", err)
	}

	err = LoadLikesForChangeLogs(db, u.Id, changeLogs)
	if err != nil {
		return nil, fmt.Errorf("error while loading likes for changelogs: %v", err)
	}

	return updateRows, nil
}

// ConvertUpdateRowsToGroups converts a list of Rows into a list of Groups
func ConvertUpdateRowsToGroups(rows []*UpdateRow, pageMap map[string]*Page) []*UpdateGroup {
	// Now that we have load last visit time for all pages,
	// go through all the update rows and group them.
	groups := make([]*UpdateGroup, 0)
	groupMap := make(map[UpdateGroupKey]*UpdateGroup)
	for _, row := range rows {
		key := UpdateGroupKey{
			GroupByPageId:        row.GroupByPageId,
			GroupByUserId:        row.GroupByUserId,
			Seen:                 row.Seen,
			IsGroupByObjectAlive: row.IsGroupByObjectAlive,
		}

		// Create/update the group.
		group, ok := groupMap[key]
		if !ok {
			group = &UpdateGroup{
				Key:            &key,
				MostRecentDate: row.CreatedAt,
				Updates:        make([]*UpdateEntry, 0),
			}
			groupMap[key] = group
			groups = append(groups, group)
		} else if group.MostRecentDate < row.CreatedAt {
			group.MostRecentDate = row.CreatedAt
		}

		createNewEntry := true
		if row.Type == PageEditUpdateType {
			// Check if this kind of update already exists
			for _, entry := range group.Updates {
				if entry.Type == row.Type && entry.SubscribedToId == row.SubscribedToId &&
					entry.ByUserId == row.ByUserId {
					createNewEntry = false
					entry.Repeated++
					break
				}
			}
		}
		if _, ok := pageMap[row.GoToPageId]; !ok {
			createNewEntry = false
		}
		if createNewEntry {
			// Add new entry to the group
			entry := &UpdateEntry{
				Id:              row.Id,
				UserId:          row.UserId,
				ByUserId:        row.ByUserId,
				Type:            row.Type,
				Repeated:        1,
				SubscribedToId:  row.SubscribedToId,
				GoToPageId:      row.GoToPageId,
				IsGoToPageAlive: row.IsGoToPageAlive,
				MarkId:          row.MarkId,
				CreatedAt:       row.CreatedAt,
				IsVisited:       pageMap != nil && row.CreatedAt < pageMap[row.GoToPageId].LastVisit,
				ChangeLog:       row.ChangeLog,
			}
			if entry.MarkId != "" {
				entry.ByUserId = ""
			}
			group.Updates = append(group.Updates, entry)
		}
	}
	return groups
}

// LoadUpdateEmail loads the text and other data for the update email
func LoadUpdateEmail(db *database.DB, userId string) (resultData *UpdateData, retErr error) {
	c := db.C

	resultData = &UpdateData{}

	// TODO: replace this with a helper function (like loadUserFromDb)
	u := &CurrentUser{}
	row := db.NewStatement(`
		SELECT id,email,emailFrequency,emailThreshold
		FROM users
		WHERE id=?`).QueryRow(userId)
	_, err := row.Scan(&u.Id, &u.Email, &u.EmailFrequency, &u.EmailThreshold)
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve a user: %v", err)
	}

	// Load the groups the user belongs to.
	if err = LoadUserGroupIds(db, u); err != nil {
		return nil, fmt.Errorf("Couldn't load user groups: %v", err)
	}

	handlerData := NewHandlerData(u)

	// Load updates and populate the maps
	resultData.UpdateRows, err = LoadUpdateRows(db, u, handlerData, true, make([]string, 0), -1)
	if err != nil {
		return nil, fmt.Errorf("failed to load updates: %v", err)
	}

	// Check to make sure there are enough updates
	resultData.UpdateCount = len(resultData.UpdateRows)
	if resultData.UpdateCount < u.EmailThreshold {
		db.C.Debugf("Not enough updates to send the email: %d < %d", resultData.UpdateCount, u.EmailThreshold)
		return nil, nil
	}
	resultData.UpdateGroups = ConvertUpdateRowsToGroups(resultData.UpdateRows, handlerData.PageMap)

	// Load pages.
	err = ExecuteLoadPipeline(db, handlerData)
	if err != nil {
		return nil, fmt.Errorf("Pipeline error: %v", err)
	}

	// Load the template file
	var templateBytes []byte
	if sessions.Live {
		resp, err := urlfetch.Client(c).Get(fmt.Sprintf("%s/static/updatesEmailInlined.html", sessions.GetDomain()))
		if err != nil {
			return nil, fmt.Errorf("Couldn't load the email template form URL: %v", err)
		}
		defer resp.Body.Close()
		templateBytes, err = ioutil.ReadAll(resp.Body)
	} else {
		templateBytes, err = ioutil.ReadFile("../site/static/updatesEmailInlined.html")
	}
	if err != nil {
		return nil, fmt.Errorf("Couldn't load the email template from file: %v", err)
	}

	funcMap := template.FuncMap{
		//"UserFirstName": func() string { return u.Id },
		"GetUserUrl": func(userId string) string {
			return fmt.Sprintf(`%s/user/%s`, sessions.GetDomainForTestEmail(), userId)
		},
		"GetUserName": func(userId string) string {
			return handlerData.UserMap[userId].FullName()
		},
		"GetPageUrl": func(pageId string) string {
			return fmt.Sprintf("%s/p/%s/"+handlerData.PageMap[pageId].Alias, sessions.GetDomainForTestEmail(), pageId)
		},
		"GetPageTitle": func(pageId string) string {
			return handlerData.PageMap[pageId].Title
		},
		"RelativeDateTime": func(date string) string {
			t, err := time.Parse(database.TimeLayout, date)
			if err != nil {
				c.Errorf("Couldn't parse %s: %v", date, err)
				return ""
			}
			return humanize.Time(t)
		},
	}

	// Create and execute template
	buffer := &bytes.Buffer{}
	t, err := template.New("email").Funcs(funcMap).Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse template: %v", err)
	}
	err = t.Execute(buffer, resultData)
	if err != nil {
		return nil, fmt.Errorf("Couldn't execute template: %v", err)
	}

	resultData.UpdateEmailAddress = u.Email
	resultData.UpdateEmailText = buffer.String()

	return resultData, nil
}

func GetAchievementUpdateTypes() []string {
	return []string{
		AddedToGroupUpdateType,
		RemovedFromGroupUpdateType,
		InviteReceivedUpdateType,
	}
}

func GetNotificationUpdateTypes() []string {
	return []string{
		TopLevelCommentUpdateType,
		ReplyUpdateType,
		PageToDomainSubmissionUpdateType,
		PageToDomainAcceptedUpdateType,
		AtMentionUpdateType,
		NewMarkUpdateType,
		ResolvedThreadUpdateType,
		ResolvedMarkUpdateType,
		AnsweredMarkUpdateType,
	}
}

func GetMaintenanceUpdateTypes() []string {
	return []string{
		PageEditUpdateType,
		NewLensUpdateType,
		ChangeLogUpdateType,
		QuestionMergedUpdateType,
		QuestionMergedReverseUpdateType,
	}
}

func getOkayToShowWhenGoToPageIsDeletedUpdateTypes() []string {
	return []string{
		ChangeLogUpdateType,
		PageEditUpdateType,
	}
}

func MarkUpdatesAsSeen(db *database.DB, userId string, types []string) error {
	statement := database.NewQuery(`
		UPDATE updates
		SET seen=TRUE
		WHERE userId=?`, userId).Add(`
			AND type IN`).AddArgsGroupStr(types).ToStatement(db)
	_, err := statement.Exec()
	return err
}
