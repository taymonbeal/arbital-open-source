/* This table contains an entry for every change that a page undergoes. */
CREATE TABLE changeLogs (
	/* Unique update id. PK. */
  id BIGINT NOT NULL AUTO_INCREMENT,
	/* The user who caused this event. FK into users. */
  userId BIGINT NOT NULL,
	/* The affected page. FK into pages. */
	pageId BIGINT NOT NULL,
	/* Edit number of the affected page. Partial FK into pages. */
	edit INT NOT NULL,
	/* Type of update */
	type VARCHAR(32) NOT NULL,
	/* When this update was created. */
  createdAt DATETIME NOT NULL,

	/* This is set for various events. E.g. if a new parent is added, this will
	be set to the parent id. */
	auxPageId BIGINT NOT NULL,

  PRIMARY KEY(id)
) CHARACTER SET utf8 COLLATE utf8_general_ci;