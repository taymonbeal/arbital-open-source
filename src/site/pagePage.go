// pagePage.go serves the page page.
package site

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/user"

	"github.com/gorilla/mux"
)

type alias struct {
	FullName  string
	PageId    int64 `json:",string"`
	PageTitle string
}

// pageTmplData stores the data that we pass to the index.tmpl to render the page
type pageTmplData struct {
	User        *user.User
	UserMap     map[int64]*dbUser
	PageMap     map[int64]*page
	Page        *page
	LinkedPages []*page
	AliasMap    map[string]*alias
	RelatedIds  []string
}

var (
	pageOptions = newPageOptions{LoadUserGroups: true}
)

// pagePage serves the page page.
var pagePage = newPageWithOptions(
	"/pages/{alias:[A-Za-z0-9_-]+}",
	pageRenderer,
	append(baseTmpls,
		"tmpl/pagePage.tmpl", "tmpl/pageHelpers.tmpl",
		"tmpl/angular.tmpl.js", "tmpl/comment.tmpl",
		"tmpl/navbar.tmpl", "tmpl/footer.tmpl"), pageOptions)

var privatePagePage = newPageWithOptions(
	"/pages/{alias:[A-Za-z0-9_-]+}/{privacyKey:[0-9]+}",
	pageRenderer,
	append(baseTmpls,
		"tmpl/pagePage.tmpl", "tmpl/pageHelpers.tmpl",
		"tmpl/angular.tmpl.js", "tmpl/comment.tmpl",
		"tmpl/navbar.tmpl", "tmpl/footer.tmpl"), pageOptions)

// loadLikes loads likes corresponding to the given pages and updates the pages.
func loadLikes(c sessions.Context, currentUserId int64, pageMap map[int64]*page) error {
	if len(pageMap) <= 0 {
		return nil
	}
	pageIdsStr := pageIdsStringFromMap(pageMap)
	query := fmt.Sprintf(`
		SELECT userId,pageId,value
		FROM (
			SELECT *
			FROM likes
			WHERE pageId IN (%s)
			ORDER BY id DESC
		) AS v
		GROUP BY userId,pageId`, pageIdsStr)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var userId int64
		var pageId int64
		var value int
		err := rows.Scan(&userId, &pageId, &value)
		if err != nil {
			return fmt.Errorf("failed to scan for a like: %v", err)
		}
		page := pageMap[pageId]
		if value > 0 {
			if page.LikeCount >= page.DislikeCount {
				page.LikeScore++
			} else {
				page.LikeScore += 2
			}
			page.LikeCount++
		} else if value < 0 {
			if page.DislikeCount >= page.LikeCount {
				page.LikeScore--
			}
			page.DislikeCount++
		}
		if userId == currentUserId {
			page.MyLikeValue = value
		}
		return nil
	})
	return err
}

// loadVotes loads probability votes corresponding to the given pages and updates the pages.
func loadVotes(c sessions.Context, currentUserId int64, pageIds string, pageMap map[int64]*page, usersMap map[int64]*dbUser) error {
	query := fmt.Sprintf(`
		SELECT userId,pageId,value,createdAt
		FROM (
			SELECT *
			FROM votes
			WHERE pageId IN (%s)
			ORDER BY id DESC
		) AS v
		GROUP BY userId,pageId`, pageIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var v vote
		var pageId int64
		err := rows.Scan(&v.UserId, &pageId, &v.Value, &v.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan for a vote: %v", err)
		}
		if v.Value == 0 {
			return nil
		}
		page := pageMap[pageId]
		if page.Votes == nil {
			page.Votes = make([]*vote, 0, 0)
		}
		page.Votes = append(page.Votes, &v)
		if _, ok := usersMap[v.UserId]; !ok {
			usersMap[v.UserId] = &dbUser{Id: v.UserId}
		}
		return nil
	})
	return err
}

// loadLastVisits loads lastVisit variable for each page.
func loadLastVisits(c sessions.Context, currentUserId int64, pageMap map[int64]*page) error {
	if len(pageMap) <= 0 {
		return nil
	}
	pageIdsStr := pageIdsStringFromMap(pageMap)
	query := fmt.Sprintf(`
		SELECT pageId,updatedAt
		FROM visits
		WHERE userId=%d AND pageId IN (%s)`,
		currentUserId, pageIdsStr)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var pageId int64
		var updatedAt string
		err := rows.Scan(&pageId, &updatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment like: %v", err)
		}
		pageMap[pageId].LastVisit = updatedAt
		return nil
	})
	return err
}

