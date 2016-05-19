// Handles requests to update subscriptions
package site

import (
	"encoding/json"
	"net/http"

	"zanaduu3/src/pages"
)

type updateSubscriptionData struct {
	ToId         string `json:"toId"`
	IsSubscribed bool   `json:"isSubscribed"`
	AsMaintainer bool   `json:asMaintainer"`
}

var updateSubscriptionHandler = siteHandler{
	URI:         "/updateSubscription/",
	HandlerFunc: updateSubscriptionHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

func updateSubscriptionHandlerFunc(params *pages.HandlerParams) *pages.Result {
	db := params.DB
	u := params.U

	// Decode data
	var data updateSubscriptionData
	err := json.NewDecoder(params.R.Body).Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode request", err).Status(http.StatusBadRequest)
	}

	// If not subscribed anymore, delete the subscription
	if !data.IsSubscribed {
		statement := db.NewStatement(`
			DELETE FROM subscriptions
			WHERE userId=? AND toId=?`)
		if _, err := statement.Exec(u.Id, data.ToId); err != nil {
			return pages.Fail("Couldn't delete a subscription", err)
		}
		return pages.Success(nil)
	}

	statement := db.NewStatement(`
			UPDATE subscriptions
			SET asMaintainer=?
			WHERE userId=? AND toId=?`)
	if _, err := statement.Exec(data.AsMaintainer, u.Id, data.ToId); err != nil {
		return pages.Fail("Couldn't update a subscription", err)
	}

	return pages.Success(nil)
}
