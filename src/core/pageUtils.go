// pageUtils.go contains various helpers for dealing with pages
package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"zanaduu3/src/database"
	"zanaduu3/src/sessions"
)

const (
	// Helpers for matching our markdown extensions
	SpacePrefix   = "(^| |\n)"
	NoParenSuffix = "($|[^(])"

	// Strings that can be used inside a regexp to match on a page alias or id.
	AliasOrPageIDRegexpStr          = "[0-9A-Za-z_]+\\.?[0-9A-Za-z_]*"
	SubdomainAliasOrPageIDRegexpStr = "[0-9A-Za-z_]*"

	// Used for replacing non-alias characters.
	ReplaceRegexpStr = "[^A-Za-z0-9_]"

	Base31Chars             = "0123456789bcdfghjklmnpqrstvwxyz"
	Base31CharsForFirstChar = "0123456789"

	StubPageID                    = "72"
	RequestForEditTagParentPageID = "3zj"
	QualityMetaTagsPageID         = "5dg"
	AClassPageID                  = "4yf"
	BClassPageID                  = "4yd"
	FeaturedClassPageID           = "4yl"
	HubPageID                     = "5ls"
	ConceptPageID                 = "6cc"

	MathDomainID = 1
)

// AddPageToMap adds a new page with the given page id to the map if it's not
// in the map already.
// Returns the new/existing page.
func AddPageToMap(pageID string, pageMap map[string]*Page, loadOptions *PageLoadOptions) *Page {
	if !IsIDValid(pageID) {
		return nil
	}
	if p, ok := pageMap[pageID]; ok {
		p.LoadOptions.Add(loadOptions)
		return p
	}
	p := NewPage(pageID)
	p.LoadOptions = *loadOptions
	pageMap[pageID] = p
	return p
}
func AddPageIDToMap(pageID string, pageMap map[string]*Page) *Page {
	return AddPageToMap(pageID, pageMap, EmptyLoadOptions)
}

// AddUserToMap adds a new user with the given user id to the map if it's not
// in the map already.
// Returns the new/existing user.
func AddUserToMap(userID string, userMap map[string]*User) *User {
	if !IsIDValid(userID) {
		return nil
	}
	if u, ok := userMap[userID]; ok {
		return u
	}
	u := &User{coreUserData: coreUserData{ID: userID}}
	userMap[userID] = u
	return u
}

// Add a markId to the mark map if it's not there already.
func AddMarkToMap(markID string, markMap map[string]*Mark) *Mark {
	mark, ok := markMap[markID]
	if !ok {
		mark = &Mark{ID: markID}
		markMap[markID] = mark
	}
	return mark
}

// PageIdsListFromMap returns a comma separated string of all pageIds in the given map.
func PageIDsListFromMap(pageMap map[string]*Page) []interface{} {
	list := make([]interface{}, 0, len(pageMap))
	for id := range pageMap {
		list = append(list, id)
	}
	return list
}

