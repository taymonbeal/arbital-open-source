// pagePage.go serves the page page.
package site

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/user"

	"github.com/gorilla/mux"
)

type comment struct {
	Id           int64
	PageId       int64
	ReplyToId    int64
	Text         string
	CreatedAt    string
	UpdatedAt    string
	Author       dbUser
	LikeCount    int
	MyLikeValue  int
	IsSubscribed bool
	Replies      []*comment
}

// Helpers for soring comments by createdAt date.
type byDate []comment

func (a byDate) Len() int           { return len(a) }
func (a byDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDate) Less(i, j int) bool { return a[i].CreatedAt < a[j].CreatedAt }

// TODO: use this for context (and potentially list of links)
type input struct {
	Id        int64
	ChildId   int64
	CreatedAt string
	UpdatedAt string
	CreatorId int64
}

type answer struct {
	// DB values.
	IndexId int
	Text    string
}

type richPage struct {
	// DB values.
	page
	LastVisit string

	// Computed values.
	InputCount   int
	IsSubscribed bool
	LikeCount    int
	DislikeCount int
	MyLikeValue  int
	VoteValue    sql.NullFloat64
	VoteCount    int
	MyVoteValue  sql.NullFloat64
	Contexts     []*richPage
	Links        []*richPage
	Comments     []*comment
}

// pageTmplData stores the data that we pass to the index.tmpl to render the page
type pageTmplData struct {
	User        *user.User
	Page        *richPage
	LinkedPages []*richPage
	Inputs      []*input
}

// pagePage serves the page page.
var pagePage = newPage(
	"/pages/{id:[0-9]+}",
	pageRenderer,
	append(baseTmpls,
		"tmpl/pagePage.tmpl", "tmpl/pageHelpers.tmpl",
		"tmpl/comment.tmpl",
		"tmpl/navbar.tmpl", "tmpl/footer.tmpl"))

var privatePagePage = newPage(
	"/pages/{id:[0-9]+}/{privacyKey:[0-9]+}",
	pageRenderer,
	append(baseTmpls,
		"tmpl/pagePage.tmpl", "tmpl/pageHelpers.tmpl",
		"tmpl/comment.tmpl",
		"tmpl/navbar.tmpl", "tmpl/footer.tmpl"))

// loadMainPage loads and returns the main page.
func loadMainPage(c sessions.Context, userId int64, pageId int64) (*richPage, error) {
	c.Infof("querying DB for page with id = %d\n", pageId)

	pagePtr, err := loadFullPage(c, pageId)
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve a page: %v", err)
	}
	mainPage := &richPage{page: *pagePtr}

	// Load contexts.
	query := fmt.Sprintf(`
		SELECT p.pageId,p.title,p.privacyKey
		FROM links as l
		JOIN pages as p
		ON l.parentId=p.pageId
		WHERE l.childId=%d AND (p.privacyKey=0 OR p.creatorId=%d) AND p.deletedBy=0 AND p.edit=0
		GROUP BY p.pageId`, pageId, userId)
	err = database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var p richPage
		err := rows.Scan(&p.PageId, &p.Title, &p.PrivacyKey)
		if err != nil {
			return fmt.Errorf("failed to scan for context page: %v", err)
		}
		mainPage.Contexts = append(mainPage.Contexts, &p)
		return nil
	})
	return mainPage, err
}

