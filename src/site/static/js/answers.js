'use strict';

// Directive for the list of answers
app.directive('arbAnswers', function($timeout, $http, $compile, pageService, userService) {
	return {
		templateUrl: 'static/html/answers.html',
		scope: {
			pageId: '@',
			// Variable that toggles whether to show the delete buttons
			showDelete: '=',
		},
		controller: function($scope) {
			$scope.pageService = pageService;
			$scope.userService = userService;
			$scope.page = pageService.pageMap[$scope.pageId];

			// Called from autocomplete when a new answer should be added.
			$scope.addAnswer = function(result) {
				if (!result) return;
				var postData = {
					questionId: $scope.pageId,
					answerPageId: result.pageId,
				};
				$http({method: 'POST', url: '/newAnswer/', data: JSON.stringify(postData)})
				.success(function(data) {
					$scope.page.answers.push(data.result.newAnswer);
				})
				.error(function(data) {
					console.error('Couldn\'t add answer:'); console.error(data);
				});
			};

			// Delete the given answer
			$scope.deleteAnswer = function(answerId) {
				var postData = {
					answerId: answerId,
				};
				$http({method: 'POST', url: '/deleteAnswer/', data: JSON.stringify(postData)})
				.success(function(data) {
					for (var n = 0; n < $scope.page.answers.length; n++) {
						var answer = $scope.page.answers[n];
						if (answer.id == answerId) {
							$scope.page.answers.splice(n, 1);
							break;
						}
					}
				})
				.error(function(data) {
					console.error('Couldn\'t add answer:'); console.error(data);
				});
			};
		},
	};
});

