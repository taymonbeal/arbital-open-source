// updateSettingsHandler.go updates the settings from the Settings page
package site

import (
	"encoding/json"

	"zanaduu3/src/pages"
)

// updateSettingsData contains data given to us in the request.
type updateSettingsData struct {
	EmailFrequency string
	EmailThreshold string
}

var updateSettingsHandler = siteHandler{
	URI:         "/updateSettings/",
	HandlerFunc: updateSettingsHandlerFunc,
	Options: pages.PageOptions{
		RequireLogin: true,
	},
}

// updateSettingsHandlerFunc handles submitting the settings from the Settings page
func updateSettingsHandlerFunc(params *pages.HandlerParams) *pages.Result {
	db := params.DB
	u := params.U

	var data updateSettingsData
	decoder := json.NewDecoder(params.R.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return pages.HandlerBadRequestFail("Couldn't decode json", err)
	}
	if data.EmailFrequency == "" || data.EmailThreshold == "" {
		return pages.HandlerBadRequestFail("Email Frequency and Email Threshold have to be set", nil)
	}
	if data.EmailThreshold <= "0" {
		return pages.HandlerBadRequestFail("Email Threshold has to be greater than 0", nil)
	}

	statement := db.NewStatement(`
		UPDATE users
		SET emailFrequency=?,emailThreshold=?
		WHERE id=?`)
	_, err = statement.Exec(data.EmailFrequency, data.EmailThreshold, u.Id)
	if err != nil {
		return pages.HandlerErrorFail("Couldn't update settings", err)
	}

	return pages.StatusOK(nil)
}
