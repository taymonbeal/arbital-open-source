"use strict";

// Create new PageJsController.
// page - page object corresponding to the page being displayed.
var PageJsController = function(page, $topParent, pageService, userService) {
	var page = page;
	var $topParent = $topParent;
	var pageId = page.PageId; // id of the page being displayed
	var userId = userService.user.Id;

	// This map contains page data we fetched from the server, e.g. when hovering over a intrasite link.
	// TODO: use pageService instead
	var fetchedPagesMap = {}; // pageId -> page data
	
	// Send a new probability vote value to the server.
	var postNewVote = function(pageId, value) {
		var data = {
			pageId: pageId,
			value: value,
		};
		$.ajax({
			type: "POST",
			url: "/newVote/",
			data: JSON.stringify(data),
		})
		.done(function(r) {
		});
	}
	
	// Set up a new vote slider. Set the slider's value based on the user's vote.
	var createVoteSlider = function($parent, pageId, votes, isPopoverVote) {
		// Convert votes into a user id -> {value, createdAt} map
		var voteMap = {};
		if (page.Votes) {
			for(var i = 0; i < page.Votes.length; i++) {
				var vote = page.Votes[i];
				voteMap[vote.UserId] = {value: vote.Value, createdAt: vote.CreatedAt};
			}
		}

		// Copy vote-template and add it to the parent.
		var $voteDiv = $("#vote-template").clone().show().attr("id", "vote" + pageId).appendTo($parent);
		var $input = $voteDiv.find(".vote-slider-input");
		$input.attr("data-slider-id", $input.attr("data-slider-id") + pageId);
		var userVoteStr = userId in voteMap ? ("" + voteMap[userId].value) : "";
		var mySlider = $input.bootstrapSlider({
			step: 1,
			precision: 0,
			selection: "none",
			handle: "square",
			value: +userVoteStr,
			ticks: [1, 10, 20, 30, 40, 50, 60, 70, 80, 90, 99],
			formatter: function(s) { return s + "%"; },
		});
		var $tooltip = $parent.find(".tooltip-main");

		// Set the value of the user's vote.
		var setMyVoteValue = function($voteDiv, userVoteStr) {
			$voteDiv.attr("my-vote", userVoteStr);
			$voteDiv.find(".my-vote").toggle(userVoteStr !== "");
			$voteDiv.find(".my-vote-value").text("| my vote is \"" + (+userVoteStr) + "%\"");
		}
		setMyVoteValue($voteDiv, userVoteStr);
	
		// Setup vote bars.
		// A bar represents users' votes for a given value. The tiled background
		// allows us to display each vote separately.
		var bars = {}; // voteValue -> {bar: jquery bar element, users: array of user ids who voted on this value}
		// Stuff for setting up the bars' css.
		var $barBackground = $parent.find(".bar-background");
		var $sliderTrack = $parent.find(".slider-track");
		var originLeft = $sliderTrack.offset().left;
		var originTop = $sliderTrack.offset().top;
		var barWidth = Math.max(5, $sliderTrack.width() / (99 - 1) * 2);
		// Set the correct css for the given bar object given the number of votes it has.
		var setBarCss = function(bar) {
			var $bar = bar.bar;
			var voteCount = bar.users.length;
			$bar.toggle(voteCount > 0);
			$bar.css("height", 11 * voteCount);
			$bar.css("z-index", 2 + voteCount);
			$barBackground.css("height", Math.max($barBackground.height(), $bar.height()));
			$barBackground.css("top", 10);
		}
		var highlightBar = function($bar, highlight) {
			var css = "url(/static/images/vote-bar.png)";
			var highlightColor = "rgba(128, 128, 255, 0.3)";
			if(highlight) {
				css = "linear-gradient(" + highlightColor + "," + highlightColor + ")," + css;
			}
			$bar.css("background", css);
			$bar.css("background-size", "100% 11px"); // have to set this each time
		};
		// Get the bar object corresponding to the given vote value. Create a new one if there isn't one.
		var getBar = function(vote) {
			if (!(vote in bars)) {
				var x = (vote - 1) / (99 - 1);
				var $bar = $("<div class='vote-bar'></div>");
				$bar.css("left", x * $sliderTrack.width() - barWidth / 2);
				$bar.css("width", barWidth);
				$barBackground.append($bar);
				bars[vote] = {bar: $bar, users: []};
			}
			return bars[vote];
		}
		for(var id in voteMap){
			// Create stacks for all the votes.
			var bar = getBar(voteMap[id].value);
			bar.users.push(id);
			setBarCss(bar);
		}

		// Convert mouse X position into % vote value.
		var voteValueFromMousePosX = function(mouseX) {
			var x = (mouseX - $sliderTrack.offset().left) / $sliderTrack.width();
			x = Math.max(0, Math.min(1, x));
			return Math.round(x * (99 - 1) + 1);
		};

		// Update the label that shows how many votes have been cast.
		var updateVoteCount = function() {
			var votesLength = Object.keys(voteMap).length;
			$voteDiv.find(".vote-count").text(votesLength + " vote" + (votesLength == 1 ? "" : "s"));
		};
		updateVoteCount();

		// Set handle's width.
		var $handle = $parent.find(".min-slider-handle");
		$handle.css("width", barWidth);

		// Don't track mouse movements and such for the vote in a popover.
		if (isPopoverVote) {
			if (!(userId in voteMap)) {
				$handle.hide();
			}
			return;
		}

		// Catch mousemove event on the text, so that it doesn't propagate to parent
		// and spawn popovers, which will prevent us clicking on "x" button to delete
		// our vote.
		$parent.find(".text-center").on("mousemove", function(event){
			return false;
		});

		var mouseInParent = false;
		var mouseInPopover = false;
		// Destroy the popover that displayes who voted for a given value.
		var $usersPopover = undefined;
		var destroyUsersPopover = function() {
			if ($usersPopover !== undefined) {
				$usersPopover.popover("destroy");
				highlightBar($usersPopover, false);
			}
			$usersPopover = undefined;
			mouseInPopover = false;
		};

		// Track mouse movement to show voter names.
		$parent.on("mouseenter", function(event) {
			mouseInParent = true;
			$handle.show();
			$tooltip.css("opacity", 1.0);
		});
		$parent.on("mouseleave", function(event) {
			mouseInParent = false;
			if (!(userId in voteMap)) {
				$handle.hide();
			} else {
				$input.bootstrapSlider("setValue", voteMap[userId].value);
			}
			$tooltip.css("opacity", 0.0);
			if (!mouseInPopover) {
				destroyUsersPopover();
			}
		});
		$parent.trigger("mouseleave");
		$parent.on("mousemove", function(event) {
			// Update slider.
			var voteValue = voteValueFromMousePosX(event.pageX);
			$input.bootstrapSlider("setValue", voteValue);
			if (mouseInPopover) return true;

			// We do a funky search to see if there is a vote nearby, and if so, show popover.
			var offset = 0, maxOffset = 5;
			var offsetSign = -1;
			while(offset <= maxOffset) {
				var hoverVoteValue = voteValue + offsetSign * offset;
				if (!(hoverVoteValue in bars)) {
					if(offsetSign < 0) offset++;
					offsetSign = -offsetSign;
					continue;
				}
				var bar = bars[hoverVoteValue];
				// Don't do anything if it's still the same bar as last time.
				if (bar.bar === $usersPopover) {
					break;
				}
				// Destroy old one.
				destroyUsersPopover();
				// Create new popover.
				$usersPopover = bar.bar;
				highlightBar(bar.bar, true);
				$usersPopover.popover({
					html : true,
					placement: "bottom",
					trigger: "manual",
					title: "Voters (" + hoverVoteValue + "%)",
					content: function() {
						var html = "";
						for(var i = 0; i < bar.users.length; i++) {
							var userId = bar.users[i];
							var user = userService.userMap[userId];
							var name = user.firstName + "&nbsp;" + user.lastName;
							html += "<a href='" + userService.getUserUrl(userId) + "'>" + name + "</a> " +
								"<span class='gray-text'>(" + voteMap[userId].createdAt + ")</span><br>";
						}
						return html;
					}
				}).popover("show");
				var $popover = $barBackground.find(".popover");
				$popover.on("mouseenter", function(event){
					mouseInPopover = true;
				});
				$popover.on("mouseleave", function(event){
					mouseInPopover = false;
					if (!mouseInParent) {
						destroyUsersPopover();
					}
				});
				break;
			}
			if (offset > maxOffset) {
				// We didn't find a bar, so destroy any existing popover.
				destroyUsersPopover();
			}
		});
	
		// Handle user's request to delete their vote.
		$voteDiv.find(".delete-my-vote-link").on("click", function(event) {
			var bar = bars[voteMap[userId].value];
			bar.users.splice(bar.users.indexOf(userId), 1);
			setBarCss(bar);
			if (bar.users.length <= 0){
				delete bars[voteMap[userId].value];
			}

			mouseInPopover = false;
			mouseInParent = false;
			delete voteMap[userId];
			$parent.trigger("mouseleave");
			$parent.trigger("mouseenter");

			postNewVote(pageId, 0.0);
			setMyVoteValue($voteDiv, "");
			updateVoteCount();
			return false;
		});
		
		// Track click to see when the user wants to vote / update their vote.
		$parent.on("click", function(event) {
			if (mouseInPopover) return true;
			if (userId in voteMap && voteMap[userId].value in bars) {
				// Update old bar.
				var bar = bars[voteMap[userId].value];
				bar.users.splice(bar.users.indexOf(userId), 1);
				setBarCss(bar);
				destroyUsersPopover();
			}

			// Set new vote and update all the things.
			var vote = voteValueFromMousePosX(event.pageX); 
			voteMap[userId] = {value: vote, createdAt: "now"};
			postNewVote(pageId, vote);
			setMyVoteValue($voteDiv, "" + vote);
			updateVoteCount();

			// Update new bar.
			var bar = getBar(vote);
			bar.users.push(userId);
			setBarCss(bar);
		});
	}
	
	// Add a popover to the given element. The element has to be an intrasite link jquery object.
	var setupIntrasiteLink = function($element) {
		var $linkPopoverTemplate = $("#link-popover-template");
		$element.popover({ 
			html : true,
			placement: "bottom",
			trigger: "hover",
			delay: { "show": 500, "hide": 100 },
			title: function() {
				var pageId = $(this).attr("page-id");
				if (fetchedPagesMap[pageId]) {
					if (fetchedPagesMap[pageId].DeletedBy !== "0") {
						return "[DELETED]";
					}
					return fetchedPagesMap[pageId].Title;
				}
				return "Loading...";
			},
			content: function() {
				var $link = $(this);
				var pageId = $link.attr("page-id");
				// TODO: replace this custom ajax fetching with our "standard" angularjs pageService.
				// Check if we already have this page cached.
				var page = fetchedPagesMap[pageId];
				if (page) {
					if (page.DeletedBy !== "0") {
						$content.html("");
						return "";
					}
					var $content = $("<div>" + $linkPopoverTemplate.html() + "</div>");
					$content.find(".popover-summary").text(page.Summary);
					$content.find(".like-count").text(page.LikeCount);
					$content.find(".dislike-count").text(page.DislikeCount);
					var myLikeValue = +page.MyLikeValue;
					if (myLikeValue > 0) {
						$content.find(".disabled-like").addClass("on");
					} else if (myLikeValue < 0) {
						$content.find(".disabled-dislike").addClass("on");
					}
					if (page.HasVote) {
						setTimeout(function(){
							var $popover = $("#" + $link.attr("aria-describedby"));
							var $content = $popover.find(".popover-content");
							createVoteSlider($content.find(".vote"), page.PageId, page.Votes, true);
						}, 100);
					}
					return $content.html();
				}
				// Check if we already issued a request to fetch this page.
				if (page === undefined) {
					// Fetch page data from the server.
					fetchedPagesMap[pageId] = null;
					var data = {pageAlias: pageId, privacyKey: $link.attr("privacy-key")};
					$.ajax({
						type: "POST",
						url: "/pageInfo/",
						data: JSON.stringify(data),
					})
					.success(function(r) {
						var page = JSON.parse(r);
						if (!page) return;
						fetchedPagesMap[page.PageId] = page;
						if (page.Alias && page.Alias !== page.PageId) {
							// Store the alias as well.
							fetchedPagesMap[page.Alias] = page;
						}
						$link.popover("show");
					});
				}
				return '<img src="/static/images/loading.gif" class="loading-indicator" style="display:block"/>'
			}
		});
	}

	// Highlight the page div. Used for selecting answers when #anchor matches.
	var highlightPageDiv = function() {
		$(".hash-anchor").removeClass("hash-anchor");
		$topParent.find(".page-body-div").addClass("hash-anchor");
	};
	if (window.location.hash === "#page-" + pageId) {
		highlightPageDiv();
	}
	
	// === Setup handlers.
	
	// Inline comments
	// Create the inline comment highlight spans for the given paragraph.
	this.createInlineCommentHighlight = function(paragraphNode, start, end, nodeClass) {
		// How many characters we passed.
		var charCount = 0;
		// Store ranges we want to highlight.
		var ranges = [];
		// Compute context and text.
		recursivelyVisitChildren(paragraphNode, function(node, nodeText, needsEscaping) {
			if (nodeText === null) return false;
			var escapedText = needsEscaping ? escapeMarkdownChars(nodeText) : nodeText;
			var nodeWholeTextLength = node.wholeText ? node.wholeText.length : 0;
			var range = document.createRange();
			var nextCharCount = charCount + escapedText.length;
			if (charCount <= start && nextCharCount >= end) {
				range.setStart(node, start - charCount);
				range.setEnd(node, Math.min(nodeWholeTextLength, end - charCount));
				ranges.push(range);
			} else if (charCount <= start && nextCharCount > start) {
				range.setStart(node, start - charCount);
				range.setEnd(node, Math.min(nodeWholeTextLength, nodeText.length));
				ranges.push(range);
			} else if (start < charCount && nextCharCount >= end) {
				range.setStart(node, 0);
				range.setEnd(node, Math.min(nodeWholeTextLength, end - charCount));
				ranges.push(range);
			} else if (start < charCount && nextCharCount > start) {
				if (nodeWholeTextLength > 0) {
					range.setStart(node, 0);
					range.setEnd(node, Math.min(nodeWholeTextLength, nodeText.length));
				} else {
					range.selectNode(node);
				}
				ranges.push(range);
			} else if (start == charCount && charCount == nextCharCount) {
				// Rare occurence, but this captures MathJax divs/spans that
				// precede the script node where we actually get the text from.
				range.selectNode(node);
				ranges.push(range);
			}
			charCount = nextCharCount;
			return charCount >= end;
		});
		// Highlight ranges after we did DOM traversal, so that there are no
		// modifications during the traversal.
		for (var i = 0; i < ranges.length; i++) {
			highlightRange(ranges[i], nodeClass);
		}
		return ranges.length > 0 ? ranges[0].startContainer : null;
	};

	var $newInlineCommentDiv = $(".new-inline-comment-div");
	var $markdownText = $topParent.find(".markdown-text");
	$markdownText.on("mouseup", function(event) {
		// Do setTimeout, because otherwise there is a bug when you double click to
		// select a word/paragraph, then click again and the selection var is still
		// the same (not cleared).
		window.setTimeout(function(){
			var show = !!processSelectedParagraphText();
			$newInlineCommentDiv.toggle(show);
			if (show) {
				pageView.setNewInlineCommentPrimaryPageId(pageId);
			}
		}, 0);
	});

	// Deleting a page
	$topParent.find(".delete-page-link").on("click", function(event) {
		$("#delete-page-alert").show();
		return false;
	});
	$topParent.find(".delete-page-cancel").on("click", function(event) {
		$("#delete-page-alert").hide();
	});
	$topParent.find(".delete-page-confirm").on("click", function(event) {
		var data = {
			pageId: pageId,
		};
		$.ajax({
			type: "POST",
			url: "/deletePage/",
			data: JSON.stringify(data),
		})
		.done(function(r) {
			smartPageReload();
		});
		return false;
	});
	
	// Page voting stuff.
	// likeClick is 1 is user clicked like and -1 if they clicked dislike.
	var processLike = function(likeClick, event) {
		var $target = $(event.target);
		var $like = $target.closest(".page-like-div");
		var $likeCount = $like.find(".like-count");
		var $dislikeCount = $like.find(".dislike-count");
		var currentLikeValue = +$like.attr("my-like");
		var newLikeValue = (likeClick === currentLikeValue) ? 0 : likeClick;
		var likes = +$likeCount.text();
		var dislikes = +$dislikeCount.text();
	
		// Update like counts.
		// This logic has been created by looking at truth tables.
		if (currentLikeValue === 1) {
			likes -= 1;
		} else if (likeClick === 1) {
			likes += 1;
		}
		if (currentLikeValue === -1) {
			dislikes -= 1;
		} else if (likeClick === -1) {
			dislikes += 1;
		}
		$likeCount.text("" + likes);
		$dislikeCount.text("" + dislikes);
	
		// Update my-like
		$like.attr("my-like", "" + newLikeValue);
	
		// Display my like
		$like.find(".like-link").toggleClass("on", newLikeValue === 1);
		$like.find(".dislike-link").toggleClass("on", newLikeValue === -1);
		
		// Notify the server
		var data = {
			pageId: pageId,
			value: newLikeValue,
		};
		$.ajax({
			type: "POST",
			url: '/newLike/',
			data: JSON.stringify(data),
		})
		.done(function(r) {
		});
		return false;
	}
	$topParent.find(".like-link").on("click", function(event) {
		return processLike(1, event);
	});
	$topParent.find(".dislike-link").on("click", function(event) {
		return processLike(-1, event);
	});
	
	// Subscription stuff.
	$topParent.find(".subscribe-to-page-link").on("click", function(event) {
		var $target = $(event.target);
		$target.toggleClass("on");
		var data = {
			pageId: pageId,
		};
		$.ajax({
			type: "POST",
			url: $target.hasClass("on") ? "/newSubscription/" : "/deleteSubscription/",
			data: JSON.stringify(data),
		})
		.done(function(r) {
		});
		return false;
	});

	// Start initializes things that have to be killed when this editPage stops existing.
	this.start = function(pageVotes) {
		// Set up markdown.
		zndMarkdown.init(false, pageId, page.Text, $topParent);

		// Intrasite link hover.
		setupIntrasiteLink($topParent.find(".intrasite-link"));

		// Setup probability vote slider.
		if (page.HasVote) {
			createVoteSlider($topParent.find(".page-vote"), pageId, page.Votes, false);
		}
	};

	// Called before this controller is destroyed.
	this.stop = function() {
	};
};

