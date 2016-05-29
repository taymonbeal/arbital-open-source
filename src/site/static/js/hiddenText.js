'use strict';

// Directive for hidden text (usually for homework problems)
app.directive('arbHiddenText', function(arb) {
	return {
		templateUrl: 'static/html/hiddenText.html',
		transclude: true,
		scope: {
			buttonText: '@',
		},
		controller: function($scope) {
			$scope.arb = arb;
			
			$scope.revealed = false;

			$scope.reveal = function() {
				$scope.revealed = true;
			};
		},
	};
});

