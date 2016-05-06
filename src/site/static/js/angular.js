'use strict';

// Set up angular module.
var app = angular.module('arbital', ['ngMaterial', 'ngResource',
		'ngMessages', 'ngSanitize', 'RecursionHelper', 'as.sortable']);

app.config(function($locationProvider, $mdIconProvider, $mdThemingProvider) {
	// Convert "rgb(#,#,#)" color to "#hex"
	var rgb2hex = function(rgb) {
		if (rgb === undefined)
			return '#000000';
		rgb = rgb.match(/^rgb\((\d+),\s*(\d+),\s*(\d+)\)$/);
		function hex(x) {
			return ('0' + parseInt(x).toString(16)).slice(-2);
		}
		return '#' + hex(rgb[1]) + hex(rgb[2]) + hex(rgb[3]);
	};
	// Create themes, by getting the colors from our css files
	$mdThemingProvider.definePalette('arb-primary-theme', $mdThemingProvider.extendPalette('teal', {
		'500': rgb2hex($('#primary-color').css('border-top-color')),
		'300': rgb2hex($('#primary-color').css('border-right-color')),
		'800': rgb2hex($('#primary-color').css('border-bottom-color')),
		'A100': rgb2hex($('#primary-color').css('border-left-color')),
		'contrastDefaultColor': 'light',
		'contrastDarkColors': ['300'],
	}));
	$mdThemingProvider.definePalette('arb-accent-theme', $mdThemingProvider.extendPalette('deep-orange', {
		'A200': rgb2hex($('#accent-color').css('border-top-color')),
		'A100': rgb2hex($('#accent-color').css('border-right-color')),
		'A400': rgb2hex($('#accent-color').css('border-bottom-color')),
		'A700': rgb2hex($('#accent-color').css('border-left-color')),
		'contrastDefaultColor': 'dark',
		'contrastLightColors': [],
	}));
	$mdThemingProvider.definePalette('arb-warn-theme', $mdThemingProvider.extendPalette('red', {
		'500': rgb2hex($('#warn-color').css('border-top-color')),
		'300': rgb2hex($('#warn-color').css('border-right-color')),
		'800': rgb2hex($('#warn-color').css('border-bottom-color')),
		'A100': rgb2hex($('#warn-color').css('border-left-color')),
		'contrastDefaultColor': 'light',
		'contrastDarkColors': ['300'],
	}));
	// Set the theme
	$mdThemingProvider.theme('default')
	.primaryPalette('arb-primary-theme', {
		'default': '500',
		'hue-1': '300',
		'hue-2': '800',
		'hue-3': 'A100',
	})
	.accentPalette('arb-accent-theme', {
		'default': 'A200',
		'hue-1': 'A100',
		'hue-2': 'A400',
		'hue-3': 'A700',
	})
	.warnPalette('arb-warn-theme', {
		'default': '500',
		'hue-1': '300',
		'hue-2': '800',
		'hue-3': 'A100',
	});

	// Set up custom icons
	$mdIconProvider.icon('arbital_logo', 'static/icons/arbital-logo.svg', 40)
		.icon('comment_plus_outline', 'static/icons/comment-plus-outline.svg')
		.icon('comment_question_outline', 'static/icons/comment-question-outline.svg')
		.icon('facebook_box', 'static/icons/facebook-box.svg')
		.icon('format_header_pound', 'static/icons/format-header-pound.svg')
		.icon('cursor-pointer', 'static/icons/cursor-pointer.svg')
		.icon('link_variant', 'static/icons/link-variant.svg')
		.icon('thumb_up_outline', 'static/icons/thumb-up-outline.svg')
		.icon('thumb_down_outline', 'static/icons/thumb-down-outline.svg');

	$locationProvider.html5Mode(true);
});

