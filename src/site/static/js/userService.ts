'use strict';

import app from './angular.ts';
import {isLive, isIntIdValid} from './util.ts';

// User service.
app.service('userService', function($http, $location, $rootScope, analyticsService, stateService) {
	var that = this;

	// Logged in user.
	this.user = undefined;

	// Map of all user objects.
	this.userMap = {};

	// These functions will be added to the current user object.
	let userFuncs = {
		canSubmitLinks: function(domainId) {
			let membership = that.user.domainMembershipMap[domainId];
			return membership && membership.canSubmitLinks;
		},
	};

	// Add the given user to userMap, while doing a smart merge.
	this.addUserToMap = function(newUser) {
		var oldUser = that.userMap[newUser.id];
		if (newUser === oldUser) return oldUser;
		if (oldUser === undefined) {
			that.userMap[newUser.id] = newUser;
			return newUser;
		}
		// Merge each variable.
		for (var k in oldUser) {
			oldUser[k] = stateService.smartMerge(oldUser[k], newUser[k]);
		}
		return oldUser;
	};

	// Call this to process data we received from the server.
	var postDataCallback = function(data) {
		if (data.resetEverything) {
			that.userMap = {};
		}

		if (data.resetEverything || (!that.userIsLoggedIn() && data.user && data.user.id)) {
			that.user = data.user;
			analyticsService.identifyUser(that.user.id, that.user.firstName + ' ' + that.user.lastName, that.user.email, that.user.analyticsId);
			for (var name in userFuncs) {
				that.user[name] = userFuncs[name];
			}
		}

		for (let userId in data.users) {
			that.addUserToMap(data.users[userId]);
		}
	};
	stateService.addPostDataCallback('userService', postDataCallback);

	// Check if the user is logged in.
	this.userIsLoggedIn = function() {
		return this.user && this.user.id != '';
	};

	// Check if the user has received any notifications.
	this.userHasReceivedNotifications = function() {
		return this.user && this.user.hasReceivedNotifications;
	};

	// Check if the user has received any maintenance updates.
	this.userHasReceivedMaintenanceUpdates = function() {
		return this.user && this.user.hasReceivedMaintenanceUpdates;
	};

	// Check if the user can approve comments in the given domain
	this.userCanApproveComments = function(domainId) {
		if (!this.user) return false;
		if (!(domainId in this.user.domainMembershipMap)) return false;
		return this.user.domainMembershipMap[domainId].canApproveComments;
	};

	// Return a user's full name.
	this.getFullName = function(userId) {
		var user = this.userMap[userId];
		if (!user) console.error('User not found: ' + userId);
		return user.firstName + ' ' + user.lastName;
	};

	// Get a map of domains the user can pick from for setting editDomainId/seeDomainId
	// page - optional parameter; if given, the returned list will definitely contain page's edit/see domain id
	this.getDomainOptions = function(page) {
		var domainOptions = {};
		for (var domainId in that.user.domainMembershipMap) {
			domainOptions[domainId] = stateService.domainMap[domainId].alias;
		}
		// Make sure both see and edit domains are in the map.
		if (isIntIdValid(page.seeDomainId)) {
			domainOptions[page.seeDomainId] = stateService.domainMap[page.seeDomainId].alias;
		}
		if (isIntIdValid(page.editDomainId)) {
			domainOptions[page.editDomainId] = stateService.domainMap[page.editDomainId].alias;
		}
		return domainOptions;
	};

	// Sign into FB and call the callback with the response.
	this.fbLogin = function(callback) {
		// Apparently FB.login is not supported in Chrome in iOS
		if (navigator.userAgent.match('CriOS')) {
			var appId = isLive() ? '1064531780272247' : '1064555696936522';
			var redirectUrl = $location.absUrl();
			// NOTE: Because of ngSilentLocation our URL state is fucked somehow, and so for now
			// the best we can do is just redirect the user to the home page.
			// TODO(alexei): now that we are not using ngSilentLocation, fix this
			redirectUrl = 'https://' + window.location.host;
			if (redirectUrl.indexOf('?') < 0 && redirectUrl[redirectUrl.length - 1] != '/') {
				redirectUrl += '/';
			}
			redirectUrl = encodeURIComponent(redirectUrl);
			window.location.href = 'https://www.facebook.com/dialog/oauth?client_id=' + appId +
					'&redirect_uri=' + redirectUrl + '&scope=email,public_profile';
		} else {
			FB.login(function(response) {
				callback(response);
			}, {scope: 'public_profile,email'});
		}
	};

	// Check if FB redirected back to use with the code
	var fbCode = $location.search().code;
	if (!fbCode && $location.search().continueUrl) {
		// For private domains, we end up with the code in the continueUrl.
		var continueUrl = decodeURIComponent($location.search().continueUrl);
		var match = continueUrl.match(/code=[A-Za-z0-9_-]+/);
		if (match && match.length > 0) {
			fbCode = match[0].substring(5);
		}
	}
	if (fbCode) {
		var data = {
			fbCodeToken: fbCode,
			fbRedirectUrl: '',
		};
		$location.search('code', undefined);
		// FB inserts this hash if there was no hash. It's a security thing. We need
		// to remove it to get our original redirectUrl.
		if ($location.hash() == '_=_') {
			$location.hash('');
		}
		data.fbRedirectUrl = 'https://' + window.location.host;
		if (data.fbRedirectUrl.indexOf('?') < 0 && data.fbRedirectUrl[data.fbRedirectUrl.length - 1] != '/') {
			data.fbRedirectUrl += '/';
		}
		$http({method: 'POST', url: '/signup/', data: JSON.stringify(data)})
		.success(function(data, status) {
			window.location.reload();
		})
		.error(function(data, status) {
			console.error('Error FB signup:'); console.log(data); console.log(status);
		});
	}

	// Get data to display a popover for the user with the given alias.
	var loadingUserPopovers = {};
	this.loadUserPopover = function(userId, successFn, errorFn) {
		if (userId in loadingUserPopovers) {
			return;
		}
		loadingUserPopovers[userId] = true;

		var createCallback = function(continuationFn) {
			return function(data) {
				delete loadingUserPopovers[userId];
				if (continuationFn) continuationFn(data);
			};
		};
		stateService.postData('/json/userPopover/', {userId: userId}, createCallback(successFn), createCallback(errorFn));
	};

	// Push user's settings to the server
	this.updateSettings = function(successFn, errorFn) {
		var data = {
			emailFrequency: that.user.emailFrequency,
			emailThreshold: that.user.emailThreshold,
			showAdvancedEditorMode: that.user.showAdvancedEditorMode,
			ignoreMathjax: that.user.ignoreMathjax,
		};
		stateService.postData('/updateSettings/', data, successFn, errorFn);
	};
});
