// databaseUtils.go contains various helpers for dealing with database and tables
package core

import (
	"fmt"
	"strings"

	"zanaduu3/src/database"
	"zanaduu3/src/sessions"
)

var (
	allPageInfosOptions = &PageInfosOptions{
		Unpublished: true,
		Deleted:     true,
	}
)

// What pages to load from pageInfos table.
type PageInfosOptions struct {
	Unpublished bool
	Deleted     bool
	Fields      []string
	WhereFilter *database.QueryPart

	// Options for WHERE clause
	OmitPrefix bool
}

// PageInfosTable is a wrapper for loading data from the pageInfos table.
// It filters all the pages to make sure the current can actually access them.
// It also filters out any pages that are deleted or aren't published.
func PageInfosTable(u *CurrentUser) *database.QueryPart {
	return PageInfosTableWithOptions(u, &PageInfosOptions{})
}

// Like PageInfosTable but allows for autosaves, snapshots, and deleted pages.
func PageInfosTableAll(u *CurrentUser) *database.QueryPart {
	return PageInfosTableWithOptions(u, allPageInfosOptions)
}

// PageInfosTableWithOptions is a wrapper for loading data from the pageInfos table.
func PageInfosTableWithOptions(u *CurrentUser, options *PageInfosOptions) *database.QueryPart {
	if u == nil && options.Unpublished && options.Deleted {
		return database.NewQuery(`pageInfos`)
	}

	var fieldsString string
	if options.Fields != nil && len(options.Fields) > 0 {
		for i, f := range options.Fields {
			fieldsString += f
			if i > 0 {
				fieldsString += ","
			}
		}
	} else {
		fieldsString = "*"
	}

	options.OmitPrefix = true
	q := database.NewQuery(`(
			SELECT ` + fieldsString + `
			FROM pageInfos
			WHERE`).AddPart(PageInfosFilterWithOptions(u, options)).Add(`
		)`)
	return q
}

func PageInfosFilter(u *CurrentUser) *database.QueryPart {
	return PageInfosFilterWithOptions(u, &PageInfosOptions{})
}

func PageInfosFilterAll(u *CurrentUser) *database.QueryPart {
	return PageInfosFilterWithOptions(u, allPageInfosOptions)
}

// PageInfosFilterWithOptions returns a clase for making sure the selected rows from pageInfos table are valid.
func PageInfosFilterWithOptions(u *CurrentUser, options *PageInfosOptions) *database.QueryPart {
	prefix := `pi.`
	if options.OmitPrefix {
		prefix = ``
	}
	q := database.NewQuery(`(TRUE`)
	if u != nil {
		allowedDomainIDs := []string{"0"}
		for domainID := range u.DomainMembershipMap {
			if CanUserSeeDomain(u, domainID) {
				allowedDomainIDs = append(allowedDomainIDs, domainID)
			}
		}
		q.Add(`AND ` + prefix + `seeDomainId IN`).AddArgsGroupStr(allowedDomainIDs)
	}
	if !options.Unpublished {
		q.Add(`AND ` + prefix + `currentEdit > 0`)
	}
	if !options.Deleted {
		q.Add(`AND NOT ` + prefix + `isDeleted`)
	}
	if options.WhereFilter != nil {
		q.Add(`AND (`).AddPart(options.WhereFilter).Add(`)`)
	}
	return q.Add(`)`)
}

// Replace a rune at a specific index in a string
func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

// Get the next highest base36 character, without vowels
// Returns the character, and true if it wrapped around to 0
// Since we decided that ids must begin with a digit, only allow characters 0-9 for the first character index
func GetNextBase31Char(c sessions.Context, char rune, isFirstChar bool) (rune, bool, error) {
	validChars := Base31Chars
	if isFirstChar {
		validChars = Base31CharsForFirstChar
	}
	index := strings.Index(validChars, strings.ToLower(string(char)))
	if index < 0 {
		return '0', false, fmt.Errorf("invalid character")
	}
	if index < len(validChars)-1 {
		nextChar := rune(validChars[index+1])
		return nextChar, false, nil
	} else {
		nextChar := rune(validChars[0])
		return nextChar, true, nil
	}
}

// Increment a base31 Id string
func IncrementBase31Id(c sessions.Context, previousID string) (string, error) {
	// Add 1 to the base36 value, skipping vowels
	// Start at the last character in the Id string, carrying the 1 as many times as necessary
	nextAvailableID := previousID
	index := len(nextAvailableID) - 1
	var newChar rune
	var err error
	processNextChar := true
	for processNextChar {
		// If we need to carry the 1 all the way to the beginning, then add a 1 at the beginning of the string
		if index < 0 {
			nextAvailableID = "1" + nextAvailableID
			processNextChar = false
		} else {
			// Increment the character at the current index in the Id string
			newChar, processNextChar, err = GetNextBase31Char(c, rune(nextAvailableID[index]), index == 0)
			if err != nil {
				return "", fmt.Errorf("Error processing id: %v", err)
			}
			nextAvailableID = replaceAtIndex(nextAvailableID, newChar, index)
			index = index - 1
		}
	}

	return nextAvailableID, nil
}

// Get the next available base36 Id string that doesn't contain vowels
func GetNextAvailableID(tx *database.Tx) (string, error) {
	// Query for the highest used pageId or userId
	var highestUsedID string
	row := database.NewQuery(`
		SELECT MAX(pageId)
		FROM (
			SELECT pageId
			FROM`).AddPart(PageInfosTableAll(nil)).Add(`AS pi
			UNION
			SELECT id
			FROM users
		) AS combined
		WHERE char_length(pageId) = (
			SELECT MAX(char_length(pageId))
			FROM (
				SELECT pageId
				FROM`).AddPart(PageInfosTableAll(nil)).Add(`AS pi
				UNION
				SELECT id
				FROM users
			) AS combined2
    )
		`).ToTxStatement(tx).QueryRow()
	_, err := row.Scan(&highestUsedID)
	if err != nil {
		return "", fmt.Errorf("Couldn't load id: %v", err)
	}
	return IncrementBase31Id(tx.DB.C, highestUsedID)
}

// Return whether the external url is already used, and the pageID of the original entry.
func IsDuplicateExternalUrl(db *database.DB, u *CurrentUser, externalUrl string) (bool, string, error) {
	var pageID, domainID string
	row := database.NewQuery(`
		SELECT pi.pageId, pi.seeDomainId
		FROM`).AddPart(PageInfosTable(nil)).Add(`AS pi
		WHERE pi.externalUrl = ?`, externalUrl).ToStatement(db).QueryRow()
	exists, err := row.Scan(&pageID, &domainID)

	if err != nil {
		return false, "", err
	}
	if !exists {
		return false, "", nil
	}

	if !CanUserSeeDomain(u, domainID) {
		return true, "", nil
	}
	return true, pageID, nil
}