// loadComments loads and returns all the comments for the given input ids from the db.
func loadComments(c sessions.Context, pageIds string) (map[int64]*comment, []int64, error) {
	commentMap := make(map[int64]*comment)
	sortedCommentIds := make([]int64, 0)

	c.Infof("querying DB for comments with pageIds = %v", pageIds)
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT c.id,pageId,replyToId,text,createdAt,updatedAt,u.id,u.firstName,u.lastName
		FROM comments AS c
		LEFT JOIN (
			SELECT id,firstName,lastName
			FROM users
		) AS u
		ON c.creatorId=u.id
		WHERE pageId IN (%s)`, pageIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var ct comment
		err := rows.Scan(
			&ct.Id,
			&ct.PageId,
			&ct.ReplyToId,
			&ct.Text,
			&ct.CreatedAt,
			&ct.UpdatedAt,
			&ct.Author.Id,
			&ct.Author.FirstName,
			&ct.Author.LastName)
		if err != nil {
			return fmt.Errorf("failed to scan for comments: %v", err)
		}
		commentMap[ct.Id] = &ct
		sortedCommentIds = append(sortedCommentIds, ct.Id)
		return nil
	})
	return commentMap, sortedCommentIds, err
}

// loadLikes loads likes corresponding to the given pages and updates the pages.
func loadLikes(c sessions.Context, currentUserId int64, pageIds string, pageMap map[int64]*richPage) error {
	if len(pageIds) <= 0 {
		return nil
	}
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT userId,pageId,value
		FROM (
			SELECT *
			FROM likes
			WHERE pageId IN (%s)
			ORDER BY id DESC
		) AS v
		GROUP BY userId,pageId`, pageIds)
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
			page.LikeCount++
		} else if value < 0 {
			page.DislikeCount++
		}
		if userId == currentUserId {
			page.MyLikeValue = value
		}
		return nil
	})
	return err
}

// loadCommentLikes loads likes corresponding to the given comments and updates the comments.
func loadCommentLikes(c sessions.Context, currentUserId int64, commentIds string, commentMap map[int64]*comment) error {
	if len(commentIds) <= 0 {
		return nil
	}
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT userId,commentId,value
		FROM commentLikes
		WHERE commentId IN (%s)`, commentIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var userId int64
		var commentId int64
		var value int
		err := rows.Scan(&userId, &commentId, &value)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment like: %v", err)
		}
		comment := commentMap[commentId]
		if value > 0 {
			comment.LikeCount++
		}
		if userId == currentUserId {
			comment.MyLikeValue = value
		}
		return nil
	})
	return err
}

// loadVotes loads probability votes corresponding to the given pages and updates the pages.
func loadVotes(c sessions.Context, currentUserId int64, pageIds string, pageMap map[int64]*richPage) error {
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT userId,pageId,value
		FROM (
			SELECT *
			FROM votes
			WHERE pageId IN (%s)
			ORDER BY id DESC
		) AS v
		GROUP BY userId,pageId`, pageIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var userId int64
		var pageId int64
		var value float64
		err := rows.Scan(&userId, &pageId, &value)
		if err != nil {
			return fmt.Errorf("failed to scan for a vote: %v", err)
		}
		if value == 0 {
			return nil
		}
		page := pageMap[pageId]
		page.VoteCount++
		page.VoteValue.Valid = true
		page.VoteValue.Float64 += value
		if userId == currentUserId {
			page.MyVoteValue = sql.NullFloat64{Valid: true, Float64: value}
		}
		return nil
	})
	for _, p := range pageMap {
		if p.VoteCount > 0 {
			p.VoteValue.Float64 /= float64(p.VoteCount)
		}
	}
	return err
}

// loadLastVisits loads lastVisit variable for each page.
func loadLastVisits(c sessions.Context, currentUserId int64, pageIds string, pageMap map[int64]*richPage) error {
	query := fmt.Sprintf(`
		SELECT pageId,updatedAt
		FROM visits
		WHERE userId=%d AND pageId IN (%s)`,
		currentUserId, pageIds)
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
// pages and comments, and then updates the given maps.
func loadSubscriptions(
	c sessions.Context, currentUserId int64,
	pageIds string, commentIds string,
	pageMap map[int64]*richPage,
	commentMap map[int64]*comment) error {

	commentClause := ""
	if len(commentIds) > 0 {
		commentClause = fmt.Sprintf("OR toCommentId IN (%s)", commentIds)
	}

	query := fmt.Sprintf(`
		SELECT toPageId,toCommentId
		FROM subscriptions
		WHERE userId=%d AND (toPageId IN (%s) %s)`,
		currentUserId, pageIds, commentClause)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var toPageId int64
		var toCommentId int64
		err := rows.Scan(&toPageId, &toCommentId)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment like: %v", err)
		}
		if toPageId > 0 {
			pageMap[toPageId].IsSubscribed = true
		} else if toCommentId > 0 {
			commentMap[toCommentId].IsSubscribed = true
		}
		return nil
	})
	return err
}