// ArbitalCtrl is used across all pages.
// NOTE: we need to include popoverService, so that it can initialize itself
app.controller('ArbitalCtrl', function($rootScope, $scope, $location, $timeout, $interval, $http, $compile, $anchorScroll, $mdDialog, userService, pageService, popoverService, urlService) {
	$scope.urlService = urlService;
	$scope.pageService = pageService;
	$scope.userService = userService;

	// Refresh all the fields that need to be updated every so often.
	var refreshAutoupdates = function() {
		$('.autoupdate').each(function(index, element) {
			$compile($(element))($scope);
		});
		$timeout(refreshAutoupdates, 30000);
	};
	refreshAutoupdates();

	// Returns an object containing a compiled element and its scope
	$scope.newElement = function(html, parentScope) {
		if (!parentScope) parentScope = $scope;
		var childScope = parentScope.$new();
		var element = $compile(html)(childScope);
		return {
			scope: childScope,
			element: element,
		};
	};
	// The element and it scope inside ng-view for the current page
	var currentView;

	// Returns a function we can use as success handler for POST requests for dynamic data.
	// callback - returns {
	//   title: title to set for the window
	//   element: optional jQuery element to add dynamically to the body
	//   error: optional error message to print
	// }
	$scope.getSuccessFunc = function(callback) {
		return function(data) {
			// Sometimes we don't get data.
			if (data) {
				console.log('Dynamic request data:'); console.log(data);
				userService.processServerData(data);
				pageService.processServerData(data);
			}

			// Because the subdomain could have any case, we need to find the alias
			// in the loaded map so we can get the alias with correct case
			if ($scope.subdomain) {
				for (var pageAlias in pageService.pageMap) {
					if ($scope.subdomain.toUpperCase() === pageAlias.toUpperCase()) {
						$scope.subdomain = pageAlias;
						pageService.privateGroupId = pageService.pageMap[pageAlias].pageId;
						break;
					}
				}
			}

			if (currentView) {
				currentView.scope.$destroy();
				currentView.element.remove();
				currentView = null;
				urlService.hasLoadedFirstPage = true;
			}

			// Get the results from page-specific callback
			$('.global-error').hide();
			var result = callback(data);
			if (result.error) {
				$('.global-error').text(result.error).show();
				document.title = 'Error - Arbital';
			}
			if (result.content) {
				// Only show the element after it and all the children have been fully compiled and linked
				result.content.element.addClass('reveal-after-render-parent');
				var $loadingBar = $('#loading-bar');
				$loadingBar.show();
				var startTime = (new Date()).getTime();

				var showEverything = function() {
					$interval.cancel(revealInterval);
					$timeout.cancel(revealTimeout);
					// Do short timeout to prevent some rendering bugs that occur on edit page
					$timeout(function() {
						result.content.element.removeClass('reveal-after-render-parent');
						$loadingBar.hide();
						$anchorScroll();
					}, 50);
				};

				var revealInterval = $interval(function() {
					var timePassed = ((new Date()).getTime() - startTime) / 1000;
					var hiddenChildren = result.content.element.find('.reveal-after-render');
					if (hiddenChildren.length > 0) {
						hiddenChildren.each(function() {
							if ($(this).children().length > 0) {
								$(this).removeClass('reveal-after-render');
							}
						});
						return;
					}
					showEverything();
				}, 50);
				// Do a timeout as well, just in case we have a buggy element
				var revealTimeout = $timeout(function() {
					console.error('Forced reveal timeout');
					showEverything();
				}, 1000);

				currentView = result.content;
				$('[ng-view]').append(result.content.element);
			}

			$('body').toggleClass('body-fix', !result.removeBodyFix);

			if (result.title) {
				document.title = result.title + ' - Arbital';
			}
		};
	};

	// Returns a function we can use as error handler for POST requests for dynamic data.
	$scope.getErrorFunc = function(urlPageType) {
		return function(data, status) {
			console.error('Error /json/' + urlPageType + '/:'); console.log(data); console.log(status);
			pageService.showToast({text: 'Couldn\'t create a new page: ' + data, isError: true});
			document.title = 'Error - Arbital';
		};
	};

	// Check to see if we should show the popup.
	$scope.closePopup = function() {
		pageService.hideNonpersistentPopup();
	};

	// Watch path changes and update Google Analytics
	$scope.$watch(function() {
		return $location.absUrl();
	}, function() {
		ga('send', 'pageview', $location.absUrl());
	});

	// The URL rule match for the current page
	var currentLocation = {};
	function resolveUrl() {
		// Get subdomain if any
		$scope.subdomain = undefined;
		var subdomainMatch = /^([A-Za-z0-9_]+)\.(localhost|arbital\.com)\/?$/.exec($location.host());
		if (subdomainMatch) {
			$scope.subdomain = subdomainMatch[1];
		}
		var path = $location.path();
		var urlRules = urlService.urlRules;
		for (var ruleIndex = 0; ruleIndex < urlRules.length; ruleIndex++) {
			var rule = urlRules[ruleIndex];
			var matches = rule.urlPattern.exec(path);
			if (matches) {
				var args = {};
				var parameters = rule.parameters;
				for (var parameterIndex = 0; parameterIndex < parameters.length; parameterIndex++) {
					var parameter = parameters[parameterIndex];
					args[parameter] = matches[parameterIndex + 1];
				}
				if (rule == currentLocation.rule && $scope.subdomain == currentLocation.subdomain) {
					var currentMatches = true;
					for (parameterIndex = 0; parameterIndex < parameters.length && currentMatches; parameterIndex++) {
						var parameter = parameters[parameterIndex];
						currentMatches = (args[parameter] == currentLocation.args[parameter]);
					}
					if (currentMatches) {
						// The host and path have not changed, don't reload
						return;
					}
				}
				var handled = rule.handler(args, $scope);
				if (!handled) {
					$('[ng-view]').empty();
					$scope.closePopup();
				}
				currentLocation = {subdomain: $scope.subdomain, rule: rule, args: args};
				return;
			}
		}
	};

	$rootScope.$on('$locationChangeSuccess', function(event, url) {
		resolveUrl();
	});

	// Resolve URL of initial page load
	resolveUrl();
});

