/* This table contains all the subscriptions. */
CREATE TABLE subscriptions (
	/* User id of the subscriber. FK into users. */
	userId VARCHAR(32) NOT NULL,
	/* Id of the thing (user, page, etc...) the user is subscribed to. FK into pages. */
	toId VARCHAR(32) NOT NULL,
	/* When this subscription was created. */
	createdAt DATETIME NOT NULL,
	/* User's trust when they subscribed. FK into userTrustSnapshots */
	trustSnapshotId BIGINT NOT NULL,
  	PRIMARY KEY(userId, toId)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
