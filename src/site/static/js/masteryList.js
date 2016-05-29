'use strict';

// Directive for listing masteries and allowing the user to claim them.
app.directive('arbMasteryList', function($timeout, $http, arb) {
	return {
		templateUrl: 'static/html/masteryList.html',
		scope: {
			idsSource: '=',
			// If true, don't show the checkboxes
			hideCheckboxes: '=',
			// If true, show the requisites the user has first
			showHasFirst: '=',
			// If true, recursively show requirements for each mastery
			showRequirements: '=',
			// If true, show clickbait for all the masteries
			showClickbait: '=',
			// If true, show the requirements on one line
			isSpan: '=',
			// Optional callback, which will receive results when pages are unlocked
			unlockedFn: '&',
		},
		controller: function($scope) {
			$scope.arb = arb;

			// Filter non-existing page ids out
			$scope.idsSource = $scope.idsSource.filter(function(pageId) {
				return (pageId in arb.pageService.pageMap) && !arb.pageService.pageMap[pageId].isDeleted;
			});

			// Sort requirements
			$scope.idsSource.sort(function(a, b) {
				var result = (arb.pageService.hasMastery(a) ? 1 : 0) - (arb.pageService.hasMastery(b) ? 1 : 0);
				if ($scope.showHasFirst) result = -result;
				if (result !== 0) return result;
				result = (arb.pageService.wantsMastery(a) ? 1 : 0) - (arb.pageService.wantsMastery(b) ? 1 : 0);
				if ($scope.showHasFirst) result = -result;
				if (result !== 0) return result;
				return arb.pageService.pageMap[a].title.localeCompare(arb.pageService.pageMap[b].title);
			});

			// Called when one of the requisites is changed.
			$scope.pageUnlocked = function(result) {
				if (!$scope.unlockedFn) return;
				$scope.unlockedFn({result: result});
			};
		},
	};
});
