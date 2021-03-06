// sendSlackInviteHandler.go sends an email to our admin to invite a user to slack

package site

import (
	"encoding/json"
	"fmt"
	"net/http"

	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"

	"zanaduu3/vendor/google.golang.org/appengine/mail"
)

// sendSlackInviteData contains data given to us in the request.
type sendSlackInviteData struct {
	Email string
}

var sendSlackInviteHandler = siteHandler{
	URI:         "/json/sendSlackInvite/",
	HandlerFunc: sendSlackInviteHandlerFunc,
	Options:     pages.PageOptions{},
}

// sendSlackInviteHandlerFunc handles requests to create/update a like.
func sendSlackInviteHandlerFunc(params *pages.HandlerParams) *pages.Result {
	db := params.DB
	u := params.U
	c := params.C

	var data sendSlackInviteData
	decoder := json.NewDecoder(params.R.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode json", err).Status(http.StatusBadRequest)
	}
	if data.Email == "" {
		return pages.Fail("Email isn't set", nil).Status(http.StatusBadRequest)
	}

	// Update user
	hashmap := make(map[string]interface{})
	hashmap["id"] = u.ID
	hashmap["isSlackMember"] = true
	statement := db.NewInsertStatement("users", hashmap, "isSlackMember")
	if _, err := statement.Exec(); err != nil {
		return pages.Fail("Couldn't update user", err)
	}

	if sessions.Live {
		// Create mail message
		msg := &mail.Message{
			Sender:  "alexei@arbital.com",
			To:      []string{"trigger@recipe.ifttt.com"},
			Subject: "#slackbot",
			Body:    fmt.Sprintf("%s (id: %s) wants to join Slack. Someone should invite them via: https://arbital.slack.com/admin", data.Email, u.ID),
		}

		err = mail.Send(c, msg)
		if err != nil {
			return pages.Fail("Couldn't send email: %v", err)
		}
	} else {
		// If not live, then do nothing, for now
	}

	return pages.Success(nil)
}
