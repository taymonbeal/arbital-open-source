/* This table contains various information about the pages. This info is not
 dependent on any specific edit number. */
CREATE TABLE pageInfos (
	/* Id of the page the info is for. */
	pageId BIGINT NOT NULL,
	/* Edit number currently used to display the page. 0 if this page hasn't
		been published. */
	currentEdit INT NOT NULL,
	/* Maximum edit number used by this page. */
	maxEdit INT NOT NULL,
	/* When this page was originally created. */
	createdAt DATETIME NOT NULL,

	/* Alias name of the page. */
	alias VARCHAR(64) NOT NULL,
	/* Page's type. */
	type VARCHAR(32) NOT NULL,
	/* How to sort the page's children. */
	sortChildrenBy VARCHAR(32) NOT NULL,
	/* True iff the page has a probability vote. */
	hasVote BOOLEAN NOT NULL,
	/* Type of the vote this page has. If empty string, it has never been set.
	 But once voting is set, it can only be turned on/off, the type cannot be
	 changed. */
	voteType VARCHAR(32) NOT NULL,

	/* === Permission settings === */
	/* see: who can see the page */
	/* act: who can perform actions on the page (e.g. vote, comment) */
	/* edit: who can edit the page */
	/* If set, only this group can see the page. FK into pages. */
	seeGroupId BIGINT NOT NULL,
	/* If set, only this group can edit the page. FK into pages. */
	editGroupId BIGINT NOT NULL,
	/* Minimum amount of karma a user needs to edit this page. */
	editKarmaLock INT NOT NULL,

	/* If set, the page is locked by this user. FK into users. */
	lockedBy BIGINT NOT NULL,
	/* Time until the user has this lock. */
	lockedUntil DATETIME NOT NULL,
	PRIMARY KEY(pageId)
) CHARACTER SET utf8 COLLATE utf8_general_ci;
