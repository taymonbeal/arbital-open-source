/* An entry for every user that has ever done anything in our system. */
CREATE TABLE users (
	/* PK. User's unique id. */
	id BIGINT NOT NULL AUTO_INCREMENT,
	/* Date this user was added to the table. */
	createdAt DATETIME NOT NULL,
	/* User's email. */
	email VARCHAR(255) NOT NULL,
	/* User's self-assigned first name. */
	firstName VARCHAR(32) NOT NULL,
	/* User's self-assigned last name. */
	lastName VARCHAR(32) NOT NULL,
	/* If the user added FB, this is their userId */
	fbUserId VARCHAR(32) NOT NULL,
	/* Date of the last website visit. */
	lastWebsiteVisit DATETIME NOT NULL,
	/* True iff the user is an admin. */
	isAdmin BOOLEAN NOT NULL,
	/* Amount of karam this user has. */
	karma INT NOT NULL,
	/* Invite code the user used to join the website, if any. */
	inviteCode VARCHAR(16) NOT NULL,
	/* Date of the last updates email. */
	updateEmailSentAt DATETIME NOT NULL,
	/* How frequently to send update emails. */
	emailFrequency VARCHAR(16) NOT NULL,
	/* How many updates before sending an update email. */
	emailThreshold INT(11) NOT NULL,
	/* If true, don't do a live preview of MathJax. */
	ignoreMathjax BOOL NOT NULL,
	UNIQUE (email),
	PRIMARY KEY(id)
) CHARACTER SET utf8 COLLATE utf8_general_ci;