// StandardizeLinks converts all alias links into pageId links.
func StandardizeLinks(db *database.DB, text string) (string, error) {
	// Populate a list of all the links
	aliasesAndIDs := make([]string, 0)
	// Track regexp matches, because ReplaceAllStringFunc doesn't support matching groups
	matches := make(map[string][]string)
	extractLinks := func(exp *regexp.Regexp) {
		submatches := exp.FindAllStringSubmatch(text, -1)
		for _, submatch := range submatches {
			matches[submatch[0]] = submatch
			lowerCaseString := strings.ToLower(submatch[2][:1]) + submatch[2][1:]
			upperCaseString := strings.ToUpper(submatch[2][:1]) + submatch[2][1:]
			aliasesAndIDs = append(aliasesAndIDs, lowerCaseString)
			aliasesAndIDs = append(aliasesAndIDs, upperCaseString)
		}
	}

	// NOTE: these regexps are waaaay too simplistic and don't account for the
	// entire complexity of Markdown, like 4 spaces, backticks, and escaped
	// brackets / parens.
	// NOTE: each regexp should have two groups that captures stuff that comes before
	// the alias, and then 0 or more groups that capture everything after
	// NOTE: we have to be careful about capturing too much around the link, because
	// if we do, the second link in `[alias] [alias]` won't get captured. For this reason
	// we check for backtick only right at the end of the expression.
	notInBackticks := "([^`]|$)"
	regexps := []*regexp.Regexp{
		// Find directly encoded urls
		regexp.MustCompile("(/p/)(" + AliasOrPageIDRegexpStr + ")" + notInBackticks),
		// Find ids and aliases using [alias optional text] syntax.
		regexp.MustCompile("(\\[[\\-\\+]?)(" + AliasOrPageIDRegexpStr + ")( [^\\]]*?)?(\\])([^(`]|$)"),
		// Find ids and aliases using [text](alias) syntax.
		regexp.MustCompile("(\\[[^\\]]+?\\]\\()(" + AliasOrPageIDRegexpStr + ")(\\))" + notInBackticks),
		// Find ids and aliases using [vote: alias] syntax.
		regexp.MustCompile("(\\[vote: ?)(" + AliasOrPageIDRegexpStr + ")(\\])" + notInBackticks),
		// Find ids and aliases using [@alias] syntax.
		regexp.MustCompile("(\\[@)(" + AliasOrPageIDRegexpStr + ")(\\])([^(`]|$)"),
	}
	for _, exp := range regexps {
		extractLinks(exp)
	}

	if len(aliasesAndIDs) <= 0 {
		return text, nil
	}

	// Populate alias -> pageId map
	aliasMap := make(map[string]string)
	rows := database.NewQuery(`
		SELECT pageId,alias
		FROM`).AddPart(PageInfosTable(nil)).Add(`AS pi
		WHERE alias IN`).AddArgsGroupStr(aliasesAndIDs).ToStatement(db).Query()
	err := rows.Process(func(db *database.DB, rows *database.Rows) error {
		var pageID, alias string
		err := rows.Scan(&pageID, &alias)
		if err != nil {
			return fmt.Errorf("failed to scan: %v", err)
		}
		aliasMap[alias] = pageID
		return nil
	})
	if err != nil {
		return "", err
	}

	// Perform replacement
	replaceAlias := func(match string) string {
		submatch := matches[match]

		lowerCaseString := strings.ToLower(submatch[2][:1]) + submatch[2][1:]
		upperCaseString := strings.ToUpper(submatch[2][:1]) + submatch[2][1:]

		// Since ReplaceAllStringFunc gives us the whole match, rather than submatch
		// array, we have stored it earlier and can now piece it together
		if id, ok := aliasMap[lowerCaseString]; ok {
			return submatch[1] + id + strings.Join(submatch[3:], "")
		}
		if id, ok := aliasMap[upperCaseString]; ok {
			return submatch[1] + id + strings.Join(submatch[3:], "")
		}
		return match
	}
	for _, exp := range regexps {
		text = exp.ReplaceAllStringFunc(text, replaceAlias)
	}
	return text, nil
}

// Extract all links from the given page text.
func ExtractPageLinks(text string, configAddress string) []string {
	// NOTE: these regexps are waaaay too simplistic and don't account for the
	// entire complexity of Markdown, like 4 spaces, backticks, and escaped
	// brackets / parens.
	linkMap := make(map[string]bool)
	extractLinks := func(exp *regexp.Regexp) {
		submatches := exp.FindAllStringSubmatch(text, -1)
		for _, submatch := range submatches {
			linkMap[submatch[1]] = true
		}
	}
	// Find directly encoded urls
	extractLinks(regexp.MustCompile(regexp.QuoteMeta(configAddress) + "/p(?:ages)?/(" + AliasOrPageIDRegexpStr + ")"))
	// Find ids and aliases using [alias optional text] syntax.
	extractLinks(regexp.MustCompile("\\[[\\-\\+]?(" + AliasOrPageIDRegexpStr + ")(?: [^\\]]*?)?\\](?:[^(]|$)"))
	// Find ids and aliases using [text](alias) syntax.
	extractLinks(regexp.MustCompile("\\[.+?\\]\\((" + AliasOrPageIDRegexpStr + ")\\)"))
	// Find ids and aliases using [vote: alias] syntax.
	extractLinks(regexp.MustCompile("\\[vote: ?(" + AliasOrPageIDRegexpStr + ")\\]"))
	// Find ids and aliases using [@alias] syntax.
	extractLinks(regexp.MustCompile("\\[@?(" + AliasOrPageIDRegexpStr + ")\\]"))

	aliasesAndIDs := make([]string, 0)
	for alias := range linkMap {
		aliasesAndIDs = append(aliasesAndIDs, alias)
	}
	return aliasesAndIDs
}

