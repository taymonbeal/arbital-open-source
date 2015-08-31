/* An update is a notification for the user that something new has happened, e.g.
there was a new comment. Updates are created only when a user is subscribed to
something, usually a page.
  Since there could be multiple new comments on a page, we don't want to count
each one as a new update. That's what "count" variable is for. This way we can
set subsequent new comments to have count of "0". Sum of all counts is what the
user sees in the navigation bar.
  When the user visits the update pages, all the counts are zeroed out, since
the user has been made aware of all the updates.
*/
CREATE TABLE updates (
	/* Unique update id. PK. */
  id BIGINT NOT NULL AUTO_INCREMENT,
	/* The update is for this user. FK into users. */
  userId BIGINT NOT NULL,
	/* Type of update */
	type VARCHAR(32) NOT NULL,
	/* When this update was created. */
  createdAt DATETIME NOT NULL,
	/* Amount this update contributes to the "number of new updates user has".
		Usually 1 or 0. */
	newCount INT NOT NULL,

	/* One of these has to be set. Updates will be grouped by this key and show up
		in the same panel. */
	groupByPageId BIGINT NOT NULL,
	groupByUserId BIGINT NOT NULL,

	/* One of these has to be set. User got this update because they are subscribed
		to "this thing".
		FK into pages. */
  subscribedToPageId BIGINT NOT NULL,
	/* FK into users. */
	subscribedToUserId BIGINT NOT NULL,

	/* One of these has to be set. User will be directed to "this thing" for more
		information about the update. */
	goToPageId BIGINT NOT NULL,

  PRIMARY KEY(id)
) CHARACTER SET utf8 COLLATE utf8_general_ci;