app.run(function($http, $location, urlService, pageService, userService) {
	// Set up mapping from URL path to specific controllers
	urlService.addUrlHandler('/', {
		name: 'IndexPage',
		handler: function(args, $scope) {
			if ($scope.subdomain) {
				// Get the private domain index page data
				$http({method: 'POST', url: '/json/domainPage/', data: JSON.stringify({})})
				.success($scope.getSuccessFunc(function(data) {
					$scope.indexPageIdsMap = data.result;
					return {
						title: pageService.pageMap[$scope.subdomain].title + ' - Private Domain',
						content: $scope.newElement('<arb-group-index group-id=\'' + data.result.domainId +
							'\' ids-map=\'::indexPageIdsMap\'></arb-group-index>'),
					};
				}))
				.error($scope.getErrorFunc('domainPage'));
			} else {
				// Get the index page data
				$http({method: 'POST', url: '/json/index/'})
				.success($scope.getSuccessFunc(function(data) {
					$scope.featuredDomains = data.result.featuredDomains;
					return {
						title: '',
						content: $scope.newElement('<arb-index featured-domains=\'::featuredDomains\'></arb-index>'),
					};
				}))
				.error($scope.getErrorFunc('index'));
			}
		},
	});
	urlService.addUrlHandler('/adminDashboard/', {
		name: 'AdminDashboardPage',
		handler: function(args, $scope) {
			var postData = {};
			// Get the data
			$http({method: 'POST', url: '/json/adminDashboardPage/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				$scope.adminDashboardData = data.result;
				return {
					title: 'Admin dashboard',
					content: $scope.newElement('<arb-admin-dashboard-page data=\'::adminDashboardData\'></arb-admin-dashboard-page>'),
				};
			}))
			.error($scope.getErrorFunc('adminDashboardPage'));
		},
	});
	urlService.addUrlHandler('/dashboard/', {
		name: 'DashboardPage',
		handler: function(args, $scope) {
			var postData = {};
			// Get the data
			$http({method: 'POST', url: '/json/dashboardPage/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				$scope.dashboardPageIdsMap = data.result;
				return {
					title: 'Your dashboard',
					content: $scope.newElement('<arb-dashboard-page ids-map=\'::dashboardPageIdsMap\'></arb-dashboard-page>'),
				};
			}))
			.error($scope.getErrorFunc('dashboardPage'));
		},
	});
	urlService.addUrlHandler('/domains/:alias', {
		name: 'DomainPageController',
		handler: function(args, $scope) {
			pageService.domainAlias = args.alias;
			var postData = {
				domainAlias: pageService.domainAlias,
			};
			// Get the domain index page data
			$http({method: 'POST', url: '/json/domainPage/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				$scope.indexPageIdsMap = data.result;
				var groupId = pageService.pageMap[pageService.domainAlias].pageId;
				return {
					title: pageService.pageMap[groupId].title,
					content: $scope.newElement('<arb-group-index group-id=\'' + groupId +
						'\' ids-map=\'::indexPageIdsMap\'></arb-group-index>'),
				};
			}))
			.error($scope.getErrorFunc('domainPage'));
		},
	});
	urlService.addUrlHandler('/edit/:alias?/:alias2?', {
		name: 'EditPage',
		handler: function(args, $scope) {
			var pageAlias = args.alias;
			// Check if we are already editing this page.
			if (pageService.primaryPage &&
					pageService.pageMap[pageAlias].pageId === pageService.primaryPage.pageId) {
				return true;
			}

			// Load the last edit for the pageAlias.
			var loadEdit = function() {
				pageService.loadEdit({
					pageAlias: pageAlias,
					success: $scope.getSuccessFunc(function() {
						// Find the page in the editMap (have to search through it manually
						// because we don't index pages by alias in editmap)
						var page;
						for (var pageId in pageService.editMap) {
							page = pageService.editMap[pageId];
							if (page.alias == pageAlias || page.pageId == pageAlias) {
								break;
							}
						}
						if ($location.search().alias) {
							// Set page's alias
							page.alias = $location.search().alias;
							$location.replace().search('alias', undefined);
						}

						urlService.ensureCanonPath(pageService.getEditPageUrl(page.pageId));
						pageService.primaryPage = page;

						// Called when the user is done editing the page.
						$scope.doneFn = function(result) {
							var page = pageService.editMap[result.pageId];
							if (!page.wasPublished && result.discard) {
								$location.path('/edit/');
							} else {
								$location.url(pageService.getPageUrl(page.pageId, {
									useEditMap: true,
									markId: $location.search().markId,
									permalink: result.deletedPage,
								}));
							}
						};
						return {
							removeBodyFix: true,
							title: 'Edit ' + (page.title ? page.title : 'New Page'),
							content: $scope.newElement('<arb-edit-page class=\'full-height\' page-id=\'' + page.pageId +
								'\' done-fn=\'doneFn(result)\' layout=\'column\'></arb-edit-page>'),
						};
					}),
					error: $scope.getErrorFunc('edit'),
				});
			};

			// Load a new page.
			var getNewPage = function () {
				var type = $location.search().type;
				$location.replace().search('type', undefined);
				var newParentIdString = $location.search().newParentId;
				$location.replace().search('newParentId', undefined);
				// Create a new page to edit
				pageService.getNewPage({
					type: type,
					parentIds: newParentIdString ? newParentIdString.split(',') : [],
					success: function(newPageId) {
						$location.replace().path(pageService.getEditPageUrl(newPageId));
					},
					error: $scope.getErrorFunc('newPage'),
				});
			};

			// Need to call /default/ in case we are creating a new page
			// TODO(alexei): have /newPage/ return /default/ data along with /edit/ data
			$http({method: 'POST', url: '/json/default/'})
			.success($scope.getSuccessFunc(function(data) {
				if (pageAlias) {
					loadEdit();
				} else {
					getNewPage();
				}
				return {
					title: 'Edit Page',
				};
			}))
			.error($scope.getErrorFunc('default'));
		},
	});
	urlService.addUrlHandler('/groups/', {
		name: 'GroupsPage',
		handler: function(args, $scope) {
			$http({method: 'POST', url: '/json/groups/'})
			.success($scope.getSuccessFunc(function(data) {
				return {
					title: 'Groups',
					content: $scope.newElement('<arb-groups-page></arb-groups-page>'),
				};
			}))
			.error($scope.getErrorFunc('groups'));
		},
	});
	urlService.addUrlHandler('/learn/:pageAlias?/:pageAlias2?', {
		name: 'LearnPage',
		handler: function(args, $scope) {
			// Get the primary page data
			var postData = {
				pageAliases: [],
				onlyWanted: $location.search()['only_wanted'] === '1', // jscs:ignore requireDotNotation
			};
			var continueLearning = false;
			if (args.pageAlias) {
				postData.pageAliases.push(args.pageAlias);
			} else if ($location.search().path) {
				postData.pageAliases = postData.pageAliases.concat($location.search().path.split(','));
			} else if (pageService.path) {
				postData.pageAliases = pageService.path.pageIds;
				continueLearning = true;
			}

			$http({method: 'POST', url: '/json/learn/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				var primaryPage = undefined;
				if (args.pageAlias) {
					primaryPage = pageService.pageMap[args.pageAlias];
					urlService.ensureCanonPath('/learn/' + primaryPage.alias);
				}

				$scope.learnPageIds = data.result.pageIds;
				$scope.learnOptionsMap = data.result.optionsMap;
				$scope.learnTutorMap = data.result.tutorMap;
				$scope.learnRequirementMap = data.result.requirementMap;
				return {
					title: 'Learn ' + (primaryPage ? primaryPage.title : ''),
					content: $scope.newElement('<arb-learn-page continue-learning=\'::' + continueLearning +
						'\' page-ids=\'::learnPageIds\'' +
						'\' options-map=\'::learnOptionsMap\'' +
						' tutor-map=\'::learnTutorMap\' requirement-map=\'::learnRequirementMap\'' +
						'></arb-learn-page>'),
				};
			}))
			.error($scope.getErrorFunc('learn'));
		},
	});
	urlService.addUrlHandler('/login/', {
		name: 'LoginPage',
		handler: function(args, $scope) {
			$http({method: 'POST', url: '/json/default/'})
			.success($scope.getSuccessFunc(function(data) {
				if (userService.user.id) {
					window.location.href = urlService.getDomainUrl();
				}
				return {
					title: 'Log In',
					content: $scope.newElement('<div class=\'md-whiteframe-1dp capped-body-width\'><arb-login></arb-login></div>'),
				};
			}))
			.error($scope.getErrorFunc('default'));
		},
	});
	urlService.addUrlHandler('/p/:alias/:alias2?', {
		name: 'PrimaryPage',
		handler: function(args, $scope) {
			// Get the primary page data
			var postData = {
				pageAlias: args.alias,
				markId: $location.search().markId,
			};
			$http({method: 'POST', url: '/json/primaryPage/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				var page = pageService.pageMap[postData.pageAlias];
				var pageTemplate = '<arb-primary-page></arb-primary-page>';

				if (!page) {
					page = pageService.deletedPagesMap[postData.pageAlias];
					if (page) {
						if (page.mergedInto) {
							urlService.goToUrl(pageService.getPageUrl(page.mergedInto));
						} else {
							urlService.goToUrl(pageService.getEditPageUrl(postData.pageAlias));
						}
						return {};
					}
					return {
						title: 'Not Found',
						error: 'Page doesn\'t exist, was deleted, or you don\'t have permission to view it.',
					};
				}

				// If this page has been merged into another, go there
				if (page.mergedInto) {
					urlService.goToUrl(pageService.getPageUrl(page.mergedInto));
					return {};
				}

				// If the page is a user page, get the additional data about user
				// - Recently created by me page ids.
				// - Recently created by me comment ids.
				// - Recently edited by me page ids.
				// - Top pages by me
				if (userService.userMap[page.pageId]) {
					$scope.userPageIdsMap = data.result;
					pageTemplate = '<arb-user-page user-id=\'' + page.pageId +
							'\' user_page_data=\'::userPageIdsMap\'></arb-user-page>';
				}

				if (page.isLens() || page.isComment()) {
					// Redirect to the primary page, but preserve all search variables
					var search = $location.search();
					$location.replace().url(pageService.getPageUrl(page.pageId));
					for (var k in search) {
						$location.search(k, search[k]);
					}
					return {};
				}

				urlService.ensureCanonPath(pageService.getPageUrl(page.pageId));
				pageService.primaryPage = page;
				return {
					title: page.title,
					content: $scope.newElement(pageTemplate),
				};
			}))
			.error($scope.getErrorFunc('primaryPage'));
		},
	});
	urlService.addUrlHandler('/pages/:alias', {
		name: 'RedirectToPrimaryPage',
		handler: function(args, $scope) {
			// Get the primary page data
			var postData = {
				pageAlias: args.alias,
			};
			$http({method: 'POST', url: '/json/redirectToPrimaryPage/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				var pageId = data;
				if (!pageId) {
					return {
						title: 'Not Found',
						error: 'Page doesn\'t exist, was deleted, or you don\'t have permission to view it.',
					};
				}
				// Redirect to the primary page, but preserve all search variables
				var search = $location.search();
				$location.replace().url(pageService.getPageUrl(pageId));
				for (var k in search) {
					$location.search(k, search[k]);
				}
				return {
				};
			}))
			.error($scope.getErrorFunc('redirectToPrimaryPage'));
		},
	});
	urlService.addUrlHandler('/requisites/', {
		name: 'RequisitesPage',
		handler: function(args, $scope) {
			$http({method: 'POST', url: '/json/requisites/'})
			.success($scope.getSuccessFunc(function(data) {
				return {
					title: 'Requisites',
					content: $scope.newElement('<arb-requisites-page></arb-requisites-page>'),
				};
			}))
			.error($scope.getErrorFunc('requisites'));
		},
	});
	urlService.addUrlHandler('/settings/', {
		name: 'SettingsPage',
		handler: function(args, $scope) {
			$http({method: 'POST', url: '/json/settingsPage/'})
			.success($scope.getSuccessFunc(function(data) {
				if (data.result) {
					$scope.domains = data.result.domains;
					// Convert invitesSent object to array for ease in angular
					$scope.invitesSent = [];
					for (var key in data.result.invitesSent) {
						$scope.invitesSent.push(data.result.invitesSent[key]);
					}
				}
				return {
					title: 'Settings',
					content: $scope.newElement('<arb-settings-page domains="::domains" ' +
						'invites-sent="::invitesSent"></arb-settings-page>'),
				};
			}))
			.error($scope.getErrorFunc('settingsPage'));
		},
	});
	urlService.addUrlHandler('/signup/', {
		name: 'SignupPage',
		handler: function(args, $scope) {
			$http({method: 'POST', url: '/json/default/'})
			.success($scope.getSuccessFunc(function(data) {
				if (userService.user.id) {
					window.location.href = urlService.getDomainUrl();
				}
				return {
					title: 'Sign Up',
					content: $scope.newElement('<arb-signup></arb-signup>'),
				};
			}))
			.error($scope.getErrorFunc('default'));
		},
	});
	urlService.addUrlHandler('/updates/', {
		name: 'UpdatesPage',
		handler: function(args, $scope) {
			var postData = {};
			// Get the explore data
			$http({method: 'POST', url: '/json/updates/', data: JSON.stringify(postData)})
			.success($scope.getSuccessFunc(function(data) {
				$scope.updateGroups = data.result.updateGroups;
				return {
					title: 'Updates',
					content: $scope.newElement('<arb-updates update-groups=\'::updateGroups\'></arb-updates>'),
				};
			}))
			.error($scope.getErrorFunc('updates'));
		},
	});
});

