"use strict";

// Directive for the Requisites page.
app.directive("arbRequisitesPage", function(pageService, userService) {
	return {
		templateUrl: "static/html/requisitesPage.html",
		scope: {
		},
		controller: function($scope) {
			$scope.pageService = pageService;
			$scope.userService = userService;

			$scope.masteryList = [];
			for (var id in pageService.masteryMap) {
				$scope.masteryList.push(id);
			}
		},
	};
});