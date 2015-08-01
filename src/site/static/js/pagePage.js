// pageView controls various portions of the page like the navigation and RHS columns.
var pageView;
$(function() {
	pageView = new function() {
		var showingNavigation = true;
		var showingRhs = false;
		var $navigation = $(".navigation-column");
		var $questionDiv = $(".question-div");
		var $rhsButtonsDiv = $(".rhs-buttons-div");
		var $newInlineCommentDiv = $(".new-inline-comment-div");
		var $commentButton = $newInlineCommentDiv.find(".new-inline-comment-button");

		// Delete any expanded inline comments or inline comment editors.
		this.clearRhs = function() {
			$questionDiv.find("znd-edit-page").remove();
			$questionDiv.find("znd-comment").remove();
			$(".inline-comment-icon").removeClass("on");
		};
	
		// Show right hand side and call the callback after the animation has played.
		this.showRhs = function(callback) {
			if (showingRhs) {
				callback();
				return;
			}
			$questionDiv.find(".toggle-inline-comment-div").animate({"left": "-=30%"});
			$rhsButtonsDiv.hide();
			$questionDiv.animate({"width": "30%"}, {queue: false});
			$navigation.animate({"margin-left": "-30%"}, {queue: false, complete: function() {
				showingRhs = true;
				if (callback) callback();
			}});
		};

		// Hide RHS.
		this.hideRhs = function(callback) {
			if (!showingRhs) {
				callback();
				return;
			}
			$questionDiv.animate({"width": "14%"}, {queue: false});
			$questionDiv.find(".toggle-inline-comment-div").animate({"left": "+=30%"});
			$navigation.animate({"margin-left": "0%"}, {queue: false, complete: function() {
				showingRhs = false;
				$rhsButtonsDiv.show();
				$(".inline-comment-highlight").removeClass("inline-comment-highlight");
				if (callback) callback();
			}});
		};

		// Hide/show an inline comment.
		this.toggleInlineComment = function($toggleDiv, callback) {
			var $inlineComment = $toggleDiv.find(".inline-comment-icon");
			if ($inlineComment.hasClass("on")) {
				this.clearRhs();
				this.hideRhs();
			} else {
				this.clearRhs();
				$(".inline-comment-highlight").removeClass("inline-comment-highlight");
				this.showRhs(function() {
					var offset = {left: $questionDiv.offset().left + 32, top: $toggleDiv.offset().top + 40};
					$(".inline-comment-div").offset(offset);
					$inlineComment.addClass("on");
					callback();
				});
			}
		};

		// Show the edit inline comment box.
		this.showEditInlineComment = function($scope, selection) {
			this.clearRhs();
			$(".toggle-inline-comment-div").hide();
			this.showRhs(function() {
				var $newInlineCommentDiv = $(".new-inline-comment-div");
				var offset = {left: $questionDiv.offset().left + 30, top: $(".inline-comment-highlight").offset().top};
				$(".inline-comment-div").offset(offset);
				createEditCommentDiv($(".inline-comment-div"), $newInlineCommentDiv, $scope, {
					anchorContext: selection.context,
					anchorText: selection.text,
					anchorOffset: selection.offset,
					primaryPageId: newInlineCommentPrimaryPageId,
					callback: function() {
						pageView.clearRhs();
						pageView.hideRhs(function() {
							$(".toggle-inline-comment-div").hide();
						});
					},
				});
			});
		};

		// Store the primary page id used for creating a new inline comment.
		var newInlineCommentPrimaryPageId;
		this.setNewInlineCommentPrimaryPageId = function(id) {
			newInlineCommentPrimaryPageId = id;
		};
	}();
});

