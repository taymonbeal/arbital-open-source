// claim.go serves the claim page.
package site

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/user"

	"github.com/gorilla/mux"
)

type comment struct {
	Id      int64
	ClaimId int64
	//ContextClaimId int64
	ReplyToId    int64
	Text         string
	CreatedAt    string
	UpdatedAt    string
	CreatorId    int64
	CreatorName  string
	UpvoteCount  int
	MyVote       int
	IsSubscribed bool
	Replies      []*comment
}

type byDate []comment

func (a byDate) Len() int           { return len(a) }
func (a byDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDate) Less(i, j int) bool { return a[i].CreatedAt < a[j].CreatedAt }

type input struct {
	Id          int64
	ChildId     int64
	CreatedAt   string
	UpdatedAt   string
	CreatorId   int64
	CreatorName string
}

type tag struct {
	Id   int64
	Text string
}

type claim struct {
	Id            int64
	Summary       string
	Text          string
	Url           string
	CreatorId     int64
	CreatorName   string
	CreatedAt     string
	UpdatedAt     string
	LastVisit     string
	PrivacyKey    sql.NullInt64
	InputCount    int
	IsSubscribed  bool
	IsParentClaim bool
	UpvoteCount   int
	DownvoteCount int
	MyVote        int
	Contexts      []*claim
	Comments      []*comment
	Tags          []*tag
}

// claimTmplData stores the data that we pass to the index.tmpl to render the page
type claimTmplData struct {
	User   *user.User
	Claim  *claim
	Claims []*claim
	Inputs []*input
}

// claimPage serves the claim page.
var claimPage = pages.Add(
	"/claims/{id:[0-9]+}",
	claimRenderer,
	append(baseTmpls,
		"tmpl/claimPage.tmpl", "tmpl/claim.tmpl",
		"tmpl/comment.tmpl", "tmpl/newComment.tmpl",
		"tmpl/navbar.tmpl")...)

var privateClaimPage = pages.Add(
	"/claims/{id:[0-9]+}/{privacyKey:[0-9]+}",
	claimRenderer,
	append(baseTmpls,
		"tmpl/claimPage.tmpl", "tmpl/claim.tmpl",
		"tmpl/comment.tmpl", "tmpl/newComment.tmpl",
		"tmpl/navbar.tmpl")...)

// loadParentClaim loads and returns the parent claim.
func loadParentClaim(c sessions.Context, userId int64, claimId string) (*claim, error) {
	c.Infof("querying DB for claim with id = %s\n", claimId)
	parentClaim := &claim{}
	query := fmt.Sprintf(`
		SELECT id,summary,text,url,creatorId,creatorName,createdAt,updatedAt,privacyKey
		FROM claims
		WHERE id=%s`, claimId)
	exists, err := database.QueryRowSql(c, query, &parentClaim.Id,
		&parentClaim.Summary, &parentClaim.Text, &parentClaim.Url,
		&parentClaim.CreatorId, &parentClaim.CreatorName,
		&parentClaim.CreatedAt, &parentClaim.UpdatedAt, &parentClaim.PrivacyKey)
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve a claim: %v", err)
	} else if !exists {
		return nil, fmt.Errorf("Unknown claim id: %s", claimId)
	}

	// Load tags.
	err = loadTags(c, parentClaim)
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve claim tags: %v", err)
	}

	// Load contexts.
	query = fmt.Sprintf(`
		SELECT c.id,c.summary,c.text,c.privacyKey
		FROM inputs as i
		JOIN claims as c
		ON i.parentId=c.id
		WHERE i.childId=%s`, claimId)
	err = database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var cl claim
		err := rows.Scan(&cl.Id, &cl.Summary, &cl.Text, &cl.PrivacyKey)
		if err != nil {
			return fmt.Errorf("failed to scan for context claim: %v", err)
		}
		parentClaim.Contexts = append(parentClaim.Contexts, &cl)
		return nil
	})
	parentClaim.IsParentClaim = true
	return parentClaim, err
}

// loadChildClaims loads and returns all claims and inputs that have the given parent claim.
func loadChildClaims(c sessions.Context, claimId string) ([]*input, []*claim, error) {
	inputs := make([]*input, 0)
	claims := make([]*claim, 0)

	c.Infof("querying DB for child claims for parent id=%s\n", claimId)
	query := fmt.Sprintf(`
		SELECT i.id,i.childId,i.createdAt,i.updatedAt,i.creatorId,i.creatorName,
			c.id,c.summary,c.text,c.url,c.creatorId,c.creatorName,c.createdAt,c.updatedAt,c.privacyKey
		FROM inputs as i
		JOIN claims as c
		ON i.childId=c.id
		WHERE i.parentId=%s`, claimId)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var i input
		var cl claim
		err := rows.Scan(
			&i.Id, &i.ChildId,
			&i.CreatedAt, &i.UpdatedAt,
			&i.CreatorId, &i.CreatorName,
			&cl.Id, &cl.Summary, &cl.Text, &cl.Url,
			&cl.CreatorId, &cl.CreatorName,
			&cl.CreatedAt, &cl.UpdatedAt,
			&cl.PrivacyKey)
		if err != nil {
			return fmt.Errorf("failed to scan for input: %v", err)
		}
		inputs = append(inputs, &i)
		claims = append(claims, &cl)
		return nil
	})
	return inputs, claims, err
}

