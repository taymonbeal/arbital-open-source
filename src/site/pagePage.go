// pagePage.go serves the page page.
package site

import (
	"fmt"
	"strconv"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"

	"github.com/gorilla/mux"
)

type alias struct {
	FullName  string `json:"fullName"`
	PageId    int64  `json:"pageId,string"`
	PageTitle string `json:"pageTitle"`
}

// pageTmplData stores the data that we pass to the index.tmpl to render the page
type pageTmplData struct {
	commonPageData
	Page        *core.Page
	LinkedPages []*core.Page
	RelatedIds  []string
}

var (
	pageOptions = pages.PageOptions{}
)

// pagePage serves the page page.
var pagePage = newPageWithOptions(
	fmt.Sprintf("/pages/{alias:%s}", core.AliasRegexpStr),
	pageRenderer,
	append(baseTmpls,
		"tmpl/pagePage.tmpl",
		"tmpl/angular.tmpl.js"), pageOptions)

// pageRenderer renders the page page.
func pageRenderer(params *pages.HandlerParams) *pages.Result {
	var data pageTmplData
	result := pageInternalRenderer(params, &data)
	if result.Data == nil {
		return pages.Fail(result.Message, result.Err)
	}

	if data.Page.Type == core.LensPageType {
		// Redirect lens pages to the parent page.
		parentId, _ := strconv.ParseInt(data.Page.ParentsStr, core.PageIdEncodeBase, 64)
		pageUrl := getPageUrl(&core.Page{Alias: fmt.Sprintf("%d", parentId)})
		return pages.RedirectWith(fmt.Sprintf("%s?lens=%d", pageUrl, data.Page.PageId))
	} else if data.Page.Type == core.CommentPageType {
		// Redirect comment pages to the primary page.
		// Note: we are actually redirecting blindly to a parent, which for replies
		// could be the parent comment. For now that's okay, since we just do anther
		// redirect then.
		for _, p := range data.Page.Parents {
			parent := data.PageMap[p.ParentId]
			if parent.Type != core.CommentPageType {
				pageUrl := getPageUrl(&core.Page{Alias: fmt.Sprintf("%d", parent.PageId)})
				return pages.RedirectWith(fmt.Sprintf("%s#subpage-%d", pageUrl, data.Page.PageId))
			}
		}
	}

	data.PrimaryPageId = data.Page.PageId
	return pages.StatusOK(result.Data)
}

