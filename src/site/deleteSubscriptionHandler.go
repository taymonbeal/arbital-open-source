// deleteSubscription.go handles requests for deleting a subscription.
package site

import (
	"encoding/json"
	"fmt"
	"net/http"

	"zanaduu3/src/database"
	"zanaduu3/src/sessions"
	"zanaduu3/src/user"
)

// deleteSubscriptionData is the object that's put into the daemon queue.
type deleteSubscriptionData struct {
	QuestionId int64 `json:",string"`
	CommentId  int64 `json:",string"`
}

// deleteSubscriptionHandler handles requests for deleting a subscription.
func deleteSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	c := sessions.NewContext(r)

	decoder := json.NewDecoder(r.Body)
	var data deleteSubscriptionData
	err := decoder.Decode(&data)
	if err != nil {
		c.Inc("delete_subscription_fail")
		c.Errorf("Couldn't decode json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get user object
	var u *user.User
	u, err = user.LoadUser(w, r)
	if err != nil {
		c.Inc("delete_subscription_fail")
		c.Errorf("Couldn't load user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := fmt.Sprintf(`
		DELETE FROM subscriptions
		WHERE userId=%d AND `, u.Id)
	if data.QuestionId > 0 {
		query += fmt.Sprintf("questionId=%d", data.QuestionId)
	} else if data.CommentId > 0 {
		query += fmt.Sprintf("commentId=%d", data.CommentId)
	} else {
		c.Inc("delete_subscription_fail")
		c.Errorf("At least one id has to be given: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err = database.ExecuteSql(c, query); err != nil {
		c.Inc("delete_subscription_fail")
		c.Errorf("Couldn't delete a subscription: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}