// loadSubscriptions loads subscription statuses corresponding to the given
// pages, and then updates the given maps.
func loadSubscriptions(c sessions.Context, currentUserId int64, pageMap map[int64]*page) error {
	pageIds := pageIdsStringFromMap(pageMap)

	query := fmt.Sprintf(`
		SELECT toPageId
		FROM subscriptions
		WHERE userId=%d AND toPageId IN (%s)`,
		currentUserId, pageIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var toPageId int64
		err := rows.Scan(&toPageId)
		if err != nil {
			return fmt.Errorf("failed to scan for a subscription: %v", err)
		}
		pageMap[toPageId].IsSubscribed = true
		return nil
	})
	return err
}

// loadAliases loads subscription statuses corresponding to the given
// pages and comments, and then updates the given maps.
func loadAliases(c sessions.Context, submatches [][]string) (map[string]*alias, error) {
	aliasMap := make(map[string]*alias)
	if len(submatches) <= 0 {
		return aliasMap, nil
	}

	var buffer bytes.Buffer
	for _, submatch := range submatches {
		buffer.WriteString(fmt.Sprintf(`"%s"`, submatch[1]))
		buffer.WriteString(",")
	}
	aliasFullNames := strings.TrimRight(buffer.String(), ",")

	query := fmt.Sprintf(`
		SELECT a.fullName,a.pageId,p.title
		FROM aliases as a
		LEFT JOIN pages as p
		ON a.pageId=p.pageId
		WHERE a.fullName IN (%s)`,
		aliasFullNames)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var a alias
		err := rows.Scan(&a.FullName, &a.PageId, &a.PageTitle)
		if err != nil {
			return fmt.Errorf("failed to scan for an alias: %v", err)
		}
		aliasMap[a.FullName] = &a
		return nil
	})
	return aliasMap, err
}

// pageRenderer renders the page page.
func pageRenderer(w http.ResponseWriter, r *http.Request, u *user.User) *pages.Result {
	c := sessions.NewContext(r)

	data, err := pageInternalRenderer(w, r, u)
	if err != nil {
		c.Errorf("%s", err)
		c.Inc("page_page_served_fail")
		return pages.InternalErrorWith(err)
	}

	funcMap := template.FuncMap{
		"GetEditLevel": func(p *page) string {
			return getEditLevel(p, data.User)
		},
		"GetDeleteLevel": func(p *page) string {
			return getDeleteLevel(p, data.User)
		},
		"GetPageEditUrl": func(p *page) string {
			return getEditPageUrl(p)
		},
	}
	c.Inc("page_page_served_success")
	return pages.StatusOK(data).AddFuncMap(funcMap)
}

