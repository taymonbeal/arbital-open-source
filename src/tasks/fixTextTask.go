// fixTextTask.go updates all pages' text fields to fix common mistakes
package tasks

import (
	"fmt"
	"regexp"
	"strings"

	"zanaduu3/src/database"
)

// FixTextTask is the object that's put into the daemon queue.
type FixTextTask struct {
}

func (task FixTextTask) Tag() string {
	return "fixTextTask"
}

// Check if this task is valid, and we can safely execute it.
func (task FixTextTask) IsValid() error {
	return nil
}

// Execute this task. Called by the actual daemon worker, don't call on BE.
// For comments on return value see tasks.QueueTask
func (task FixTextTask) Execute(db *database.DB) (delay int, err error) {
	c := db.C

	if err = task.IsValid(); err != nil {
		return 0, err
	}

	c.Infof("==== FIX TEXT START ====")
	defer c.Infof("==== FIX TEXT COMPLETED ====")

	rows := db.NewStatement(`
			SELECT pageId,edit,text
			FROM pages
			WHERE isLiveEdit`).Query()
	if err = rows.Process(fixText3); err != nil {
		c.Infof("ERROR, failed to fix text: %v", err)
		return 0, err
	}

	return 0, err
}

func fixText3(db *database.DB, rows *database.Rows) error {
	var pageID, edit string
	var text string
	if err := rows.Scan(&pageID, &edit, &text); err != nil {
		return fmt.Errorf("failed to scan a page: %v", err)
	}

	// Find and replace "Click [here to edit](http://arbital.com/edit/base10id"; links with "Click [here to edit](http://arbital.com/edit/base36id";

	// First remove all instances of "http://zanaduu3.appspot.com/pages/" in the links, leaving just the pageId
	// On the first pass, accept anything inside the parentheses, since the text we want to remove isn't a valid alias
	exp := regexp.MustCompile("Click \\[here to edit\\]\\(http\\:\\/\\/arbital\\.com\\/edit\\/([0-9]+)")

	submatches := exp.FindAllStringSubmatch(text, -1)
	base10Id := "0"
	base36Id := "0"
	for _, submatch := range submatches {
		db.C.Infof("submatch: %v", submatch)
		db.C.Infof("submatch[0]: %v", submatch[0])
		db.C.Infof("submatch[1]: %v", submatch[1])
		//base10Id = submatch[1]

		rows = database.NewQuery(`
				SELECT base36id,base10id
				FROM base10tobase36
				WHERE base10id = `).AddArg(submatch[1]).ToStatement(db).Query()

		err := rows.Process(func(db *database.DB, rows *database.Rows) error {
			err := rows.Scan(&base36Id, &base10Id)
			if err != nil {
				return fmt.Errorf("failed to scan: %v", err)
			}
			db.C.Infof("base10Id: %v", base10Id)
			db.C.Infof("base36Id: %v", base36Id)
			return nil
		})
		if err != nil {
			return err
		}
	}

	newText := exp.ReplaceAllStringFunc(text, func(submatch string) string {
		//exp.ReplaceAllStringFunc(text, func(submatch string) string {

		result := submatch
		result = strings.Replace(result, base10Id, base36Id, -1)
		db.C.Infof("submatch: %v", submatch)
		db.C.Infof("result  : %v", result)

		return result
	})

	//db.C.Infof("newText: %v", newText)

	if newText != text {
		db.C.Infof("========================== %s", text)
		db.C.Infof("========================== %s", newText)
		hashmap := make(map[string]interface{})
		hashmap["pageId"] = pageID
		hashmap["edit"] = edit
		hashmap["text"] = newText
		statement := db.NewInsertStatement("pages", hashmap, "text")
		if _, err := statement.Exec(); err != nil {
			return fmt.Errorf("Couldn't update pages table: %v", err)
		}
	}

	exp = regexp.MustCompile("Click \\[here to edit\\]\\(http\\:\\/\\/arbital\\.com\\/edit\\/" + pageID)

	submatches = exp.FindAllStringSubmatch(newText, -1)

	for _, submatch := range submatches {
		db.C.Infof("correct submatch: %v", submatch)
	}

	return nil
}