// pageInternalRenderer renders the page page.
func pageInternalRenderer(params *pages.HandlerParams, data *pageTmplData) *pages.Result {
	c := params.C
	db := params.DB
	u := params.U

	var err error
	data.User = u

	// Figure out main page's id
	var pageId int64
	pageAlias := mux.Vars(params.R)["alias"]
	pageId, err = strconv.ParseInt(pageAlias, 10, 64)
	if err != nil {
		// Okay, it's not an id, but could be an alias.
		row := db.NewStatement(`
			SELECT pageId
			FROM pages
			WHERE alias=? AND isCurrentEdit AND deletedBy<=0`).QueryRow(pageAlias)
		exists, err := row.Scan(&pageId)
		if err != nil {
			return pages.Fail("Couldn't convert alias=>pageId", err)
		} else if !exists {
			return pages.Fail(fmt.Sprintf("There is no page with alias: %s", pageAlias), nil)
		}
	}

	// Load the main page
	data.Page, err = loadFullEdit(db, pageId, data.User.Id, &loadEditOptions{ignoreParents: true})
	if err != nil {
		return pages.Fail("Couldn't retrieve a page", err)
	} else if data.Page == nil {
		return pages.Fail(fmt.Sprintf("Couldn't find a page with id: %d", pageId), nil)
	}

	// Redirect lens pages to the parent page.
	if data.Page.Type == core.LensPageType {
		return pages.StatusOK(&data)
	}

	// Create maps.
	mainPageMap := make(map[int64]*core.Page)
	data.PageMap = make(map[int64]*core.Page)
	data.UserMap = make(map[int64]*core.User)
	data.GroupMap = make(map[int64]*core.Group)
	mainPageMap[data.Page.PageId] = data.Page

	// Load children
	err = loadChildrenIds(db, data.PageMap, loadChildrenIdsOptions{ForPages: mainPageMap, LoadHasChildren: true})
	if err != nil {
		return pages.Fail("Couldn't load children", err)
	}

	// Create embedded pages map, which will have pages that are displayed more
	// fully and need additional info loaded.
	embeddedPageMap := make(map[int64]*core.Page)
	embeddedPageMap[data.Page.PageId] = data.Page
	if data.Page.Type == core.QuestionPageType {
		for id, p := range data.PageMap {
			if p.Type == core.AnswerPageType {
				embeddedPageMap[id] = p
			}
		}
	}


	// Load comment ids.
	err = loadSubpageIds(db, data.PageMap, embeddedPageMap)
	if err != nil {
		return pages.Fail("Couldn't load subpages", err)
	}

	// Add comments and questions to the embedded pages map.
	for id, p := range data.PageMap {
		if p.Type == core.CommentPageType || p.Type == core.QuestionPageType {
			embeddedPageMap[id] = p
		}
	}

	// Load parents
	err = loadParentsIds(db, data.PageMap, loadParentsIdsOptions{ForPages: mainPageMap, LoadHasParents: true})
	if err != nil {
		return pages.Fail("Couldn't load parents", err)
	}

	// Load where page is linked from.
	/*query := fmt.Sprintf(`
		SELECT p.pageId
		FROM links as l
		JOIN pages as p
		ON l.parentId=p.pageId
		WHERE (l.childAlias=%d || l.childAlias="%s") AND p.isCurrentEdit
		GROUP BY p.pageId`, pageId, data.Page.Alias)
	data.Page.LinkedFrom, err = loadPageIds(c, query, mainPageMap)
	if err != nil {
		return pages.Fail("Couldn't load contexts", err)
	}*/

	// Load page ids of related pages (pages that have at least all the same parents).
	parentIds := make([]interface{}, len(data.Page.Parents))
	for i, parent := range data.Page.Parents {
		parentIds[i] = parent.ParentId
	}
	if len(parentIds) > 0 {
		rows := database.NewQuery(`
			SELECT childId
			FROM pagePairs AS pp
			WHERE parentId IN`).AddArgsGroup(parentIds).Add(` AND childId != ?`, data.Page.PageId).Add(`
			GROUP BY childId
			HAVING SUM(1)>=?`, len(data.Page.Parents)).ToStatement(db).Query()
		data.RelatedIds, err = loadPageIds(rows, data.PageMap)
		if err != nil {
			return pages.Fail("Couldn't load related ids", err)
		}
	}

	// Load the domains for the primary page
	err = loadDomains(db, u, data.Page, data.PageMap, data.GroupMap)
	if err != nil {
		return pages.Fail("Couldn't load domains", err)
	}

	// Load links
	err = loadLinks(db, data.PageMap, &loadLinksOptions{FromPageMap: embeddedPageMap})
	if err != nil {
		return pages.Fail("Couldn't load links", err)
	}

	// Load pages.
	err = core.LoadPages(db, data.PageMap, u.Id, &core.LoadPageOptions{LoadText: true})
	if err != nil {
		return pages.Fail("error while loading pages", err)
	}

	// Erase Text from pages that don't need it.
	// Also erase pages that weren't loaded.
	for _, p := range data.PageMap {
		if (data.Page.Type != core.QuestionPageType || p.Type != core.AnswerPageType) && (p.Type != core.CommentPageType && p.Type != core.QuestionPageType) {
			p.Text = ""
		}
	}

	// From here on, we also load info for the main page as well.
	data.PageMap[data.Page.PageId] = data.Page

	// Load auxillary data.
	q := params.R.URL.Query()
	options := loadAuxPageDataOptions{ForcedLastVisit: q.Get("lastVisit")}
	err = loadAuxPageData(db, data.User.Id, data.PageMap, &options)
	if err != nil {
		return pages.Fail("error while loading aux data", err)
	}

	// Load all the votes
	err = loadVotes(db, data.User.Id, mainPageMap, data.UserMap)
	if err != nil {
		return pages.Fail("error while fetching votes", err)
	}

	// Load child draft
	err = loadChildDraft(db, u.Id, data.Page, data.PageMap)
	if err != nil {
		return pages.Fail("Couldn't load child draft", err)
	}

	// Load all the groups.
	err = loadGroupNames(db, u, data.GroupMap)
	if err != nil {
		return pages.Fail("Couldn't load group names", err)
	}

	// Load all the users.
	data.UserMap[u.Id] = &core.User{Id: u.Id}
	for _, p := range data.PageMap {
		data.UserMap[p.CreatorId] = &core.User{Id: p.CreatorId}
	}
	err = core.LoadUsers(db, data.UserMap)
	if err != nil {
		return pages.Fail("error while loading users", err)
	}

	// From here on we can render the page successfully. Further queries are nice,
	// but not mandatory, so we are not going to return an error if they fail.

	// Add a visit to embedded pages.
	args := make([]interface{}, 0, 3*len(embeddedPageMap))
	for _, pg := range embeddedPageMap {
		args = append(args, data.User.Id, pg.PageId, database.Now())
	}
	statement := db.NewStatement(`
		INSERT INTO visits (userId, pageId, createdAt)
		VALUES` + database.ArgsPlaceholder(len(args), 3))
	_, err = statement.Exec(args...)
	if err != nil {
		c.Errorf("Error updating visits: %v", err)
	}

	return pages.StatusOK(&data)
}
