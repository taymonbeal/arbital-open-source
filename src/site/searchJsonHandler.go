// searchJsonHandler.go contains the handler for searching all the pages.

package site

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"zanaduu3/src/core"
	"zanaduu3/src/elastic"
	"zanaduu3/src/pages"
)

const (
	// How many results to get from Elastic
	searchSize = 20
	// How many results to return to the FE
	returnSearchSize = 10
	// Minimum allowed score for results
	minSearchScore = 0.0
)

type searchJSONData struct {
	Term string
	// If this is set, only pages of this type will be returned
	PageType string
	// If this is set, pages of these types will not be returned
	FilterPageTypes []string
	// If this is set, pages in this domain will get a boost
	PreferredEditDomainID string
}

var searchHandler = siteHandler{
	URI:         "/json/search/",
	HandlerFunc: searchJSONHandler,
}

// searchJsonHandler handles the request.
func searchJSONHandler(params *pages.HandlerParams) *pages.Result {
	u := params.U

	// Decode data
	var data searchJSONData
	decoder := json.NewDecoder(params.R.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return pages.Fail("Error decoding JSON", err)
	}
	if data.Term == "" {
		return pages.Fail("No search term specified", nil).Status(http.StatusBadRequest)
	}

	var domainIDs []string
	for domainID := range u.DomainMembershipMap {
		if core.CanUserSeeDomain(u, domainID) {
			domainIDs = append(domainIDs, "\""+domainID+"\"")
		}
	}
	domainIDs = append(domainIDs, "\"0\"")

	data.Term = elastic.EscapeMatchTerm(data.Term)

	// Filter by type of the page
	mandatoryTypeFilter := ""
	if data.PageType != "" {
		mandatoryTypeFilter = fmt.Sprintf(`{"term": { "type": "%s" } },`, elastic.EscapeMatchTerm(data.PageType))
	}
	forbiddenTypeFilter := ""
	if types := data.FilterPageTypes; len(types) != 0 {
		for i, s := range types {
			types[i] = elastic.EscapeMatchTerm(s)
		}
		b, err := json.Marshal(types)
		if err == nil {
			forbiddenTypeFilter = fmt.Sprintf(`{"terms": { "type": %s } },`, b)
		} else {
			return pages.Fail("Error constructing ElasticSearch query: %v", err)
		}
	}

	// To allow for partial matching on the last word, we have to split it off
	lastSpaceIndex := strings.LastIndex(data.Term, " ")
	textMatch := fmt.Sprintf(`
		{
			"match_phrase_prefix": {
				"title": {
					"query": "%[1]s",
					"boost": 3
				}
			}
		},
		{
			"match_phrase_prefix": { "clickbait": "%[1]s" }
		},
		{
			"match_phrase_prefix": { "text": "%[1]s" }
		},`, data.Term)
	if lastSpaceIndex > 0 {
		matchTerm := data.Term[:lastSpaceIndex]
		prefixTerm := data.Term[lastSpaceIndex+1:]
		textMatch = fmt.Sprintf(`
		{
			"match": {
				"title": {
					"query": "%[1]s",
					"boost": 3
				}
			}
		},
		{
			"match_phrase_prefix": {
				"title": {
					"query": "%[2]s",
					"boost": 3
				}
			}
		},
		{
			"match": { "clickbait": "%[1]s" }
		},
		{
			"match_phrase_prefix": { "clickbait": "%[2]s" }
		},
		{
			"match": { "text": "%[1]s" }
		},
		{
			"match_phrase_prefix": { "text": "%[2]s" }
		},
		`, matchTerm, prefixTerm)
	}

	// Construct the search JSON
	jsonStr := fmt.Sprintf(`{
		"min_score": %[1]v,
		"size": %[2]d,
		"query": {
			"filtered": {
				"query": {
					"bool": {
						"should": [
							{
								"term": { "pageId": "%[3]s" }
							},
							{
								"term": {
									"alias": {
										"value": "%[3]s",
										"boost": 3
									}
								}
							},
							{
								"term": {
									"editDomainId": {
										"value": "%[5]s",
										"boost": 2
									}
								}
							},`+textMatch+`
							{
								"match_phrase_prefix": { "alias": "%[3]s" }
							},
							{
								"match_phrase_prefix": { "externalUrl": "%[3]s" }
							}
						]
					}
				},
				"filter": {
					"bool": {
						"must_not": [`+forbiddenTypeFilter+`
							{
								"terms": { "type": ["comment"] }
							}
						],
						"must": [`+mandatoryTypeFilter+`
							{
								"terms": { "seeDomainId": [%[4]s] }
							}
						]
					}
				}
			}
		},
		"_source": []
	}`, minSearchScore, searchSize, data.Term, strings.Join(domainIDs, ","), data.PreferredEditDomainID)
	return searchJSONInternalHandler(params, jsonStr)
}

func searchJSONInternalHandler(params *pages.HandlerParams, query string) *pages.Result {
	db := params.DB
	u := params.U
	returnData := core.NewHandlerData(u)

	// Perform search.
	results, err := elastic.SearchPageIndex(params.C, query)
	if err != nil {
		return pages.Fail("Error with elastic search", err)
	}

	loadOptions := (&core.PageLoadOptions{
		Tags:     true,
		Creators: u.ID != "",
	}).Add(core.TitlePlusLoadOptions)

	// Create page map.
	for _, hit := range results.Hits.Hits {
		core.AddPageToMap(hit.ID, returnData.PageMap, loadOptions)
	}

	// Load pages.
	err = core.ExecuteLoadPipeline(db, returnData)
	if err != nil {
		return pages.Fail("error while loading pages", err)
	}

	// Adjust results' scores: tag id -> multiplier factor
	tagMultiplierMap := map[string]float32{
		"22t": 0.75, // Just a requisite
		"15r": 0.85, // Out of date
		"4v":  0.75, // Work in progress
		"72":  0.65, // Stub
		"6cc": 2.0,  // Concept
	}
	for _, hit := range results.Hits.Hits {
		if page, ok := returnData.PageMap[hit.Source.PageID]; ok {
			// Adjust the score based on tags
			for _, tagID := range page.TagIDs {
				if penalty, ok := tagMultiplierMap[tagID]; ok {
					hit.Score *= penalty
				}
			}
			// Adjust the score based on likes
			if page.LikeScore > 0 {
				hit.Score *= float32(math.Log(float64(page.LikeScore))/10) + 1
			}
			if page.MyLikeValue > 0 {
				hit.Score *= 1.2
			}
			// Adjust the score if the user created the page
			if u.ID != "" {
				for _, creatorID := range page.CreatorIDs {
					if creatorID == u.ID {
						hit.Score *= 1.2
						break
					}
				}
			}
			// Adjust the score if it's a user page
			if page.Type == core.GroupPageType {
				hit.Score *= 0.2
			}
		} else {
			hit.Score = 0
		}
	}

	sort.Sort(results.Hits.Hits)
	if returnSearchSize < len(results.Hits.Hits) {
		results.Hits.Hits = results.Hits.Hits[0:returnSearchSize]
	} else {
		results.Hits.Hits = results.Hits.Hits[0:len(results.Hits.Hits)]
	}
	returnData.ResultMap["search"] = results.Hits
	return pages.Success(returnData)
}