// Directive for showing a standard Zanaduu page.
app.directive("zndPage", function (pageService, userService, $compile, $timeout) {
	return {
		templateUrl: "/static/html/page.html",
		controller: function ($scope, pageService, userService) {
			$scope.userService = userService;
			$scope.page = pageService.pageMap[$scope.pageId];
		},
		scope: {
			pageId: "@",
		},
		link: function(scope, element, attrs) {

			// Set up Page JS Controller.
			$timeout(function(){
				scope.pageJsController = new PageJsController(scope.page, element, pageService, userService);
				scope.pageJsController.start();

				if (scope.page.CommentIds != null) {
					// Process comments in two passes. First normal comments.
					processComments(false);
					$timeout(function() {
						// Inline comments after a delay long enough for MathJax to have been processed.
						processComments(true);
					}, 3000);
				}
			});

			// Track toggle-inline-comment offsets, so we can prevent overlap.
			var inlineCommentOffsets = [];
			var fixInlineCommentOffset = function(offset) {
				for (var i = 0; i < inlineCommentOffsets.length; i++) {
					var o = inlineCommentOffsets[i];
					if (Math.abs(offset.top - o.top) < 25) {
						if (Math.abs(offset.left - o.left) < 30) {
							offset.left = o.left + 35;
						}
					}
				}
				inlineCommentOffsets.push(offset);
			}

			// Create a toggle-inline-comment-div.
			var createNewInlineCommentToggle = function(pageId, paragraphNode, anchorOffset, anchorLength) {
				var highlightClass = "inline-comment-" + pageId;
				var $commentDiv = $(".toggle-inline-comment-div.template").clone();
				$commentDiv.attr("id", "comment-" + pageId).removeClass("template");
				var commentCount = pageService.pageMap[pageId].Children.length + 1;
				$commentDiv.find(".inline-comment-count").text("" + commentCount);
				$(".question-div").append($commentDiv);

				// Process mouse events.
				var $commentIcon = $commentDiv.find(".inline-comment-icon");
				$commentIcon.on("mouseenter", function(event) {
					$("." + highlightClass).addClass("inline-comment-highlight");
				});
				$commentIcon.on("mouseleave", function(event) {
					if ($commentIcon.hasClass("on")) return true;
					$("." + highlightClass).removeClass("inline-comment-highlight");
				});
				$commentIcon.on("click", function(event) {
					pageView.toggleInlineComment($commentDiv, function() {
						$("." + highlightClass).addClass("inline-comment-highlight");
						var $comment = $compile("<znd-comment primary-page-id='" + scope.page.PageId +
								"' page-id='" + pageId + "'></znd-comment>")(scope);
						$(".inline-comment-div").append($comment);
					});
					return false;
				});

				var commentIconLeft = $(".question-div").offset().left + 10;
				var anchorNode = scope.pageJsController.createInlineCommentHighlight(paragraphNode, anchorOffset, anchorOffset + anchorLength, highlightClass);
				if (anchorNode) {
					if (anchorNode.nodeType != Node.ELEMENT_NODE) {
						anchorNode = anchorNode.parentElement;
					}
					var offset = {left: commentIconLeft, top: $(anchorNode).offset().top};
					fixInlineCommentOffset(offset);
					$commentDiv.offset(offset);
					if (window.location.hash === "#comment-" + pageId) {
						$commentIcon.trigger("click");
						$("html, body").animate({
			        scrollTop: $(anchorNode).offset().top - 100
				    }, 1000);
					}
				} else {
					$commentDiv.hide();
					console.log("ERROR: couldn't find anchor node for inline comment");
				}
			}

			// Dynamically create comment elements.
			var processComments = function(allowInline) {
				var $comments = element.find(".comments");
				var $markdown = element.find(".markdown-text");
				var dmp = new diff_match_patch();
				dmp.Match_MaxBits = 10000;
				dmp.Match_Distance = 10000;

				// If we have inline comments, we'll need to compute the raw text for
				// each paragraph.
				var paragraphTexts = undefined;
				var populateParagraphTexts = function() {
					paragraphTexts = [];
					var i = 0;
					$markdown.children().each(function() {
						paragraphTexts.push(getParagraphText($(this).get(0)).context);
						i++;
					});
				};

				// Go through comments in chronological order.
				scope.page.CommentIds.sort(pageService.getChildSortFunc({SortChildrenBy: "chronological", Type: "comment"}));
				for (var n = 0; n < scope.page.CommentIds.length; n++) {
					var comment = pageService.pageMap[scope.page.CommentIds[n]];
					// Check if the comment in anchored and we can still find the paragraph.
					if (comment.AnchorContext && comment.AnchorText) {
						if (!allowInline) continue;
						// Find the best paragraph.
						var bestParagraphNode, bestParagraphText, bestScore = Number.MAX_SAFE_INTEGER;
						if (!paragraphTexts) {
							populateParagraphTexts();
						}
						for (var i = 0; i < paragraphTexts.length; i++) {
							var text = paragraphTexts[i];
							var diffs = dmp.diff_main(text, comment.AnchorContext);
							var score = dmp.diff_levenshtein(diffs);
							if (score < bestScore) {
								bestParagraphNode = $markdown.children().get(i);
								bestParagraphText = text;
								bestScore = score;
							}
						}
						if (bestScore > comment.AnchorContext.length / 2) {
							// This is not a good paragraph match. Continue processing as a normal comment.
							comment.Text = "> " + comment.AnchorText + "\n\n" + comment.Text;
						} else {
							// Find offset into the best paragraph.
							var anchorLength;
							var anchorOffset = dmp.match_main(bestParagraphText, comment.AnchorText, comment.AnchorOffset);
							if (anchorOffset < 0) {
								// Couldn't find a match within the paragraph. We'll just use paragraph as the anchor.
								anchorOffset = 0;
								anchorLength = bestParagraphText.length;
							} else {
								// Figure out how long the highlighted anchor should be.
								var remainingText = bestParagraphText.substring(anchorOffset);
								var diffs = dmp.diff_main(remainingText, comment.AnchorText);
								anchorLength = remainingText.length;
								if (diffs.length > 0) {
									// Note: we can potentially be more clever here and discount
									// edits done after anchorText.length chars higher.
									var lastDiff = diffs[diffs.length - 1];
									if (lastDiff[0] < 0) {
										anchorLength -= lastDiff[1].length;
									}
								}
							}
							createNewInlineCommentToggle(comment.PageId, bestParagraphNode, anchorOffset, anchorLength);
							continue;
						}
					}
					// Make sure this comment is not a reply (i.e. it has a parent comment)
					// If it's a reply, add it as a child to the corresponding parent comment.
					if (comment.Parents != null) {
						var hasParentComment = false;
						for (var i = 0; i < comment.Parents.length; i++) {
							var parent = pageService.pageMap[comment.Parents[i].ParentId];
							hasParentComment = parent.Type === "comment";
							if (hasParentComment) {
								if (parent.Children == null) parent.Children = [];
								parent.Children.push({ParentId: parent.PageId, ChildId: comment.PageId});
								break;
							}
						}
						if (hasParentComment) continue;
					}
					var $comment = $compile("<znd-comment primary-page-id='" + scope.pageId +
							"' page-id='" + comment.PageId + "'></znd-comment>")(scope);
					$comments.prepend($comment);
				}
			};
		},
	};
});