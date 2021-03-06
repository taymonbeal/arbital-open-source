'use strict';

import app from './angular.ts';

// Directive for showing a window for creating/editing a mark
app.directive('arbMarkInfo', function($interval, arb) {
	return {
		templateUrl: versionUrl('static/html/markInfo.html'),
		scope: {
			// Id of the query mark that was created.
			markId: '@',
			// Set to true if the user just created this mark.
			isNew: '=',
			// Whether we should show the context of the mark.
			showContext: '=',
		},
		controller: function($scope) {
			$scope.arb = arb;
			$scope.mark = arb.markService.markMap[$scope.markId];
			$scope.isOnPage = $scope.mark.pageId == arb.pageService.getCurrentPageId();

			$scope.showContext = $scope.showContext || !$scope.isOnPage;

			// Call to resolve the mark with the given page.
			$scope.resolveWith = function(pageId) {
				arb.markService.resolveMark({
					markId: $scope.markId,
					resolvedPageId: $scope.mark.pageId,
				});
				$scope.mark.resolvedPageId = pageId;
				$scope.mark.resolvedBy = arb.userService.user.id;
				$scope.hidePopup({dismiss: true});
			};

			// Called when an author wants to resolve the mark.
			$scope.dismissMark = function() {
				arb.markService.resolveMark({
					markId: $scope.markId,
					resolvedPageId: '',
				});
				$scope.mark.resolvedPageId = '';
				$scope.mark.resolvedBy = arb.userService.user.id;
				$scope.hidePopup({dismiss: true});
			};
		},
		link: function(scope: any, element, attrs) {
			// Hide current event window, if it makes sense.
			scope.hidePopup = function(result) {
				if (scope.isOnPage) {
					arb.popupService.hidePopup(result);
				}
			};
		},
	};
});