// UpdatePageLinks updates the links table for the given page by parsing the text.
func UpdatePageLinks(tx *database.Tx, pageID string, text string, configAddress string) error {
	// Delete old links.
	statement := tx.DB.NewStatement("DELETE FROM links WHERE parentId=?").WithTx(tx)
	_, err := statement.Exec(pageID)
	if err != nil {
		return fmt.Errorf("Couldn't delete old links: %v", err)
	}

	aliasesAndIDs := ExtractPageLinks(text, configAddress)
	if len(aliasesAndIDs) > 0 {
		// Populate linkTuples
		linkMap := make(map[string]bool) // track which aliases we already added to the list
		valuesList := make([]interface{}, 0)
		for _, alias := range aliasesAndIDs {
			lowercaseAlias := strings.ToLower(alias)
			if linkMap[lowercaseAlias] {
				continue
			}
			valuesList = append(valuesList, pageID, lowercaseAlias)
			linkMap[lowercaseAlias] = true
		}

		// Insert all the tuples into the links table.
		statement := tx.DB.NewStatement(`
			INSERT INTO links (parentId,childAlias)
			VALUES ` + database.ArgsPlaceholder(len(valuesList), 2)).WithTx(tx)
		if _, err = statement.Exec(valuesList...); err != nil {
			return fmt.Errorf("Couldn't insert links: %v", err)
		}
	}
	return nil
}

// GetPageLockedUntilTime returns time until the user can have the lock if the locked
// the page right now.
func GetPageLockedUntilTime() string {
	return time.Now().UTC().Add(PageLockDuration * time.Second).Format(database.TimeLayout)
}
func GetPageQuickLockedUntilTime() string {
	return time.Now().UTC().Add(PageQuickLockDuration * time.Second).Format(database.TimeLayout)
}

// ExtractSummaries extracts the summaries from the given page text.
func ExtractSummaries(pageID string, text string) (map[string]string, []interface{}) {
	const defaultSummary = "Summary"
	re := regexp.MustCompile("(?ms)^\\[summary(\\([^)]+\\))?: ?([\\s\\S]+?)\\] *(\\z|\n\\z|\n\n)")
	summaries := make(map[string]string)

	submatches := re.FindAllStringSubmatch(text, -1)
	for _, submatch := range submatches {
		name := strings.Trim(submatch[1], "()")
		text := submatch[2]
		if name == "" {
			name = defaultSummary
		}
		summaries[name] = strings.TrimSpace(text)
	}
	if _, ok := summaries[defaultSummary]; !ok {
		// If no summaries, just extract the first line.
		re = regexp.MustCompile("^(.*)")
		submatch := re.FindStringSubmatch(text)
		summaries[defaultSummary] = strings.TrimSpace(submatch[1])
	}

	// Compute values for doing INSERT
	summaryValues := make([]interface{}, 0)
	for name, text := range summaries {
		summaryValues = append(summaryValues, pageID, name, text)
	}
	return summaries, summaryValues
}

// ExtractTodoCount extracts the number of todos from a page text.
func ExtractTodoCount(text string) int {
	// Match [todo: text] or |todo: text| or ||todo: text|| (any number of vertical bars)

	// Regexp for todo with brackets, [todo: text]
	re := regexp.MustCompile("\\[todo: ?[^\\]]*\\]")
	submatches := re.FindAllString(text, -1)
	// Regexp for todo with vertical bars, |todo: text|, ||todo: text|| etc.
	re = regexp.MustCompile("\\|+?todo: ?[^\\|]*\\|+")
	submatches = append(submatches, re.FindAllString(text, -1)...)
	todoCount := len(submatches)

	// Match [ red link text]
	re = regexp.MustCompile("\\[ [^\\]]+\\]")
	submatches = re.FindAllString(text, -1)
	return todoCount + len(submatches)
}