// loadInputCounts computes how many inputs each claim has.
func loadInputCounts(c sessions.Context, claimIds string, claimMap map[int64]*claim) error {
	query := fmt.Sprintf(`
		SELECT parentId,sum(1)
		FROM inputs
		WHERE parentId IN (%s)
		GROUP BY parentId`, claimIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var parentId int64
		var count int
		err := rows.Scan(&parentId, &count)
		if err != nil {
			return fmt.Errorf("failed to scan for an input: %v", err)
		}
		claimMap[parentId].InputCount = count
		return nil
	})
	return err
}

// loadComments loads and returns all the comments for the given input ids from the db.
func loadComments(c sessions.Context, claimIds string) (map[int64]*comment, []int64, error) {
	commentMap := make(map[int64]*comment)
	sortedCommentIds := make([]int64, 0)

	c.Infof("querying DB for comments with claimIds = %v", claimIds)
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT id,claimId,replyToId,text,createdAt,updatedAt,creatorId,creatorName
		FROM comments
		WHERE claimId IN (%s)`, /*AND (contextClaimId=0 OR contextClaimId=%s)`*/ claimIds /*, claimId*/)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var ct comment
		err := rows.Scan(
			&ct.Id,
			&ct.ClaimId,
			//&ct.ContextClaimId,
			&ct.ReplyToId,
			&ct.Text,
			&ct.CreatedAt,
			&ct.UpdatedAt,
			&ct.CreatorId,
			&ct.CreatorName)
		if err != nil {
			return fmt.Errorf("failed to scan for comments: %v", err)
		}
		commentMap[ct.Id] = &ct
		sortedCommentIds = append(sortedCommentIds, ct.Id)
		return nil
	})
	return commentMap, sortedCommentIds, err
}

