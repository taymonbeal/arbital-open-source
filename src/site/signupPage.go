// signupPage.go serves the signup page.
package site

import (
	"fmt"
	"net/http"
	"strings"

	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"
	"zanaduu3/src/user"
)

// signupData stores the data that we pass to the signup.tmpl to render the page
type signupData struct {
	User        *user.User
	ContinueUri string
}

// signupPage serves the questions page.
var signupPage = pages.Add(
	"/signup/",
	signupRenderer,
	append(baseTmpls,
		"tmpl/signup.tmpl", "tmpl/navbar.tmpl")...)

// signupRenderer renders the signup page.
func signupRenderer(w http.ResponseWriter, r *http.Request) *pages.Result {
	c := sessions.NewContext(r)

	var data signupData
	var err error
	data.User, err = user.LoadUser(w, r)
	if err != nil {
		c.Inc("update_prior_vote_fail")
		c.Errorf("Couldn't load user: %v", err)
		return pages.InternalErrorWith(fmt.Errorf("Couldn't load user."))
	}

	if data.User.Id <= 0 {
		return pages.RedirectWith(data.User.LoginLink)
	}

	q := r.URL.Query()
	if q.Get("inviteCode") != "" {
		// This is a form submission.
		if strings.ToLower(q.Get("inviteCode")) != "truth" {
			return pages.InternalErrorWith(fmt.Errorf("Invalid invite code"))
		}
		hashmap := make(map[string]interface{})
		hashmap["id"] = data.User.Id
		hashmap["firstName"] = q.Get("firstName")
		hashmap["lastName"] = q.Get("lastName")
		sql := database.GetInsertSql("users", hashmap, "firstName", "lastName")
		if _, err = database.ExecuteSql(c, sql); err != nil {
			c.Errorf("Couldn't update user's name: %v", err)
			return pages.InternalErrorWith(fmt.Errorf("Couldn't update user's name"))
		}
		data.User.FirstName = q.Get("firstName")
		data.User.LastName = q.Get("lastName")
		data.User.IsLoggedIn = true
		err = data.User.Save(w, r)
		if err != nil {
			c.Errorf("Couldn't re-save the user after adding the name: %v", err)
		}
		return pages.RedirectWith(q.Get("continueUri"))
	}
	data.ContinueUri = q.Get("continueUri")
	c.Inc("signup_page_served_success")
	return pages.StatusOK(data)
}
