import app from './angular.ts';

import {aliasMatch} from './markdownService.ts';

// Directive to show the discussion section for a page
app.directive('arbPageDiscussion', function($compile, $location, $timeout, arb) {
	return {
		templateUrl: versionUrl('static/html/pageDiscussion.html'),
		scope: {
			pageId: '@',
		},
		controller: function($scope) {
			$scope.arb = arb;
			$scope.page = arb.stateService.pageMap[$scope.pageId];
			$scope.page.subpageIds = $scope.page.commentIds || [];
			$scope.page.subpageIds.sort(arb.pageService.getChildSortFunc('likes'));
			$scope.showNewCommentLoading = false;
			$scope.state = {showEditorComments: false};

			// Return text for the "new comment" button
			$scope.getNewCommentText = function() {
				var text = '';
				if (!$scope.page.permissions.comment.has) {
					text += 'Propose';
				} else if ($scope.visibleCommentCount() <= 0) {
					text += 'Write first';
				} else {
					text += 'New';
				}
				if ($scope.state.showEditorComments) {
					text += ' editor comment';
				} else {
					text += ' comment';
				}
				return text;
			};

			// Process user clicking on New Comment button
			$scope.newCommentClick = function() {
				arb.signupService.wrapInSignupFlow('new comment', function() {
					$scope.showNewCommentLoading = true;
					arb.pageService.newComment({
						parentPageId: $scope.pageId,
						isEditorComment: $scope.state.showEditorComments,
						success: function(newCommentId) {
							$scope.showNewCommentLoading = false;
							$scope.newCommentId = newCommentId;
						},
					});
				});
			};

			// Called when the user is done editing the new comment
			$scope.newCommentDone = function(result) {
				$scope.newCommentId = undefined;
				if (!result.discard) {
					arb.pageService.newCommentCreated(result.pageId);
				}
			};

			// Track (globally) whether or not to show editor comments.
			if (!$scope.state.showEditorComments && $location.hash()) {
				// If hash points to a subpage for editors, show it
				var matches = (new RegExp('^subpage-' + aliasMatch + '$')).exec($location.hash());
				if (matches) {
					var page = arb.stateService.pageMap[matches[1]];
					if (page) {
						$scope.state.showEditorComments = page.isEditorComment;
					}
				}
			}

			$scope.toggleEditorComments = function() {
				$scope.state.showEditorComments = !$scope.state.showEditorComments;
			};

			// Compute how many visible comments there are.
			$scope.visibleCommentCount = function() {
				var count = 0;
				for (var n = 0; n < $scope.page.commentIds.length; n++) {
					var commentId = $scope.page.commentIds[n];
					var comment = arb.stateService.pageMap[commentId];
					if (comment.isResolved || comment.isDeleted) continue;
					count += $scope.shouldShowSubpage(commentId) ? 1 : 0;
				}
				return count;
			};

			// Compute whether the given subpage should be shown
			$scope.shouldShowSubpage = function(subpageId) {
				let subpage = arb.stateService.pageMap[subpageId];
				if ($scope.state.showEditorComments != subpage.isEditorComment) return false;
				if (!subpage.isVisibleApprovedComment()) return false;
				return true;
			};
		},
	};
});