// pageInternalRenderer renders the page page.
func pageInternalRenderer(w http.ResponseWriter, r *http.Request, u *user.User) (*pageTmplData, error) {
	var err error
	var data pageTmplData
	data.User = u
	c := sessions.NewContext(r)

	// Figure out main page's id
	var pageId int64
	pageAlias := mux.Vars(r)["alias"]
	pageId, err = strconv.ParseInt(pageAlias, 10, 64)
	if err != nil {
		// Okay, it's not an id, but could be an alias.
		query := fmt.Sprintf(`SELECT pageId FROM aliases WHERE fullName="%s"`, pageAlias)
		exists, err := database.QueryRowSql(c, query, &pageId)
		if err != nil {
			return nil, fmt.Errorf("Couldn't query aliases: %v", err)
		} else if !exists {
			return nil, fmt.Errorf("Page with alias '%s' doesn't exists", pageAlias)
		}
	}

	// Load the main page
	data.Page, err = loadEdit(c, pageId, data.User.Id, loadEditOptions{ignoreParents: true})
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve a page: %v", err)
	} else if data.Page == nil {
		return nil, fmt.Errorf("Couldn't find a page with id: %d", pageId)
	}

	// Create maps.
	mainPageMap := make(map[int64]*page)
	data.PageMap = make(map[int64]*page)
	data.UserMap = make(map[int64]*dbUser)
	mainPageMap[data.Page.PageId] = data.Page

	// Load children
	err = loadChildrenIds(c, data.PageMap, loadChildrenIdsOptions{ForPages: mainPageMap, LoadHasChildren: true})
	if err != nil {
		return nil, fmt.Errorf("Couldn't load children: %v", err)
	}

	// Create embedded pages map, which will have pages that are displayed more
	// fully and need additional info loaded.
	embeddedPageMap := make(map[int64]*page)
	embeddedPageMap[data.Page.PageId] = data.Page
	if data.Page.Type == questionPageType {
		for id, p := range data.PageMap {
			if p.Type == answerPageType {
				embeddedPageMap[id] = p
			}
		}
	}

	// Load comment ids.
	err = loadCommentIds(c, data.PageMap, embeddedPageMap)
	if err != nil {
		return nil, fmt.Errorf("Couldn't load comments: %v", err)
	}
	// Add comments to the embedded pages map.
	for id, p := range data.PageMap {
		if p.Type == commentPageType {
			embeddedPageMap[id] = p
		}
	}

	// Load parents
	err = loadParentsIds(c, data.PageMap, loadParentsIdsOptions{ForPages: mainPageMap, LoadHasParents: true})
	if err != nil {
		return nil, fmt.Errorf("Couldn't load parents: %v", err)
	}

	if data.Page.Type != questionPageType {
		// Load potential question draft.
		query := fmt.Sprintf(`
			SELECT pageId
			FROM pages
			WHERE type="question" AND creatorId=%d AND deletedBy<=0 AND parents REGEXP "(^|,)%s($|,)"
			GROUP BY pageId
			HAVING SUM(isCurrentEdit)<=0`, u.Id, strconv.FormatInt(pageId, pageIdEncodeBase))
		_, err = database.QueryRowSql(c, query, &data.Page.ChildDraftId)
		if err != nil {
			return nil, fmt.Errorf("Couldn't load question draft: %v", err)
		}
	} else {
		// Load potential answer draft.
		query := fmt.Sprintf(`
			SELECT pageId
			FROM pages
			WHERE type="answer" AND creatorId=%d AND deletedBy<=0 AND parents REGEXP "(^|,)%s($|,)"
			GROUP BY pageId
			HAVING SUM(isCurrentEdit)<=0`, u.Id, strconv.FormatInt(pageId, pageIdEncodeBase))
		_, err = database.QueryRowSql(c, query, &data.Page.ChildDraftId)
		if err != nil {
			return nil, fmt.Errorf("Couldn't load answer draft id: %v", err)
		}
		if data.Page.ChildDraftId > 0 {
			p, err := loadFullEdit(c, data.Page.ChildDraftId, u.Id)
			if err != nil {
				return nil, fmt.Errorf("Couldn't load answer draft: %v", err)
			}
			data.PageMap[p.PageId] = p
		}
	}

	// Load links
	err = loadLinks(c, mainPageMap, true)
	if err != nil {
		return nil, fmt.Errorf("Couldn't load links: %v", err)
	}

	// Load where page is linked from.
	// TODO: also account for old aliases
	query := fmt.Sprintf(`
		SELECT p.pageId
		FROM links as l
		JOIN pages as p
		ON l.parentId=p.pageId
		WHERE (l.childAlias=%d || l.childAlias="%s") AND p.isCurrentEdit
		GROUP BY p.pageId`, pageId, data.Page.Alias)
	data.Page.LinkedFrom, err = loadPageIds(c, query, mainPageMap)
	if err != nil {
		return nil, fmt.Errorf("Couldn't load contexts: %v", err)
	}

	// Check privacy setting
	if data.Page.PrivacyKey > 0 {
		privacyKey := mux.Vars(r)["privacyKey"]
		if privacyKey != fmt.Sprintf("%d", data.Page.PrivacyKey) {
			return nil, fmt.Errorf("Unauthorized access. You don't have the correct privacy key.")
		}
	}

	// Load all aliases.
	re := regexp.MustCompile(`\[\[([A-Za-z0-9_-]+?)\]\]`)
	aliases := re.FindAllStringSubmatch(data.Page.Text, -1)
	data.AliasMap, err = loadAliases(c, aliases)
	if err != nil {
		return nil, fmt.Errorf("error while fetching aliases: %v", err)
	}

	// Load page ids of related pages (pages that have at least all the same parents).
	parentIds := make([]string, len(data.Page.Parents))
	for i, parent := range data.Page.Parents {
		parentIds[i] = fmt.Sprintf("%d", parent.ParentId)
	}
	if len(parentIds) > 0 {
		parentIdsStr := strings.Join(parentIds, ",")
		query := fmt.Sprintf(`
			SELECT childId
			FROM pagePairs AS pp
			WHERE parentId IN (%s) AND childId!=%d
			GROUP BY childId
			HAVING SUM(1)>=%d`, parentIdsStr, data.Page.PageId, len(data.Page.Parents))
		data.RelatedIds, err = loadPageIds(c, query, data.PageMap)
		if err != nil {
			return nil, fmt.Errorf("Couldn't load related ids: %v", err)
		}
	}

	// Load pages.
	err = loadPages(c, data.PageMap, u.Id, loadPageOptions{loadText: true})
	if err != nil {
		return nil, fmt.Errorf("error while loading pages: %v", err)
	}

	// Erase Text from pages that don't need it.
	for _, p := range data.PageMap {
		if (data.Page.Type != questionPageType || p.Type != answerPageType) && p.Type != commentPageType {
			p.Text = ""
		}
	}

	// Load whether or not pages have drafts.
	err = loadDraftExistence(c, embeddedPageMap, data.User.Id)
	if err != nil {
		return nil, fmt.Errorf("Couldn't load draft existence: %v", err)
	}

	// Load original creation date.
	pageIdsStr := pageIdsStringFromMap(embeddedPageMap)
	query = fmt.Sprintf(`
		SELECT pageId,MIN(createdAt)
		FROM pages
		WHERE pageId IN (%s) AND NOT isAutosave AND NOT isSnapshot
		GROUP BY 1`, pageIdsStr)
	err = database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var pageId int64
		var originalCreatedAt string
		err := rows.Scan(&pageId, &originalCreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan for original createdAt: %v", err)
		}
		embeddedPageMap[pageId].OriginalCreatedAt = originalCreatedAt
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Couldn't load original createdAt: %v", err)
	}

	// Get last visits.
	q := r.URL.Query()
	forcedLastVisit := q.Get("lastVisit")
	if forcedLastVisit == "" {
		err = loadLastVisits(c, data.User.Id, embeddedPageMap)
		if err != nil {
			return nil, fmt.Errorf("error while fetching a visit: %v", err)
		}
	} else {
		for _, p := range embeddedPageMap {
			p.LastVisit = forcedLastVisit
		}
	}

	// From here on, we also load info for the main page as well.
	data.PageMap[data.Page.PageId] = data.Page

	// Load all the subscription statuses.
	if data.User.Id > 0 {
		err = loadSubscriptions(c, data.User.Id, data.PageMap)
		if err != nil {
			return nil, fmt.Errorf("error while fetching subscriptions: %v", err)
		}
	}

	// Load all the likes
	err = loadLikes(c, data.User.Id, data.PageMap)
	if err != nil {
		return nil, fmt.Errorf("error while fetching likes: %v", err)
	}

	// Load all the votes
	err = loadVotes(c, data.User.Id, fmt.Sprintf("%d", pageId), mainPageMap, data.UserMap)
	if err != nil {
		return nil, fmt.Errorf("error while fetching votes: %v", err)
	}

	// Load all the users.
	data.UserMap[u.Id] = &dbUser{Id: u.Id}
	for _, p := range data.PageMap {
		data.UserMap[p.CreatorId] = &dbUser{Id: p.CreatorId}
	}
	err = loadUsersInfo(c, data.UserMap)
	if err != nil {
		return nil, fmt.Errorf("error while loading users: %v", err)
	}

	// From here on we can render the page successfully. Further queries are nice,
	// but not mandatory, so we are not going to return an error if they fail.
	if data.User.Id > 0 {
		// Mark the relevant updates as read.
		query := fmt.Sprintf(
			`UPDATE updates
			SET seen=1,updatedAt='%s'
			WHERE contextPageId=%d AND userId=%d`,
			database.Now(), pageId, data.User.Id)
		if _, err := database.ExecuteSql(c, query); err != nil {
			return nil, fmt.Errorf("Couldn't update updates: %v", err)
		}

		// Update last visit date.
		values := ""
		for _, pg := range embeddedPageMap {
			values += fmt.Sprintf("(%d, %d, '%s', '%s'),",
				data.User.Id, pg.PageId, database.Now(), database.Now())
		}
		values = strings.TrimRight(values, ",")
		sql := fmt.Sprintf(`
			INSERT INTO visits (userId, pageId, createdAt, updatedAt)
			VALUES %s
			ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`, values)
		if _, err = database.ExecuteSql(c, sql); err != nil {
			return nil, fmt.Errorf("Couldn't update visits: %v", err)
		}

		// Load updates count.
		data.User.UpdateCount, err = loadUpdateCount(c, data.User.Id)
		if err != nil {
			return nil, fmt.Errorf("Couldn't retrieve updates count: %v", err)
		}
	}

	return &data, nil
}
