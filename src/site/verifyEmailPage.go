// verifyEmailPage.go user is directed here when they click on a link to verify email

package site

import (
	"zanaduu3/src/pages"
	"zanaduu3/src/stormpath"
)

var verifyEmailPage = newPage(verifyEmailRenderer, dynamicTmpls)

func verifyEmailRenderer(params *pages.HandlerParams) *pages.Result {
	c := params.C

	sptoken := params.R.FormValue("sptoken")
	if sptoken == "" {
		return pages.Fail("Are you in the right place? (No sptoken found.)", nil)
	}

	err := stormpath.VerifyEmail(c, params.R.FormValue("sptoken"))
	if err != nil {
		return pages.Fail("Verification failed", err)
	}

	return pages.RedirectWith("/login")
}
