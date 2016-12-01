// externalUrlHandler.go gets info about an external url

package site

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"golang.org/x/net/html"
	"google.golang.org/appengine/urlfetch"

	"zanaduu3/src/core"
	"zanaduu3/src/pages"
)

var externalUrlHandler = siteHandler{
	URI:         "/getExternalUrlData/",
	HandlerFunc: externalUrlHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

// externalUrlData contains parameters passed in via the request.
type externalUrlData struct {
	RawExternalUrlString string
}

// externalUrlHandlerFunc handles the request.
func externalUrlHandlerFunc(params *pages.HandlerParams) *pages.Result {
	db := params.DB
	u := params.U
	returnData := core.NewHandlerData(u)

	// Decode data
	var data externalUrlData
	err := json.NewDecoder(params.R.Body).Decode(&data)
	if err != nil {
		return pages.Fail("Couldn't decode request", err).Status(http.StatusBadRequest)
	}

	externalUrl, err := url.Parse(data.RawExternalUrlString)
	if err != nil {
		// If not a valid url, just return.
		return pages.Success(returnData)
	}
	externalUrlString := externalUrl.String()

	isDupe, originalPageID, err := core.IsDuplicateExternalUrl(db, u, externalUrlString)
	if err != nil {
		return pages.Fail("Couldn't check if external url is already in use.", err)
	}

	if isDupe {
		returnData.ResultMap["isDupe"] = isDupe
		returnData.ResultMap["originalPageID"] = originalPageID

		// Load data
		core.AddPageToMap(originalPageID, returnData.PageMap, core.TitlePlusLoadOptions)
		err = core.ExecuteLoadPipeline(db, returnData)
		if err != nil {
			return pages.Fail("Pipeline error", err)
		}
	} else {
		resp, err := urlfetch.Client(db.C).Get(externalUrlString)
		if err != nil {
			// If can't find the page, just return.
			return pages.Success(returnData)
		}

		defer resp.Body.Close()
		htmlBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return pages.Fail("Couldn't read request response.", err)
		}

		title, err := getTitle(externalUrlString, string(htmlBytes))
		if err != nil {
			return pages.Fail("Couldn't get title from html.", err)
		}
		returnData.ResultMap["title"] = title
	}

	return pages.Success(returnData)
}

func getTitle(url string, htmlString string) (string, error) {
	og := opengraph.NewOpenGraph()
	err := og.ProcessHTML(strings.NewReader(htmlString))
	if err != nil {
		return "", err
	}

	title := og.Title
	if len(title) == 0 {
		title, err = getTitleFromMetaTag(htmlString)
		if err != nil {
			return "", err
		}
	}

	title = strings.TrimSpace(title)

	// special case to strip endings from the titles of links to LessWrong
	lowercaseUrl := strings.ToLower(url)
	if strings.HasPrefix(lowercaseUrl, "https://lesswrong.com") || strings.HasPrefix(lowercaseUrl, "http://lesswrong.com") {
		title = strings.TrimSuffix(title, " - Less Wrong")
	}

	return title, nil
}

func getTitleFromMetaTag(htmlString string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return "", err
	}

	for queue := []*html.Node{doc}; len(queue) > 0; queue = queue[1:] {
		n := queue[0]
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "name":
					name = a.Val
				case "content":
					content = a.Val
				}
			}
			if name == "title" {
				return content, nil
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			queue = append(queue, c)
		}
	}

	return "", nil
}
