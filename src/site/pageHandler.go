// pageHandler.go has the functions and wrappers for handling pages
package site

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"appengine/taskqueue"

	"zanaduu3/src/core"
	"zanaduu3/src/database"
	"zanaduu3/src/pages"
	"zanaduu3/src/sessions"

	"github.com/gorilla/mux"
)

// Handler serves HTTP.
type handler http.HandlerFunc

// commonPageData contains data that is common between all pages.
type commonPageData struct {
	// Logged in user
	User *core.CurrentUser
	// Map of page id -> currently live version of the page
	PageMap map[string]*core.Page
	// Map of page id -> some edit of the page
	EditMap    map[string]*core.Page
	UserMap    map[string]*core.User
	MasteryMap map[string]*core.Mastery
}

// pageHandlerWrapper wraps one of our page handlers.
func pageHandlerWrapper(p *pages.Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If live, check that this is an HTTPS request
		if sessions.Live && r.URL.Scheme != "https" {
			safeUrl := strings.Replace(r.URL.String(), "http", "https", 1)
			http.Redirect(w, r, safeUrl, http.StatusSeeOther)
		}

		c := sessions.NewContext(r)
		params := pages.HandlerParams{W: w, R: r, C: c}
		rand.Seed(time.Now().UnixNano())

		// Redirect www.
		if mux.Vars(r)["www"] != "" {
			if sessions.Live {
				newUrl := strings.Replace(r.URL.String(), "www.", "", -1)
				c.Debugf("Redirecting '%s' to '%s' because of 'www'", r.URL.String(), newUrl)
				http.Redirect(w, r, newUrl, http.StatusSeeOther)
			} else {
				subdomainStr := ""
				if mux.Vars(r)["subdomain"] != "" {
					subdomainStr = mux.Vars(r)["subdomain"] + "."
				}
				url := fmt.Sprintf("http://%s%s%s", subdomainStr, sessions.GetRawDomain(), r.URL.String())
				http.Redirect(w, r, url, http.StatusSeeOther)
			}
			return
		}

		// Helper func to when an error occurs and we should render error page.
		fail := func(responseCode int, message string, err error) {
			c.Inc(fmt.Sprintf("%s-fail", r.URL.Path))
			c.Errorf("%s: %v", message, err)
			w.WriteHeader(responseCode)
			fmt.Fprintf(w, "Error rendering the page: %s", message)
		}

		// Recover from panic.
		if sessions.Live {
			defer func() {
				if r := recover(); r != nil {
					c.Errorf("%v", r)
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "%s", "Super serious error has occurred. Super. Serious. Error.")
				}
			}()
		}

		// Open DB connection
		db, err := database.GetDB(c)
		if err != nil {
			fail(http.StatusInternalServerError, "Couldn't open DB", err)
			return
		}
		params.DB = db

		// Get user object
		u, err := core.LoadCurrentUser(w, r, db)
		if err != nil {
			fail(http.StatusInternalServerError, "Couldn't load user", err)
			return
		}
		params.U = u

		// Get subdomain info
		params.PrivateGroupId, err = loadSubdomain(r, db, u)
		if err != nil {
			fail(http.StatusInternalServerError, "Couldn't load subdomain", err)
			return
		}

		// When in a subdomain, we always have to be logged in
		if core.IsIdValid(params.PrivateGroupId) && !core.IsIdValid(u.Id) {
			if r.URL.Path != "/login/" {
				http.Redirect(w, r, fmt.Sprintf("/login/?continueUrl=%s", url.QueryEscape(r.URL.String())), http.StatusSeeOther)
			}
		}
		if userId := u.GetSomeId(); userId != "" {
			statement := db.NewStatement(`
						UPDATE users
						SET lastWebsiteVisit=?
						WHERE id=?`)
			if _, err := statement.Exec(database.Now(), userId); err != nil {
				fail(http.StatusInternalServerError, "Couldn't update users", err)
				return
			}
		}
		// Check if we have access to the private group
		if core.IsIdValid(params.PrivateGroupId) &&
			!u.IsMemberOfGroup(params.PrivateGroupId) &&
			r.URL.Path != "/login/" {
			fail(http.StatusForbidden, "Don't have access to this group", nil)
			return
		}

		// Call the page's renderer
		result := p.Render(&params)
		if result.ResponseCode != http.StatusOK && result.ResponseCode != http.StatusSeeOther {
			c.Errorf("Primary renderer failed")
			fail(result.ResponseCode, result.Message, result.Err)
			return
		}
		if result.Data == nil {
			result.Data = map[string]string{
				"Title":       "Arbital",
				"Url":         "https://" + r.Host + r.RequestURI,
				"Description": "",
			}
		}

		// Load updates count. (Loading it afterwards since it could be affected by the page)
		u.UpdateCount, err = core.LoadUpdateCount(db, u.Id)
		if err != nil {
			fail(http.StatusInternalServerError, "Couldn't retrieve updates count", err)
			return
		}

		p.ServeHTTP(w, r, result)
		c.Inc(fmt.Sprintf("%s-success", r.URL.Path))
	}
}

// newPage returns a new page using default options.
func newPage(renderer pages.Renderer, tmpls []string) pages.Page {
	return pages.Add("", renderer, pages.PageOptions{}, tmpls...)
}

// monitor sends counters added within the handler off to monitoring.
func (fn handler) monitor() handler {
	return func(w http.ResponseWriter, r *http.Request) {
		c := sessions.NewContext(r)
		fn(w, r)
		// At end of each request, add task to reporting queue.
		t, err := c.Report()
		if err != nil {
			c.Errorf("failed to create monitoring task: %v\n", err)
			return
		}
		if t == nil {
			// no monitoring task, nothing to do.
			return
		}
		_, err = taskqueue.Add(c, t, "report-monitoring")
		if err != nil {
			c.Errorf("failed to add monitoring POST task to queue: %v\n", err)
			return
		}
	}
}
