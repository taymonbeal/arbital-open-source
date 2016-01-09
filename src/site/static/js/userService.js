"use strict";

// User service.
app.service("userService", function(){
	var that = this;

	// Logged in user.
	this.user = undefined;

	// Map of all user objects.
	this.userMap = {};

	// Check if we can let this user do stuff.
	this.userIsCool = function() {
		return this.user && this.user.karma >= 200;
	};

	// Return url to the user page.
	this.getUserUrl = function(userId) {
		return "/user/" + userId;
	};

	// Return a user's full name.
	this.getFullName = function(userId) {
		var user = this.userMap[userId];
		if (!user) console.error("User not found: " + userId);
		return user.firstName + " " + user.lastName;
	};

	// Call this to process data we received from the server.
	this.processServerData = function(data) {
		if (data.resetEverything) {
			that.userMap = {};
		}
		if (data.user) {
			that.user = data.user;
		}
		$.extend(that.userMap, data["users"]);
	}
});
