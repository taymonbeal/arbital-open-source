'use strict';

import app from './angular.ts';

// Directive for the entire primary page.
app.directive('arbPrimaryPage', function($compile, $location, $timeout, arb) {
	return {
		templateUrl: versionUrl('static/html/primaryPage.html'),
		scope: {
			noFooter: '=',
		},
		controller: function($scope) {
			$scope.arb = arb;
			$scope.page = arb.stateService.primaryPage;
			$scope.page.childIds.sort(arb.pageService.getChildSortFunc($scope.page.sortChildrenBy));
			$scope.page.relatedIds.sort(arb.pageService.getChildSortFunc('likes'));
		},
		link: function(scope: any, element, attrs) {
			if (scope.page.editDomainId == '1') {
				element.addClass('math-background');
			}
		},
	};
});
