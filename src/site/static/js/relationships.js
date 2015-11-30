"use strict";

// Directive for showing the parents, children, tags, or requirements.
app.directive("arbRelationships", function($q, $timeout, $http, pageService, userService, autocompleteService) {
	return {
		templateUrl: "/static/html/relationships.html",
		scope: {
			pageId: "@",
			type: "@",
			customTitle: "@",
			forceEditMode: "@",
		},
		controller: function($scope) {
			$scope.pageService = pageService;
			$scope.userService = userService;
			if ($scope.forceEditMode) {
				$scope.page = pageService.editMap[$scope.pageId];
			} else {
				$scope.page = pageService.pageMap[$scope.pageId];
			}
			$scope.inEditMode = $scope.forceEditMode;

			// Helper variables
			$scope.isParentType = $scope.type === "parent";
			$scope.isTagType = $scope.type === "tag";
			$scope.isRequirementType = $scope.type === "requirement";

			// Compute various variables based on the type
			if ($scope.isParentType) {
				$scope.title = "Parents";
				$scope.idsSource = $scope.page.parentIds;
			} else if ($scope.isTagType) {
				$scope.title = "Tags";
				$scope.idsSource = $scope.page.taggedAsIds;
			} else if ($scope.isRequirementType) {
				$scope.title = "Requirements";
				$scope.idsSource = $scope.page.requirementIds;
			}
			if ($scope.customTitle) {
				$scope.title = $scope.customTitle;
			}

			// Compute if we should show the panel
			$scope.showPanel = $scope.forceEditMode;

			// Do some custom stuff for requirements
			if ($scope.isRequirementType) {
				// Check if the user has the given mastery.
				$scope.hasMastery = function(requirementId) {
					return pageService.masteryMap[requirementId].has;
				}

				// Don't show the panel if the user has met all the requirements
				if (!$scope.foceEditMode) {
					for (var n = 0; n < $scope.page.requirementIds.length; n++) {
						$scope.showPanel |= !$scope.hasMastery($scope.page.requirementIds[n]);
					}
				}
	
				// Sort requirements
				$scope.page.requirementIds.sort(function(a, b) {
					return ($scope.hasMastery(a) ? 1 : 0) - ($scope.hasMastery(b) ? 1 : 0);
				});
			}

			// Toggle edit mode.
			$scope.inEditModeToggle = function() {
				$scope.inEditMode = !$scope.inEditMode;
			};

			// Set up search
			$scope.getSearchResults = function(text) {
				if (!text) return [];
				var deferred = $q.defer();
				autocompleteService.parentsSource({term: text}, function(results) {
					deferred.resolve(results);
				});
        return deferred.promise;
			};
			$scope.searchResultSelected = function(result) {
				if (!result) return;
				var data = {
					parentId: result.label,
					childId: $scope.page.pageId,
					type: $scope.type,
				};
				$http({method: "POST", url: "/newPagePair/", data: JSON.stringify(data)})
				.error(function(data, status){
					console.error("Error creating a " + $scope.type + ":"); console.log(data); console.log(status);
				});

				if ($scope.isRequirementType) {
					pageService.masteryMap[data.parentId] = {pageId: data.parentId, isMet: true, isManuallySet: true};
				}
				$scope.idsSource.push(data.parentId);
			}

			// Process deleting a relationship
			$scope.deleteRelationship = function(otherPageId) {
				var options = {
					parentId: otherPageId,
					childId: $scope.page.pageId,
					type: $scope.type,
				};
				pageService.deletePagePair(options);
				$scope.idsSource.splice($scope.idsSource.indexOf(options.parentId), 1);
			};

			// Toggle whether or not the user meets a requirement
			$scope.toggleRequirement = function(requirementId) {
				pageService.updateMastery($scope, requirementId, !$scope.hasMastery(requirementId));
			};
		},
	};
});

