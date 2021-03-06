'use strict';

import app from './angular.ts';
import {isLive} from './util.ts';

declare var ga: any;
declare var heap: any;

// arb.analyticsService is a wrapper for Google Analytics
app.service('analyticsService', function($http, $location, stateService) {
	var that = this;

	// Function to send data to both Mixpanel and Heap
	this.reportEventToHeapAndMixpanel = function(action, data) {
		heap.track(action, data);
	};
	var track = this.reportEventToHeapAndMixpanel;

	// This is called the first time user is signed up.
	this.signupSuccess = function(userId) {
		track('Signup', {userId: userId});
	};

	// This is called to identify the user to the analytics platforms.
	this.identifyUser = function(userId, fullName, email, analyticsId) {
		heap.addUserProperties({
			'analyticsId': analyticsId,
		});

		if (!!userId) {
			heap.identify(userId);
			track('Identify');

			// full story
			let id = userId;
			if (id == '1') {
				// full story can't handle a user id of '1' (see: http://help.fullstory.com/develop-js/identify)
				id = 'alexei';
			}
		}

		if (!isLive()) return;
		ga('set', 'userId', userId);
	};

	// This is called when a user goes to any web page.
	this.reportWebPageView = function(data) {
		data = data || {};
		data.path = $location.path();
		track('Web page view', data);

		if (!isLive()) return;
		// Set the page, which which will be included with all future events.
		ga('set', 'page', $location.path());
		// Send "pageview" event, since we switched new a new view
		ga('send', 'pageview');
	};

	// This is called when a user goes to read a page.
	this.reportPageIdView = function(pageId) {
		track('Page view', {pageId: pageId});
		// Set the page, which which will be included with all future events.
		ga('set', 'pageId', pageId);
	};

	// This is called when a user switches lenses.
	this.reportLensSwitch = function(fromPageId, toPageId) {
		track('Lens switch', {fromPageId: fromPageId, toPageId: toPageId});
	};

	// This is called when a page popover is diplayed.
	this.reportPopover = function(pageId) {
		track('Popover', {
			pageId: pageId,
			primaryPageId: stateService.primaryPage ? stateService.primaryPage.pageId : undefined,
		});
	};

	// Called when the user does something with the path/arc they are on.
	this.reportPathUpdate = function(path) {
		track('Path step', {
			guideId: path.guideId,
			pathId: path.id,
			pagesCount: path.pages.length - 1,
			progress: path.progress,
			percentComplete: Math.round(100 * path.progress / (path.pages.length - 1)),
		});
		// Create a single event that we can use for funnels
		track('Arc ' + path.guideId + '; step ' + path.progress);
	};

	// Called when a user edits a page
	this.reportEditPageAction = function(event, action) {
		track(action);

		if (!isLive()) return;
		ga('send', {
			hitType: 'event',
			eventCategory: 'Edit',
			eventAction: action,
			eventLabel: event.target.href,
			eventValue: 1,
		});
	};

	// Called when a user submits a page to domain
	this.reportPageToDomainSubmission = function() {
		if (!isLive()) return;
		ga('send', {
			hitType: 'event',
			eventCategory: 'Page',
			eventAction: 'submitToDomain',
			eventLabel: '1lw',
			eventValue: 1,
		});
	};

	// Called when a user does something with the signup dialog
	this.reportSignupAction = function(action, attemptedAction) {
		if (!isLive()) return;
		ga('send', {
			hitType: 'event',
			eventCategory: 'Signup',
			eventAction: action,
			eventLabel: attemptedAction,
			eventValue: 1,
		});
	};

	// Called when a user publishes a page
	this.reportPublishAction = function(action, pageId, length) {
		if (!isLive()) return;
		ga('send', {
			hitType: 'event',
			eventCategory: 'Publish',
			eventAction: action,
			eventLabel: pageId,
			eventValue: length,
		});
	};
});