// simpleDateTime filter converts our typical date&time string into local time.
app.filter('simpleDateTime', function() {
	return function(input) {
		return moment.utc(input).local().format('LT, l');
	};
});

// relativeDateTime converts date&time into a relative string, e.g. "5 days ago"
app.filter('relativeDateTime', function() {
	return function(input) {
		if (moment.utc().diff(moment.utc(input), 'days') <= 7) {
			return moment.utc(input).fromNow();
		} else {
			return moment.utc(input).local().format('MMM Do, YYYY [at] LT');
		}
	};
});
app.filter('relativeDateTimeNoSuffix', function() {
	return function(input) {
		return moment.utc(input).fromNow(true);
	};
});

// numSuffix filter converts a number string to a 2 digit number with a suffix, e.g. K, M, G
app.filter('numSuffix', function() {
	return function(input) {
		var num = +input;
		if (num >= 100000) return (Math.round(num / 100000) / 10) + 'M';
		if (num >= 100) return (Math.round(num / 100) / 10) + 'K';
		return input;
	};
});

// shorten filter shortens a string to the given number of characters
app.filter('shorten', function() {
	return function(input, charCount) {
		if (!input || input.length <= charCount) return input;
		var s = input.substring(0, charCount);
		var lastSpaceIndex = s.lastIndexOf(' ');
		if (lastSpaceIndex < 0) return s + '...';
		return input.substring(0, lastSpaceIndex) + '...';
	};
});