// loadVotes loads votes corresponding to the given claims and updates the claims.
func loadVotes(c sessions.Context, currentUserId int64, claimIds string, claimMap map[int64]*claim) error {
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT userId,claimId,value
		FROM (SELECT * FROM votes ORDER BY id DESC) AS v
		WHERE claimId IN (%s)
		GROUP BY userId,claimId`, claimIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var userId int64
		var claimId int64
		var value int
		err := rows.Scan(&userId, &claimId, &value)
		if err != nil {
			return fmt.Errorf("failed to scan for a vote: %v", err)
		}
		claim := claimMap[claimId]
		if value > 0 {
			claim.UpvoteCount++
		} else if value < 0 {
			claim.DownvoteCount++
		}
		if userId == currentUserId {
			claim.MyVote = value
		}
		return nil
	})
	return err
}

// loadCommentVotes loads votes corresponding to the given comments and updates the comments.
func loadCommentVotes(c sessions.Context, currentUserId int64, commentIds string, commentMap map[int64]*comment) error {
	// Workaround for: https://github.com/go-sql-driver/mysql/issues/304
	query := fmt.Sprintf(`
		SELECT userId,commentId,value
		FROM commentVotes
		WHERE commentId IN (%s)`, commentIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var userId int64
		var commentId int64
		var value int
		err := rows.Scan(&userId, &commentId, &value)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment vote: %v", err)
		}
		comment := commentMap[commentId]
		if value > 0 {
			comment.UpvoteCount++
		}
		if userId == currentUserId {
			comment.MyVote = value
		}
		return nil
	})
	return err
}

// loadLastVisits loads lastVisit variable for each claim.
func loadLastVisits(c sessions.Context, currentUserId int64, claimIds string, claimMap map[int64]*claim) error {
	query := fmt.Sprintf(`
		SELECT claimId,updatedAt
		FROM visits
		WHERE userId=%d AND claimId IN (%s)`,
		currentUserId, claimIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var claimId int64
		var updatedAt string
		err := rows.Scan(&claimId, &updatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment vote: %v", err)
		}
		claimMap[claimId].LastVisit = updatedAt
		return nil
	})
	return err
}

// loadSubscriptions loads subscription statuses corresponding to the given
// claims and comments, and then updates the given maps.
func loadSubscriptions(
	c sessions.Context, currentUserId int64,
	claimIds string, commentIds string,
	claimMap map[int64]*claim,
	commentMap map[int64]*comment) error {

	query := fmt.Sprintf(`
		SELECT claimId,commentId
		FROM subscriptions
		WHERE userId=%d AND (claimId IN (%s) OR commentId IN (%s))`,
		currentUserId, claimIds, commentIds)
	err := database.QuerySql(c, query, func(c sessions.Context, rows *sql.Rows) error {
		var claimId int64
		var commentId int64
		err := rows.Scan(&claimId, &commentId)
		if err != nil {
			return fmt.Errorf("failed to scan for a comment vote: %v", err)
		}
		if claimId > 0 {
			claimMap[claimId].IsSubscribed = true
		} else if commentId > 0 {
			commentMap[commentId].IsSubscribed = true
		}
		return nil
	})
	return err
}

// claimRenderer renders the claim page.
func claimRenderer(w http.ResponseWriter, r *http.Request) *pages.Result {
	var data claimTmplData
	c := sessions.NewContext(r)

	var err error
	/*db, err := database.GetDB(c)
	if err != nil {
		c.Errorf("error while getting DB: %v", err)
		return pages.InternalErrorWith(err)
	}*/

	// Load user, if possible
	data.User, err = user.LoadUser(w, r)
	if err != nil {
		c.Errorf("Couldn't load user: %v", err)
		return pages.InternalErrorWith(err)
	}

	// Load the parent claim
	claimMap := make(map[int64]*claim)
	claimId := mux.Vars(r)["id"]
	parentClaim, err := loadParentClaim(c, data.User.Id, claimId)
	if err != nil {
		c.Inc("claim_fetch_fail")
		c.Errorf("error while fetching a claim: %v", err)
		return pages.InternalErrorWith(err)
	}
	claimMap[parentClaim.Id] = parentClaim
	data.Claim = parentClaim

	// Check privacy setting
	if parentClaim.PrivacyKey.Valid {
		privacyKey := mux.Vars(r)["privacyKey"]
		if privacyKey != fmt.Sprintf("%d", parentClaim.PrivacyKey.Int64) {
			return pages.UnauthorizedWith(err)
		}
	}

	// Load all the inputs and corresponding child claims
	data.Inputs, data.Claims, err = loadChildClaims(c, claimId)
	if err != nil {
		c.Inc("inputs_fetch_fail")
		c.Errorf("error while fetching input for claim id: %s\n%v", claimId, err)
		return pages.InternalErrorWith(err)
	}

	// Get a string of all claim ids and populate claimMap
	var buffer bytes.Buffer
	for _, c := range data.Claims {
		claimMap[c.Id] = c
		buffer.WriteString(fmt.Sprintf("%d", c.Id))
		buffer.WriteString(",")
	}
	buffer.WriteString(claimId)
	claimIds := buffer.String()

	// Load input counts
	err = loadInputCounts(c, claimIds, claimMap)
	if err != nil {
		c.Inc("inputs_fetch_fail")
		c.Errorf("error while fetching inputs: %v", err)
		return pages.InternalErrorWith(err)
	}

	// Load all the votes
	err = loadVotes(c, data.User.Id, claimIds, claimMap)
	if err != nil {
		c.Inc("votes_fetch_fail")
		c.Errorf("error while fetching votes: %v", err)
		return pages.InternalErrorWith(err)
	}

	// Get last visits.
	q := r.URL.Query()
	forcedLastVisit := q.Get("lastVisit")
	if forcedLastVisit == "" {
		err = loadLastVisits(c, data.User.Id, claimIds, claimMap)
		if err != nil {
			c.Errorf("error while fetching a visit: %v", err)
		}
	} else {
		for _, cl := range claimMap {
			cl.LastVisit = forcedLastVisit
		}
	}

	// Load all the comments
	var commentMap map[int64]*comment // commentId -> comment
	var sortedCommentKeys []int64     // need this for in-order iteration
	commentMap, sortedCommentKeys, err = loadComments(c, claimIds)
	if err != nil {
		c.Inc("comments_fetch_fail")
		c.Errorf("error while fetching comments: %v", err)
		return pages.InternalErrorWith(err)
	}
	for _, key := range sortedCommentKeys {
		comment := commentMap[key]
		claimObj, ok := claimMap[comment.ClaimId]
		if !ok {
			c.Errorf("couldn't find claim for a comment: %d\n%v", key, err)
			return pages.InternalErrorWith(err)
		}
		if comment.ReplyToId > 0 {
			parent := commentMap[comment.ReplyToId]
			parent.Replies = append(parent.Replies, commentMap[key])
		} else {
			claimObj.Comments = append(claimObj.Comments, commentMap[key])
		}
	}

	// Get a string of all comment ids.
	buffer.Reset()
	for id, _ := range commentMap {
		buffer.WriteString(fmt.Sprintf("%d", id))
		buffer.WriteString(",")
	}
	commentIds := buffer.String()
	if len(commentIds) <= 0 {
		commentIds = "-1"
	} else {
		commentIds = commentIds[0 : len(commentIds)-1]
	}

	// Load all the comment votes
	err = loadCommentVotes(c, data.User.Id, commentIds, commentMap)
	if err != nil {
		c.Inc("comment_votes_fetch_fail")
		c.Errorf("error while fetching comment votes: %v", err)
		return pages.InternalErrorWith(err)
	}

	if data.User.Id > 0 {
		// Load subscription statuses.
		err = loadSubscriptions(c, data.User.Id, claimIds, commentIds, claimMap, commentMap)
		if err != nil {
			c.Inc("subscriptions_fetch_fail")
			c.Errorf("error while fetching subscriptions: %v", err)
			return pages.InternalErrorWith(err)
		}

		// From here on we can render the page successfully. Further queries are nice,
		// but not mandatory, so we are not going to return an error if they fail.

		// Mark the newInput updates for the parent claim as read.
		query := fmt.Sprintf(
			`UPDATE updates
			SET seen=1,updatedAt='%s'
			WHERE claimId=%s AND userId=%d AND type='newInput'`,
			database.Now(), claimId, data.User.Id)
		if _, err := database.ExecuteSql(c, query); err != nil {
			c.Errorf("Couldn't update updates: %v", err)
		}

		// Mark all other types of updates related to loaded claims as seen.
		query = fmt.Sprintf(`
			UPDATE updates
			SET seen=1,updatedAt='%s'
			WHERE claimId IN (%s) AND userId=%d AND type!='newInput'`,
			database.Now(), claimIds, data.User.Id)
		if _, err := database.ExecuteSql(c, query); err != nil {
			c.Errorf("Couldn't update updates: %v", err)
		}

		// Update last visit date.
		values := ""
		for _, cl := range claimMap {
			values += fmt.Sprintf("(%d, %d, '%s', '%s'),",
				data.User.Id, cl.Id, database.Now(), database.Now())
		}
		values = values[0 : len(values)-1] // remove last comma
		sql := fmt.Sprintf(`
			INSERT INTO visits (userId, claimId, createdAt, updatedAt)
			VALUES %s
			ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`, values)
		if _, err = database.ExecuteSql(c, sql); err != nil {
			c.Errorf("Couldn't update visits: %v", err)
		}

		// Load updates count.
		query = fmt.Sprintf(`
			SELECT COALESCE(SUM(count), 0)
			FROM updates
			WHERE userId=%d AND seen=0`, data.User.Id)
		_, err = database.QueryRowSql(c, query, &data.User.UpdateCount)
		if err != nil {
			c.Errorf("Couldn't retrieve updates count: %v", err)
		}
	}

	funcMap := template.FuncMap{
		"UserId":     func() int64 { return data.User.Id },
		"IsAdmin":    func() bool { return data.User.IsAdmin },
		"IsLoggedIn": func() bool { return data.User.IsLoggedIn },
		"IsNewClaim": func(c *claim) bool {
			return c.CreatorId != data.User.Id && c.LastVisit != "" && c.CreatedAt >= c.LastVisit
		},
		"IsUpdatedClaim": func(c *claim) bool {
			return c.CreatorId != data.User.Id && c.LastVisit != "" && c.UpdatedAt >= c.LastVisit
		},
		"IsNewComment": func(c *comment) bool {
			lastVisit := claimMap[c.ClaimId].LastVisit
			return c.CreatorId != data.User.Id && lastVisit != "" && c.CreatedAt >= lastVisit
		},
		"IsUpdatedComment": func(c *comment) bool {
			lastVisit := claimMap[c.ClaimId].LastVisit
			return c.CreatorId != data.User.Id && lastVisit != "" && c.UpdatedAt >= lastVisit
		},
		"GetClaimUrl": func(c *claim) string {
			privacyAddon := ""
			if c.PrivacyKey.Valid {
				privacyAddon = fmt.Sprintf("/%d", c.PrivacyKey.Int64)
			}
			return fmt.Sprintf("/claims/%d%s", c.Id, privacyAddon)
		},
		"Sanitize": func(s string) template.HTML {
			s = template.HTMLEscapeString(s)
			s = strings.Replace(s, "\n", "<br>", -1)
			return template.HTML(s)
		},
	}
	c.Inc("claim_page_served_success")
	return pages.StatusOK(data).SetFuncMap(funcMap)
}