// pageRenderer renders the page page.
func pageRenderer(w http.ResponseWriter, r *http.Request, u *user.User) *pages.Result {
	var err error
	var data pageTmplData
	data.User = u
	c := sessions.NewContext(r)

	// Load the parent page
	var pageId int64
	pageMap := make(map[int64]*richPage)
	pageIdStr := mux.Vars(r)["id"]
	pageId, err = strconv.ParseInt(pageIdStr, 10, 64)
	if err != nil {
		c.Inc("page_fetch_fail")
		c.Errorf("invalid id passed: %v", err)
		return showError(w, r, err)
	}
	mainPage, err := loadMainPage(c, data.User.Id, pageId)
	if err != nil {
		c.Inc("page_fetch_fail")
		c.Errorf("error while fetching a page: %v", err)
		return showError(w, r, err)
	}
	pageMap[mainPage.PageId] = mainPage
	data.Page = mainPage

	// Check privacy setting
	if mainPage.PrivacyKey > 0 {
		privacyKey := mux.Vars(r)["privacyKey"]
		if privacyKey != fmt.Sprintf("%d", mainPage.PrivacyKey) {
			return showError(w, r, fmt.Errorf("Unauthorized access. You don't have the correct privacy key."))
		}
	}

	// Load all the likes
	err = loadLikes(c, data.User.Id, pageIdStr, pageMap)
	if err != nil {
		c.Inc("likes_fetch_fail")
		c.Errorf("error while fetching likes: %v", err)
		return pages.InternalErrorWith(err)
	}

	// Load all the votes
	err = loadVotes(c, data.User.Id, pageIdStr, pageMap)
	if err != nil {
		c.Inc("votes_fetch_fail")
		c.Errorf("error while fetching votes: %v", err)
		return pages.InternalErrorWith(err)
	}

	// Get last visits.
	q := r.URL.Query()
	forcedLastVisit := q.Get("lastVisit")
	if forcedLastVisit == "" {
		err = loadLastVisits(c, data.User.Id, pageIdStr, pageMap)
		if err != nil {
			c.Errorf("error while fetching a visit: %v", err)
		}
	} else {
		for _, cl := range pageMap {
			cl.LastVisit = forcedLastVisit
		}
	}

	// Load all the comments
	var commentMap map[int64]*comment // commentId -> comment
	var sortedCommentKeys []int64     // need this for in-order iteration
	commentMap, sortedCommentKeys, err = loadComments(c, pageIdStr)
	if err != nil {
		c.Inc("comments_fetch_fail")
		c.Errorf("error while fetching comments: %v", err)
		return pages.InternalErrorWith(err)
	}
	for _, key := range sortedCommentKeys {
		comment := commentMap[key]
		pageObj, ok := pageMap[comment.PageId]
		if !ok {
			c.Errorf("couldn't find page for a comment: %d\n%v", key, err)
			return pages.InternalErrorWith(err)
		}
		if comment.ReplyToId > 0 {
			parent := commentMap[comment.ReplyToId]
			parent.Replies = append(parent.Replies, commentMap[key])
		} else {
			pageObj.Comments = append(pageObj.Comments, commentMap[key])
		}
	}

	// Get a string of all comment ids.
	var buffer bytes.Buffer
	//buffer.Reset()
	for id, _ := range commentMap {
		buffer.WriteString(fmt.Sprintf("%d", id))
		buffer.WriteString(",")
	}
	commentIds := strings.TrimRight(buffer.String(), ",")

	// Load all the comment likes
	err = loadCommentLikes(c, data.User.Id, commentIds, commentMap)
	if err != nil {
		c.Inc("comment_likes_fetch_fail")
		c.Errorf("error while fetching comment likes: %v", err)
		return pages.InternalErrorWith(err)
	}

	if data.User.Id > 0 {
		// Load subscription statuses.
		err = loadSubscriptions(c, data.User.Id, pageIdStr, commentIds, pageMap, commentMap)
		if err != nil {
			c.Inc("subscriptions_fetch_fail")
			c.Errorf("error while fetching subscriptions: %v", err)
			return pages.InternalErrorWith(err)
		}

		// From here on we can render the page successfully. Further queries are nice,
		// but not mandatory, so we are not going to return an error if they fail.

		// Mark the relevant updates as read.
		query := fmt.Sprintf(
			`UPDATE updates
			SET seen=1,updatedAt='%s'
			WHERE contextPageId=%d AND userId=%d`,
			database.Now(), pageId, data.User.Id)
		if _, err := database.ExecuteSql(c, query); err != nil {
			c.Errorf("Couldn't update updates: %v", err)
		}

		// Update last visit date.
		values := ""
		for _, pg := range pageMap {
			values += fmt.Sprintf("(%d, %d, '%s', '%s'),",
				data.User.Id, pg.PageId, database.Now(), database.Now())
		}
		values = strings.TrimRight(values, ",")
		sql := fmt.Sprintf(`
			INSERT INTO visits (userId, pageId, createdAt, updatedAt)
			VALUES %s
			ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`, values)
		if _, err = database.ExecuteSql(c, sql); err != nil {
			c.Errorf("Couldn't update visits: %v", err)
		}

		// Load updates count.
		data.User.UpdateCount, err = loadUpdateCount(c, data.User.Id)
		if err != nil {
			c.Errorf("Couldn't retrieve updates count: %v", err)
		}
	}

	funcMap := template.FuncMap{
		"IsUpdatedPage": func(p *richPage) bool {
			return p.Author.Id != data.User.Id && p.LastVisit != "" && p.CreatedAt >= p.LastVisit
		},
		"IsNewComment": func(c *comment) bool {
			lastVisit := pageMap[c.PageId].LastVisit
			return c.Author.Id != data.User.Id && lastVisit != "" && c.CreatedAt >= lastVisit
		},
		"IsUpdatedComment": func(c *comment) bool {
			lastVisit := pageMap[c.PageId].LastVisit
			return c.Author.Id != data.User.Id && lastVisit != "" && c.UpdatedAt >= lastVisit
		},
		// Check if we should even bother showing edit and delete page icons.
		"ShowEditIcons": func(p *richPage) bool {
			return getEditLevel(&p.page, data.User) >= 0 || getDeleteLevel(&p.page, data.User) >= 0
		},
		"GetEditLevel": func(p *richPage) int {
			return getEditLevel(&p.page, data.User)
		},
		"GetDeleteLevel": func(p *richPage) int {
			return getDeleteLevel(&p.page, data.User)
		},
		"GetPageUrl": func(p *richPage) string {
			return getPageUrl(&p.page)
		},
		"GetPageEditUrl": func(p *richPage) string {
			return getEditPageUrl(&p.page)
		},
		"GetUserUrl": func(userId int64) string {
			return getUserUrl(userId)
		},
		"GetTagUrl": func(tagId int64) string {
			return getTagUrl(tagId)
		},
		"Sanitize": func(s string) template.HTML {
			s = template.HTMLEscapeString(s)
			s = strings.Replace(s, "\n", "<br>", -1)
			return template.HTML(s)
		},
	}
	c.Inc("page_page_served_success")
	return pages.StatusOK(data).AddFuncMap(funcMap)
}