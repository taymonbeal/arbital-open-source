/* This file contains the recent changes to schemas, sorted from oldest to newest. */
CREATE TABLE domains (
	/* Domain id. */
	id BIGINT NOT NULL AUTO_INCREMENT,
	/* Id of the home page for this domain. FK into pageInfos. */
	pageId VARCHAR(32) NOT NULL,
	/* When this page was originally created. */
	createdAt DATETIME NOT NULL,
	/* Id of the user who created the page. FK into users. */
	createdBy VARCHAR(32) NOT NULL,
	/* Alias name of the domain. */
	alias VARCHAR(64) NOT NULL,

	/* ============ Various domain settings ============ */
	/* If true, any registered user can comment. */
	canUsersComment BOOL NOT NULL,
	/* If true, any registered user can propose an edit. */
	canUsersProposeEdits BOOL NOT NULL,

	PRIMARY KEY(id)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
insert into domains (pageId,createdAt,createdBy,alias,canUsersComment,canUsersProposeEdits) (select pageId,createdAt,createdBy,alias,true,true from pageInfos where type="domain");
update pageInfos set type="wiki" where type="domain";
