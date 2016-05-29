'use strict';

var notEscaped = '(^|\\\\`|\\\\\\[|(?:[^A-Za-z0-9_`[\\\\]|\\\\\\\\))';
var noParen = '(?=$|[^(])';
var nakedAliasMatch = '[\\-\\+]?[A-Za-z0-9_]+\\.?[A-Za-z0-9_]*';
var aliasMatch = '(' + nakedAliasMatch + ')';
var anyUrlMatch = /\b((?:[a-z][\w-]+:(?:\/{1,3}|[a-z0-9%])|www\d{0,3}[.]|[a-z0-9.\-]+[.][a-z]{2,4}\/)(?:[^\s()<>]+|\(([^\s()<>]+|(\([^\s()<>]+\)))*\))+(?:\(([^\s()<>]+|(\([^\s()<>]+\)))*\)|[^\s`!()\[\]{};:'".,<>?«»“”‘’]))/i;

// [vote: alias]
var voteEmbedRegexp = new RegExp(notEscaped +
		'\\[vote: ?' + aliasMatch + '\\]' + noParen, 'g');
// [alias/url text]
var forwardLinkRegexp = new RegExp(notEscaped +
		'\\[([^ \\]]+?) (?![^\\]]*?\\\\\\])([^\\]]+?)\\]' + noParen, 'g');
// [alias]
var simpleLinkRegexp = new RegExp(notEscaped +
		'\\[' + aliasMatch + '\\]' + noParen, 'g');
// [text](alias)
var complexLinkRegexp = new RegExp(notEscaped +
		'\\[([^\\]]+?)\\]' + // match [Text]
		'\\(' + aliasMatch + '\\)', 'g'); // match (Alias)
// [@alias]
var atAliasRegexp = new RegExp(notEscaped +
		'\\[@' + aliasMatch + '\\]' + noParen, 'g');

// markdownService provides a constructor you can use to create a markdown converter,
// either for converting markdown to text or editing.
app.service('markdownService', function($compile, $timeout, pageService, userService) {
	// Store an array of page aliases that failed to load, so that we don't keep trying to reload them
	var failedPageAliases = {};

	// Pass in a pageId to create an editor for that page
	var createConverter = function(isEditor, pageId, postConversionCallback) {
		// NOTE: not using $location, because we need port number
		var host = window.location.host;
		var converter = Markdown.getSanitizingConverter();

		// Process $$$mathjax$$$ blocks.
		if (isEditor) {
			var mathjaxBlockRegexp = new RegExp('^(~D~D~D[\\s\\S]+?~D~D~D) *(?=\Z|\n)', 'gm');
			converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
				return text.replace(mathjaxBlockRegexp, function(whole, mathjaxText) {
					var encodedText = encodeURIComponent(mathjaxText);
					var key = '$$$' + encodedText.substring(6, encodedText.length - 6) + '$$$';
					var cachedValue = pageService.getMathjaxCacheValue(key);
					var style = cachedValue ? ('style=\'' + cachedValue.style + '\' ') : '';
					return '<div ' + style + 'class=\'MathJax_Display\' arb-math-compiler=\'' + encodedText + '\'></div>';
				});
			});
		}

		// Process [summary(optional):markdown] blocks.
		var summaryBlockRegexp = new RegExp('^\\[summary(\\([^)\n\r]+\\))?: ?([\\s\\S]+?)\\] *(?=\Z|\n\Z|\n\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(summaryBlockRegexp, function(whole, summaryName, summary) {
				if (isEditor) {
					return runBlockGamut('---\n\n**Summary' + (summaryName || '') + ':** ' + summary + '\n\n---');
				} else {
					return runBlockGamut('');
				}
			});
		});

		// Process %knows-requisite([alias]):markdown% blocks.
		var hasReqBlockRegexp = new RegExp('^(%+)(!?)knows-requisite\\(\\[' + aliasMatch + '\\]\\): ?([\\s\\S]+?)\\1 *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(hasReqBlockRegexp, function(whole, bars, not, alias, markdown) {
				var pageId = (alias in pageService.pageMap) ? pageService.pageMap[alias].pageId : alias;
				var div = '<div ng-show=\'' + (not ? '!' : '') + 'pageService.hasMastery("' + pageId + '")\'>';
				if (isEditor) {
					div = '<div class=\'conditional-text editor-block\'>';
				}
				return div + runBlockGamut(markdown) + '\n\n</div>';
			});
		});

		// Process %wants-requisite([alias]):markdown% blocks.
		var wantsReqBlockRegexp = new RegExp('^(%+)(!?)wants-requisite\\(\\[' + aliasMatch + '\\]\\): ?([\\s\\S]+?)\\1 *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(wantsReqBlockRegexp, function(whole, bars, not, alias, markdown) {
				var pageId = (alias in pageService.pageMap) ? pageService.pageMap[alias].pageId : alias;
				var div = '<div ng-show=\'' + (not ? '!' : '') + 'pageService.wantsMastery("' + pageId + '")\'>';
				if (isEditor) {
					div = '<div class=\'conditional-text editor-block\'>';
				}
				return div + runBlockGamut(markdown) + '\n\n</div>';
			});
		});

		// Process %todo:markdown% blocks.
		var todoBlockRegexp = new RegExp('^(%+)todo: ?([\\s\\S]+?)\\1 *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(todoBlockRegexp, function(whole, bars, markdown) {
				if (isEditor) {
					return '<div class=\'todo-text editor-block\'>' + runBlockGamut(markdown) + '\n\n</div>';
				}
				return '';
			});
		});

		// Process %comment:markdown% blocks.
		var commentBlockRegexp = new RegExp('^(%+)comment: ?([\\s\\S]+?)\\1 *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(commentBlockRegexp, function(whole, bars, markdown) {
				if (isEditor) {
					return '<div class=\'info-text editor-block\'>' + runBlockGamut(markdown) + '\n\n</div>';
				}
				return '';
			});
		});

		// Process %hidden: text% blocks.
		var hiddenBlockRegexp = new RegExp('^(%+)hidden\\(([\\s\\S]+?)\\): ?([\\s\\S]+?)\\1 *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(hiddenBlockRegexp, function(whole, bars, buttonText, text) {
				var blockText = text + '\n\n';
				return '<div arb-hidden-text button-text=\'' + buttonText + '\'>' + runBlockGamut(blockText) + '\n\n</div>';
			});
		});

		// Process [multiple-choice(objectAlias): text
		// a: text
		// knows: [alias1],[alias2]...
		// wants: [alias1],[alias2]...
		// ] blocks.
		var mcBlockRegexp = new RegExp('^\\[multiple-choice\\(' + aliasMatch + '\\): ?([^\n]+?)\n' +
				'(a: ?[^\n]+?\n)' + // choice, e.g. "a: Carrots"
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'(-knows: ?[^\n]+?\n)?' +
				'(-wants: ?[^\n]+?\n)?' +
				'(b: ?[^\n]+?\n)' + // choice, e.g. "b: Carrots"
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'(-knows: ?[^\n]+?\n)?' +
				'(-wants: ?[^\n]+?\n)?' +
				'(c: ?[^\n]+?\n)?' + // choice, e.g. "c: Carrots"
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'(-knows: ?[^\n]+?\n)?' +
				'(-wants: ?[^\n]+?\n)?' +
				'(d: ?[^\n]+?\n)?' + // choice, e.g. "d: Carrots"
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'(-knows: ?[^\n]+?\n)?' +
				'(-wants: ?[^\n]+?\n)?' +
				'(e: ?[^\n]+?\n)?' + // choice, e.g. "e: Carrots"
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'(-knows: ?[^\n]+?\n)?' +
				'(-wants: ?[^\n]+?\n)?' +
				'\\] *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(mcBlockRegexp, function() {
				var result = [];
				// Process captured groups
				for (var n = 2; n < arguments.length; n++) {
					var arg = arguments[n];
					if (+arg) break; // there are extra arguments that we don't need, starting with some number
					if (!arg) continue;
					if (n == 2) { // question text
						result.push(arg + '\n\n');
					} else {
						// Match answer line
						var match = arg.match(/^([a-e]): ?([\s\S]+?)\n$/);
						if (match) {
							result.push('- ' + match[2] + '\n');
							continue;
						}
						result.push(' - ' + arg);
					}
				}
				return '<arb-multiple-choice page-id=\'' + pageId + '\' object-alias=\'' + arguments[1] + '\'>' +
					runBlockGamut(result.join('')) + '\n\n</arb-multiple-choice>';
			});
		});

		// Process [checkbox: text
		// knows: [alias1],[alias2]...
		// wants: [alias1],[alias2]...
		// ] blocks.
		var checkboxBlockRegexp = new RegExp('^\\[checkbox: ?([^\n]+?)\n' +
				'(knows: ?[^\n]+?\n)?' +
				'(wants: ?[^\n]+?\n)?' +
				'\\] *(?=\Z|\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(checkboxBlockRegexp, function(whole, text, knows, wants) {
				var blockText = text + '\n\n' + (knows ? '- ' + knows + '\n' : '') + (wants ? '- ' + wants : '');
				return '<arb-checkbox>' + runBlockGamut(blockText) + '\n\n</arb-checkbox>';
			});
		});

		// Process [toc:] block.
		var tocBlockRegexp = new RegExp('^\\[toc:\\] *(?=\Z|\n\Z|\n\n)', 'gm');
		converter.hooks.chain('preBlockGamut', function(text, runBlockGamut) {
			return text.replace(tocBlockRegexp, function(whole, text, knows, wants) {
				return '<arb-table-of-contents page-id=\'' + pageId + '\'></arb-table-of-contents>';
			});
		});

		// Process $$mathjax$$ spans.
		if (isEditor) {
			var mathjaxSpanRegexp = new RegExp(notEscaped + '(~D~D[\\s\\S]+?~D~D)', 'g');
			converter.hooks.chain('preSpanGamut', function(text) {
				return text.replace(mathjaxSpanRegexp, function(whole, prefix, mathjaxText) {
					return prefix + '<span arb-math-compiler=\'' + encodeURIComponent(mathjaxText) + '\'></span>';
				});
			});
		}

		// Process %knows-requisite([alias]): markdown% spans.
		var hasReqSpanRegexp = new RegExp(notEscaped + '(%+)(!?)knows-requisite\\(\\[' + aliasMatch + '\\]\\): ?([\\s\\S]+?)\\2', 'g');
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(hasReqSpanRegexp, function(whole, prefix, bars, not, alias, markdown) {
				var pageId = (alias in pageService.pageMap) ? pageService.pageMap[alias].pageId : alias;
				var span = '<span ng-show=\'' + (not ? '!' : '') + 'pageService.hasMastery("' + pageId + '")\'>';
				if (isEditor) {
					span = '<span class=\'conditional-text\'>';
				}
				return prefix + span  + markdown + '</span>';
			});
		});

		// Process %wants-requisite([alias]): markdown% spans.
		var wantsReqSpanRegexp = new RegExp(notEscaped + '(%+)(!?)wants-requisite\\(\\[' + aliasMatch + '\\]\\): ?([\\s\\S]+?)\\2', 'g');
		converter.hooks.chain('preSpanGamut', function(text, run) {
			return text.replace(wantsReqSpanRegexp, function(whole, prefix, bars, not, alias, markdown) {
				var pageId = (alias in pageService.pageMap) ? pageService.pageMap[alias].pageId : alias;
				var span = '<span ng-show=\'' + (not ? '!' : '') + 'pageService.wantsMastery("' + pageId + '")\'>';
				if (isEditor) {
					span = '<span class=\'conditional-text\'>';
				}
				return prefix + span + markdown + '</span>';
			});
		});

		// Process [todo:text] spans.
		var todoSpanRegexp = new RegExp(notEscaped +
				'\\[todo: ?([^\\]]+?)\\]' + noParen, 'g');
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(todoSpanRegexp, function(whole, prefix, text) {
				if (isEditor) {
					return prefix + '<span class=\'todo-text\'>';
				}
				return prefix;
			});
		});

		// Process [comment:text] spans.
		var commentSpanRegexp = new RegExp(notEscaped +
				'\\[comment: ?([^\\]]+?)\\]' + noParen, 'g');
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(commentSpanRegexp, function(whole, prefix, text) {
				if (isEditor) {
					return prefix + '<span class=\'info-text\'>';
				}
				return prefix;
			});
		});

		// Process [vote:alias] spans.
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(voteEmbedRegexp, function(whole, prefix, alias) {
				return prefix + '[Embedded ' + alias + ' vote. ](' + pageService.getPageUrl(alias, {includeHost: true}) + '/?embedVote=1)';
			});
		});

		// Convert [ text] spans into links.
		var spaceTextRegexp = new RegExp(notEscaped +
				'\\[ ([^\\]]+?)\\]' + noParen, 'g');
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(spaceTextRegexp, function(whole, prefix, text) {
				var editUrl = pageService.getNewPageUrl({includeHost: true});
				return prefix + '[' + text + '](' + editUrl + ')';
			});
		});

		// Convert [alias/url text] spans into links.
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(forwardLinkRegexp, function(whole, prefix, alias, text) {
				var matches = alias.match(anyUrlMatch);
				if (matches) {
					var url = matches[0];
					return prefix + '[' + text + '](' + url + ')';
				}
				matches = alias.match(aliasMatch);
				if (matches && matches[0] == alias) {
					var page = pageService.pageMap[alias];
					if (page) {
						var url = pageService.getPageUrl(page.pageId, {includeHost: true});
						return prefix + '[' + text + '](' + url + ')';
					} else {
						var url = pageService.getPageUrl(alias, {includeHost: true});
						return prefix + '[' + text + '](' + url + ')';
					}
				} else {
					return whole;
				}
			});
		});

		// Convert [alias] spans into links.
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(simpleLinkRegexp, function(whole, prefix, alias) {
				var firstAliasChar = alias.substring(0, 1);
				var trimmedAlias = alias;
				if (firstAliasChar == '-' || firstAliasChar == '+') {
					trimmedAlias = alias.substring(1);
				}
				var page = pageService.pageMap[trimmedAlias];
				if (page) {
					var url = pageService.getPageUrl(page.pageId, {includeHost: true});
					// Match the page title's case to the alias's case
					var casedTitle;
					if (firstAliasChar == '+') {
						casedTitle = page.title.substring(0, 1).toUpperCase() + page.title.substring(1);
					} else {
						casedTitle = page.title.substring(0, 1).toLowerCase() + page.title.substring(1);
					}
					return prefix + '[' + casedTitle + '](' + url + ')';
				} else {
					var url = pageService.getPageUrl(trimmedAlias, {includeHost: true});
					return prefix + '[' + trimmedAlias + '](' + url + ')';
				}
			});
		});

		// Convert [@alias] spans into links.
		converter.hooks.chain('preSpanGamut', function(text) {
			return text.replace(atAliasRegexp, function(whole, prefix, alias) {
				var page = pageService.pageMap[alias];
				if (page) {
					var url = pageService.getUserUrl(page.pageId, {includeHost: true});
					return prefix + '[' + page.title + '](' + url + ')';
				} else {
					var url = pageService.getUserUrl(alias, {includeHost: true});
					return prefix + '[' + alias + '](' + url + ')';
				}
			});
		});

		if (isEditor) {
			// Setup the editor stuff.
			var editor = new Markdown.Editor(converter, pageId);
			if (!userService.user.ignoreMathjax) {
				InitMathjax(converter, editor, pageId);
			}
			converter.hooks.chain('postConversion', function(text) {
				if (postConversionCallback) {
					postConversionCallback(function() {
						editor.refreshPreview();
					});
				}
				return text;
			});

			editor.run();
		} else {
			InitMathjax(converter);
		}

		return converter;
	};

	// Process all the links in the give element.
	// If refreshFunc is set, fetch page's we don't have yet, and call that function
	// when one of them is loaded.
	this.processLinks = function(scope, $pageText, refreshFunc) {
		// Setup attributes for page links that are within our domain.
		// NOTE: not using $location, because we need port number
		var pageRe = new RegExp('^(?:https?:\/\/)?(?:www\.)?' + // match http and www stuff
			getHostMatchRegex(window.location.host) + // match the url host part
			'\/(?:p|edit)\/' + aliasMatch + '?' + // [1] capture page alias
			'\/?' + // optional ending /
			'(.*)'); // optional other stuff

		// Setup attributes for user links that are within our domain.
		var userRe = new RegExp('^(?:https?:\/\/)?(?:www\.)?' + // match http and www stuff
			window.location.host + // match the url host part
			'\/user\/' + aliasMatch + // [1] capture user alias
			'(\/' + nakedAliasMatch + ')?' + // [2] optional alias
			'\/?' + // optional ending /
			'(.*)'); // optional other stuff

		var processPageLink = function($element, href, pageAlias, searchParams) {
			if (href == $element.text()) {
				// This is a normal link and we should leave it as such
				return;
			}
			pageAlias = pageAlias || '';
			// Check if we are embedding a vote
			if (searchParams.indexOf('embedVote') > 0) {
				$element.attr('embed-vote-id', pageAlias).addClass('red-link');
			} else if (pageAlias && pageAlias in pageService.pageMap) {
				$element.addClass('intrasite-link').attr('page-id', pageAlias);
				$element.attr('page-id', pageService.pageMap[pageAlias].pageId);
				if (pageService.pageMap[pageAlias].isDeleted) {
					// Link to a deleted page.
					$element.addClass('red-link');
				} else {
					// Normal healthy link!
				}
			} else {
				// Mark as red link
				$element.addClass('intrasite-link red-link').attr('page-id', pageAlias);
				$element.attr('href', $element.attr('href').replace(/\/p\//, '/edit/'));
				if (refreshFunc && pageAlias === '') {
					$element.addClass('red-todo-text');
				}
				if (refreshFunc && !(pageAlias in failedPageAliases)) {
					// Try to load the page
					pageService.loadTitle(pageAlias, {
						silentFail: true,
						success: function() {
							if (pageAlias in pageService.pageMap) {
								refreshFunc();
							} else {
								failedPageAliases[pageAlias] = true;
							}
						}
					});
				}
			}
		};

		$pageText.find('a').each(function(index, element) {
			var $element = $(element);
			var parts = $element.attr('href').match(pageRe);
			if (parts !== null && !$element.hasClass('intrasite-link')) {
				processPageLink($element, parts[0], parts[1], parts[2]);
			}

			parts = $element.attr('href').match(userRe);
			if (parts !== null) {
				var userAlias = parts[1];

				if (!$element.hasClass('user-link')) {
					$element.addClass('user-link').attr('user-id', userAlias);
					if (userAlias in pageService.pageMap) {
					} else {
						// Mark as red link
						$element.addClass('red-link');
						if (refreshFunc && !(userAlias in failedPageAliases)) {
							pageService.loadTitle(userAlias, {
								silentFail: true,
								success: function() {
									if (userAlias in pageService.pageMap) {
										refreshFunc();
									} else {
										failedPageAliases[userAlias] = true;
									}
								}
							});
						}
					}
				}
			}
		});

		var index = 1; // start with 1, because masteryMap is at 0 (see pageService.js)
		$pageText.find('arb-multiple-choice,arb-checkbox').each(function() {
			$(this).attr('index', index++);
		});
		// NOTE: have to compile children individually because otherwise there is a bug
		// with intrasite popovers in preview.
		$pageText.children().each(function(index) {
			$compile($(this))(scope);
		});
		if (refreshFunc) {
			$timeout.cancel(scope._mathRenderPromise);
			scope._mathRenderPromise = $timeout(function() {
				$pageText.find('[arb-math-compiler]').each(function() {
					var $element = $(this);
					var encodedMathjaxText = $element.attr('arb-math-compiler');
					var cacheMathjaxDom = function() {
						if ($element.closest('body').length <= 0) {
							return;
						}
						pageService.cacheMathjax(encodedMathjaxText, {
							html: $element.html(),
							style: 'width:' + $element.css('width') + ';height:' + $element.css('height'),
						});
					};

					// Read from cache
					var cachedValue = pageService.getMathjaxCacheValue(encodedMathjaxText);
					if (cachedValue) {
						$element.html(cachedValue.html);
					} else {
						$element.text(decodeURIComponent(encodedMathjaxText));
						$timeout(function() {
							MathJax.Hub.Queue(['Typeset', MathJax.Hub, $element.get(0)]);
							MathJax.Hub.Queue(cacheMathjaxDom);
						});
					}
					$element.removeClass('MathJax_Display').removeAttr('style');
				});
			}, scope._mathRenderPromise ? 500 : 0);
		}
	};

	this.createConverter = function(pageId) {
		return createConverter(false, pageId);
	};

	this.createEditConverter = function(pageId, postConversionCallback) {
		failedPageAliases = {};
		return createConverter(true, pageId, postConversionCallback);
	};
});

// Directive for rendering markdown text.
app.directive('arbMarkdown', function($compile, $timeout, pageService, markdownService) {
	return {
		scope: {
			// One of these ids has to be set.
			pageId: '@',
			markId: '@',

			// If summary name is set, we'll display the page's corresponding summary
			summaryName: '@',
		},
		controller: function($scope) {
			$scope.pageService = pageService;
			$scope.page = !!$scope.pageId ? pageService.pageMap[$scope.pageId] : undefined;
			$scope.mark = !!$scope.markId ? pageService.markMap[$scope.markId] : undefined;
		},
		link: function(scope, element, attrs) {
			element.addClass('markdown-text reveal-after-render');

			// Remove the class manually in case the text is empty
			$timeout(function() {
				element.removeClass('reveal-after-render');
			});

			// Convert page text to html.
			// Note: converter takes pageId, which might not be set if we are displaying
			// a mark, but it should be ok, since the mark doesn't have most markdown features.
			var converter = markdownService.createConverter(scope.pageId);
			if (scope.page) {
				var html =  scope.page.text;
				if (scope.page.anchorText) {
					html = '>' + scope.page.anchorText + '\n\n' + html;
				}
			} else if (scope.mark) {
				var html = scope.mark.anchorContext;
			}
			if (scope.summaryName) {
				html = scope.page.summaries[scope.summaryName];
				html = html || scope.page.summaries['Summary']; // jscs:ignore requireDotNotation
				if (!html) {
					// Take the first one.
					for (var key in scope.page.summaries) {
						html = scope.page.summaries[key];
						break;
					}
				}
			}
			var html = converter.makeHtml(html);
			var $pageText = element;
			$pageText.html(html);
			window.setTimeout(function() {
				MathJax.Hub.Queue(['Typeset', MathJax.Hub, $pageText.get(0)]);
				MathJax.Hub.Queue(['processLinks', markdownService, scope, $pageText]);
				// Highlight the anchorText for marks.
				MathJax.Hub.Queue(function() {
					if (scope.mark) {
						var highlightClass = 'inline-comment-highlight-hover';
						createInlineCommentHighlight(element.children().get(0), scope.mark.anchorOffset,
							scope.mark.anchorOffset + scope.mark.anchorText.length, highlightClass);
					}
				});
			});
		},
	};
});
