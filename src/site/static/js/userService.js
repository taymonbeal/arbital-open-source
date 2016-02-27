"use strict";

// User service.
app.service("userService", function($http, $location, $rootScope){
	var that = this;

	// Logged in user.
	this.user = undefined;

	// Map of all user objects.
	this.userMap = {};

	// Check if we can let this user do stuff.
	this.userIsCool = function() {
		return this.user && this.user.karma >= 200;
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
			this.userMap = {};
		}

		if (data.user) {
			this.user = data.user;
		}
		$.extend(that.userMap, data["users"]);
	};

	this.isTouchDevice = "ontouchstart" in window // works in most browsers
		|| (navigator.MaxTouchPoints > 0)
		|| (navigator.msMaxTouchPoints > 0);

	// Sign into FB and call the callback with the response.
	this.fbLogin = function(callback) {
		// Apparently FB.login is not supported in Chrome in iOS
		if (navigator.userAgent.match("CriOS")) {
			var appId = isLive() ? "1064531780272247" : "1064555696936522";
			var redirectUrl = encodeURIComponent($location.absUrl());
			window.location.href = "https://www.facebook.com/dialog/oauth?client_id=" + appId +
					"&redirect_uri=" + redirectUrl + "&scope=email,public_profile";
		} else {
			window.FB.login(function(response){
				callback(response);
			}, {scope: 'public_profile,email'});
		}
	};

	// Check if FB redirected back to use with the code
	if ($location.search().code) {
		var data = {
			fbCodeToken: $location.search().code,
		};
		$location.search("code", undefined);
		// FB inserts this hash if there was no hash. It's a security thing. We need
		// to remove it to get our original redirectUrl.
		if ($location.hash() == "_=_") {
			$location.hash("");
		}
		data.fbRedirectUrl = $location.absUrl();
		$http({method: "POST", url: "/signup/", data: JSON.stringify(data)})
		.success(function(data, status){
			window.location.reload();
		})
		.error(function(data, status){
			console.error("Error FB signup:"); console.log(data); console.log(status);
		});
	}
});
