'use strict';

// toolbar directive displays the toolbar at the top of each page
app.directive('arbToolbar', function($mdSidenav, $http, $mdPanel, $location, $compile, $rootScope, $timeout,
		$q, $mdMedia, arb) {
	return {
		templateUrl: 'static/html/toolbar.html',
		scope: {
			loadingBarValue: '=',
			currentUrl: '=',
		},
		controller: function($scope) {
			$scope.arb = arb;
			$scope.isTinyScreen = !$mdMedia('gt-xs');

			$scope.doAutofocus = function() {
				return !arb.isTouchDevice && !arb.urlService.hasLoadedFirstPage;
			};

			// Called when a search result is selected
			$scope.searchResultSelected = function(result) {
				if (result) {
					arb.urlService.goToUrl(arb.urlService.getPageUrl(result.pageId));
				}
			};

			$scope.getSignupUrl = function() {
				return '/signup/?continueUrl=' + encodeURIComponent($location.absUrl());
			};

			// Open RHS menu
			$scope.toggleRightMenu = function() {
				$mdSidenav('right').toggle();
			};

			$scope.logout = function() {
				Cookies.remove('masteryMap');
				Cookies.remove('arbital');
				window.location.reload();
			};

			// Hide toolbar in the edit screen
			$scope.$on('$locationChangeSuccess', function() {
				$scope.hide = $location.path().indexOf('/edit') === 0;
			});
			$scope.hide = $location.path().indexOf('/edit') === 0;

			$scope.showNotifications = function(ev) {
				arb.userService.user.newNotificationCount = 0;
				showPanel(
					ev,
					'/notifications/',
					'.notifications-icon',
					'<arb-updates-panel post-url="/json/notifications/" hide-title="true" num-to-display="100" more-link="/notifications"></arb-udpates-panel>'
				);
			};

			$scope.showAchievements = function(ev) {
				arb.userService.user.newAchievementCount = 0;
				showPanel(
					ev,
					'/achievements/',
					'.achievements-icon',
					'<arb-hedons-mode-panel hide-title="true" num-to-display="100"></arb-hedons-mode-panel>'
				);
			};

			$scope.showMaintenanceUpdates = function(ev) {
				arb.userService.user.maintenanceUpdateCount = 0;
				showPanel(
					ev,
					'/maintain/',
					'.maintenance-updates-icon',
					'<arb-updates-panel post-url="/json/maintain/" hide-title="true" num-to-display="100" more-link="/maintain"></arb-updates-panel>'
				);
			};

			var showPanel = function(ev, fullPageUrl, relPosElement, panelTemplate) {
				if (!$mdMedia('gt-sm')) {
					arb.urlService.goToUrl(fullPageUrl);
					return;
				}

				var position = $mdPanel.newPanelPosition()
					.relativeTo(relPosElement)
					.addPanelPosition($mdPanel.xPosition.ALIGN_END, $mdPanel.yPosition.BELOW);
				var config = {
					template: panelTemplate,
					position: position,
					panelClass: 'popover-panel',
					openFrom: ev,
					clickOutsideToClose: true,
					escapeToClose: true,
					focusOnOpen: false,
					zIndex: 200000,
				};
				var panel = $mdPanel.create(config);
				panel.open();

				$scope.$on('$locationChangeSuccess', function() {
					panel.close();
				});
			};
		},
	};
});