// GetPageFullUrl returns the full url for accessing the given page.
func GetPageFullURL(subdomain string, pageID string) string {
	if len(subdomain) > 0 {
		subdomain += "."
	}
	domain := strings.TrimPrefix(sessions.GetRawDomain(), "http://")
	return fmt.Sprintf("http://%s%s/p/%s", subdomain, domain, pageID)
}

// GetEditPageFullUrl returns the full url for editing the given page.
func GetEditPageFullURL(subdomain string, pageID string) string {
	if len(subdomain) > 0 {
		subdomain += "."
	}
	domain := strings.TrimPrefix(sessions.GetRawDomain(), "http://")
	return fmt.Sprintf("http://%s%s/edit/%s", subdomain, domain, pageID)
}

// CorrectPageType converts the page type to lowercase and checks that it's
// an actual page type we support.
func CorrectPageType(pageType string) (string, error) {
	pageType = strings.ToLower(pageType)
	if pageType != WikiPageType &&
		pageType != QuestionPageType &&
		pageType != CommentPageType &&
		pageType != GroupPageType {
		return pageType, fmt.Errorf("Invalid page type: %s", pageType)
	}
	return pageType, nil
}
func CorrectPagePairType(pagePairType string) (string, error) {
	pagePairType = strings.ToLower(pagePairType)
	if pagePairType != ParentPagePairType &&
		pagePairType != TagPagePairType &&
		pagePairType != RequirementPagePairType &&
		pagePairType != SubjectPagePairType {
		return pagePairType, fmt.Errorf("Incorrect type: %s", pagePairType)
	}
	return pagePairType, nil
}

func IsIDValid(pageID string) bool {
	if len(pageID) > 0 && pageID[0] > '0' && pageID[0] <= '9' {
		return true
	}
	return false
}

// We store int64 ids in strings. Because of that invalid ids can have two values: "" and "0".
// Check if the given int64 id is valid.
func IsIntIDValid(id string) bool {
	return id != "" && id != "0"
}

// Check if the given alias is valid
func IsAliasValid(alias string) bool {
	return regexp.MustCompile("^" + AliasOrPageIDRegexpStr + "$").MatchString(alias)
}

func GetCommentParents(db *database.DB, pageID string) (string, string, error) {
	var commentParentID string
	var commentPrimaryPageID string
	rows := database.NewQuery(`
		SELECT pi.pageId,pi.type
		FROM`).AddPart(PageInfosTable(nil)).Add(`AS pi
		JOIN pagePairs AS pp
		ON (pi.pageId=pp.parentId)
		WHERE pp.type=?`, ParentPagePairType).Add(`
			AND pp.childId=?`, pageID).Add(`
		`).ToStatement(db).Query()
	err := rows.Process(func(db *database.DB, rows *database.Rows) error {
		var parentID string
		var pageType string
		err := rows.Scan(&parentID, &pageType)
		if err != nil {
			return fmt.Errorf("failed to scan: %v", err)
		}
		if pageType == CommentPageType {
			if IsIDValid(commentParentID) {
				return fmt.Errorf("Can't have more than one comment parent")
			}
			commentParentID = parentID
		} else {
			if IsIDValid(commentPrimaryPageID) {
				return fmt.Errorf("Can't have more than one non-comment parent for a comment")
			}
			commentPrimaryPageID = parentID
		}
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to process rows: %v", err)
	} else if !IsIDValid(commentPrimaryPageID) {
		return "", "", fmt.Errorf("Comment pages need at least one normal page parent")
	}

	return commentParentID, commentPrimaryPageID, nil
}

// Return true iff the string is in the list
func IsStringInList(str string, list []string) bool {
	for _, listID := range list {
		if str == listID {
			return true
		}
	}
	return false
}
