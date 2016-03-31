/* An update is a notification for the user that something new has happened, e.g.
there was a new comment. Updates are created only when a user is subscribed to
something, usually a page.

If there are multiple new comments on (or edits to) a page, we don't want to count
each one as a new update for the count that the user sees in the navigation bar
(on the bell icon). That's what the "newCount" variable is for -- after the first
update of a particular type, for a particular page, we set subsequent similar
updates to have newCount of "0". What the user sees is the sum of all newCounts.

When the user visits the update pages, all the counts are zeroed out, since
the user has been made aware of all the updates.
*/
CREATE TABLE updates (
	/* Unique update id. PK. */
	id BIGINT NOT NULL AUTO_INCREMENT,

	/* The update is for this user. FK into users. */
	userId VARCHAR(32) NOT NULL,

	/* The update was generated by this user. FK into users. */
	byUserId VARCHAR(32) NOT NULL,

	/* Type of update */
	type VARCHAR(32) NOT NULL,

	/* When this update was created. */
	createdAt DATETIME NOT NULL,

	/* Amount this update contributes to the "number of new updates user has".
		Usually 1 or 0. */
	newCount INT NOT NULL,

	/* True if this update has been emailed out. */
	emailed BOOLEAN NOT NULL,

	/* One of these has to be set. Updates will be grouped by this key and show up
		in the same panel. */
	groupByPageId VARCHAR(32) NOT NULL,
	groupByUserId VARCHAR(32) NOT NULL,

	/* User got this update because they are subscribed to "this thing". FK into
		pages. */
	subscribedToId VARCHAR(32) NOT NULL,

	/* User will be directed to "this thing" for more information about the update. */
	goToPageId VARCHAR(32) NOT NULL,


	/* Only set if type is 'pageInfoEdit'. Used to show what changed on the updates page.
		FK into changeLogs. */
	changeLogId VARCHAR(32) NOT NULL,

	PRIMARY KEY(id)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