// MainCtrl is for the Page page.
app.controller("MainCtrl", function($scope, $compile, pageService, userService) {
	$scope.pageService = pageService;
	$scope.$compile = $compile;
	$scope.relatedIds = gRelatedIds;
	$scope.questionIds = [];
	$scope.answerIds = [];
	$scope.page = pageService.primaryPage;

	// Set up children pages and question ids.
	$scope.initialChildren = {};
	$scope.initialChildrenCount = 0;
	for (var n = 0; n < $scope.page.Children.length; n++) {
		var id = $scope.page.Children[n].ChildId;
		var page = pageService.pageMap[id];
		if (page.Type === "question") {
			$scope.questionIds.push(id);
		} else if (page.Type === "answer") {
			$scope.answerIds.push(id);
		} else {
			$scope.initialChildren[id] = page;
			$scope.initialChildrenCount++;
		}
	}

	// Sort question ids by likes, but put the ones created by current user first.
	$scope.questionIds.sort(function(id1, id2) {
		var page1 = pageService.pageMap[id1];
		var page2 = pageService.pageMap[id2];
		var ownerDiff = (page2.CreatorId == userService.user.Id ? 1 : 0) -
				(page1.CreatorId == userService.user.Id ? 1 : 0);
		if (ownerDiff != 0) return ownerDiff;
		return page2.LikeScore - page1.LikeScore;
	});

	// Set up parents pages.
	$scope.initialParents = {};
	$scope.initialParentsCount = $scope.page.Parents.length;
	for (var n = 0; n < $scope.initialParentsCount; n++) {
		var id = $scope.page.Parents[n].ParentId;
		$scope.initialParents[id] = pageService.pageMap[id];
	}

	// Question button stuff.
	keepDivFixed($(".rhs-buttons-div"));

	// Process question button click.
	$(".question-button").on("click", function(event) {
		$(document).trigger("new-page-modal-event", {
			modalKey: "newQuestion",
			parentPageId: $scope.page.PageId,
			callback: function(result) {
				if (result.abandon) {
					$scope.$apply(function() {
						$scope.page.ChildDraftId = 0;
					});
				} else if (result.hidden) {
					$scope.$apply(function() {
						$scope.page.ChildDraftId = result.alias;
					});
				} else {
					smartPageReload();
				}
			},
		});
	});

	// Inline comment button stuff.
	var $newInlineCommentDiv = $(".new-inline-comment-div");
	var $commentButton = $newInlineCommentDiv.find(".new-inline-comment-button");
	// Process new inline comment button click.
	$commentButton.on("click", function(event) {
		$(".inline-comment-highlight").removeClass("inline-comment-highlight");
		var selection = getSelectedParagraphText();
		if (selection) {
			pageView.showEditInlineComment($scope, selection);
		}
		return false;
	});

	// Add answers pages.
	var $answersList = $(".answers-list");
	for (var n = 0; n < $scope.answerIds.length; n++){
		var el = $compile("<znd-page page-id='" + $scope.answerIds[n] + "'></znd-page><hr></hr>")($scope);
		$answersList.append(el);
	}

	// Add edit page for the answer.
	if ($scope.page.Type === "question") {
		$scope.answerDoneFn = function(result) {
			if (result.abandon) {
				getNewAnswerId();
			} else if (result.alias) {
				window.location.assign($scope.page.Url + "#page-" + result.alias);
				window.location.reload();
			}
		};

		var createAnswerEditPage = function(newPageId) {
			var page = pageService.pageMap[newPageId];
			page.Type = "answer";
			page.Parents = [{ParentId: $scope.page.PageId, ChildId: newPageId}];
			var el = $compile("<znd-edit-page page-id='" + newPageId +
				"' primary-page-id='" + $scope.page.PageId +
				"' done-fn='answerDoneFn(result)'></znd-edit-page>")($scope);
			$(".new-answer").append(el);
		};
		var getNewAnswerId = function() {
			$(".new-answer").find("znd-edit-page").remove();
			pageService.loadPages([],
				function(data, status) {
					createAnswerEditPage(Object.keys(data)[0]);
				}, function(data, status) {
					console.log("Couldn't load pages: " + loadPagesIds);
				}
			);
		};
		if ($scope.page.ChildDraftId > 0) {
			createAnswerEditPage($scope.page.ChildDraftId);
		} else {
			getNewAnswerId();
		}
	}
});