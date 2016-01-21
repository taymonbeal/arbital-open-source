'use strict';

/* jasmine specs for directives go here */

describe('directives', function() {

	var $compile,
			$rootScope;
	var scope, ctrl, $httpBackend;

	// Load the myApp module, which contains the directive
	beforeEach(module('arbital'));

	beforeEach(module('static/html/autocomplete.html'));
	beforeEach(module('static/html/composeFab.html'));
	beforeEach(module('static/html/confirmButton.html'));
	beforeEach(module('static/html/dashboardPage.html'));
	beforeEach(module('static/html/discussion.html'));
	beforeEach(module('static/html/editPage.html'));
	beforeEach(module('static/html/editPageDialog.html'));
	beforeEach(module('static/html/groupIndexPage.html'));
	beforeEach(module('static/html/groupsPage.html'));
	beforeEach(module('static/html/indexPage.html'));
	beforeEach(module('static/html/inlineComment.html'));
	beforeEach(module('static/html/intrasitePopover.html'));
	beforeEach(module('static/html/lens.html'));
	beforeEach(module('static/html/likes.html'));
	beforeEach(module('static/html/page.html'));
	beforeEach(module('static/html/pageList.html'));
	beforeEach(module('static/html/pageTitle.html'));
	beforeEach(module('static/html/primaryPage.html'));
	beforeEach(module('static/html/relationships.html'));
	beforeEach(module('static/html/settingsPage.html'));
	beforeEach(module('static/html/signupPage.html'));
	beforeEach(module('static/html/subpage.html'));
	beforeEach(module('static/html/subscribe.html'));
	beforeEach(module('static/html/toolbar.html'));
	beforeEach(module('static/html/updatesPage.html'));
	beforeEach(module('static/html/userName.html'));
	beforeEach(module('static/html/userPage.html'));
	beforeEach(module('static/html/userPopover.html'));
	beforeEach(module('static/html/voteBar.html'));

	var testPage = {

		pageId:1,
		edit:0,
		type:"",
		title:"title",
		clickbait:"",
		textLength:0,
		alias:"",
		sortChildrenBy:"",
		hasVote:false,
		voteType:"",
		creatorId:0,
		createdAt:"",
		originalCreatedAt:"",
		editKarmaLock:0,
		seeGroupId:0,
		editGroupId:0,
		isAutosave:false,
		isSnapshot:false,
		isCurrentEdit:false,
		isMinorEdit:false,
		todoCount:0,
		anchorContext:"",
		anchorText:"",
		anchorOffset:0,

		text:"text",
		metaText:"",

		isSubscribed:false,
		subscriberCount:0,
		likeCount:0,
		dislikeCount:0,
		myLikeValue:0,
		likeScore:0,
		lastVisit:"",
		hasDraft:false,

		currentEditNum:0,
		wasPublished:false,
		votes:[],
		lockedVoteType:"",
		maxEditEver:0,
		myLastAutosaveEdit:0,
		redLinkCount:0,
		childDraftId:0,
		lockedBy:0,
		lockedUntil:"",
		nextPageId:0,
		prevPageId:0,
		usedAsMastery:false,

		summaries:[],

		answerIds:[],
		commentIds:[],
		questionIds:[],
		lensIds:[],
		taggedAsIds:[],
		relatedIds:[],
		requirementIds:[],

		answerCount:0,
		commentCount:0,

		domainIds:[],

		changeLogs:[],

		hasChildren:false,
		hasParents:false,
		childIds:[],
		parentIds:[],

		members:[]
	};

	var testUser = {
		id:"1",
		firstName:"firstName",
		lastName:"lastname",
		lastWebsiteVisit:0,
		isSubscribed:0,
	}

	// Store references to $rootScope and $compile
	// so they are available to all tests in this describe block
	beforeEach(inject(function(_$compile_, _$rootScope_){
		// The injector unwraps the underscores (_) from around the parameter names when matching
		$compile = _$compile_;
		$rootScope = _$rootScope_;
	}));

	beforeEach(inject(function(_$httpBackend_, $rootScope, $controller) {
		$httpBackend = _$httpBackend_;
		$httpBackend.whenPOST('/json/userPopover/').
				respond([{}]);

		$httpBackend.whenPOST('/json/intrasitePopover/').
				respond([{}]);

		$httpBackend.whenGET('static/icons/arbital-logo.svg').respond([{}]);
		$httpBackend.whenGET('static/icons/thumb-up-outline.svg').respond([{}]);
		$httpBackend.whenGET('static/icons/thumb-down-outline.svg').respond([{}]);
		$httpBackend.whenGET('static/icons/link-variant.svg').respond([{}]);
		$httpBackend.whenGET('static/icons/comment-plus-outline.svg').respond([{}]);
		$httpBackend.whenGET('static/icons/format-header-pound.svg').respond([{}]);

		ctrl = $controller('ArbitalCtrl', {$scope: $rootScope});

		$rootScope.pageService.addPageToMap(testPage);
		$rootScope.pageId = 1;
		$rootScope.pageService.primaryPage = testPage;
		$rootScope.pageService.editMap[$rootScope.pageId] = testPage;
		$rootScope.userService.user = testUser;
	}));
/*
	it('testing arb-user-popover', function() {
		var element = $compile("<arb-user-popover user-id='" + "1" +
			"' direction='" + "down" + "' arrow-offset='" + "0" +
			"'></arb-user-popover>")($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});

	it('testing arb-intrasite-popover', function() {
		var element = $compile("<arb-intrasite-popover page-id='" + 1 +
			"' direction='" + "down" + "' arrow-offset='" + 0 +
			"'></arb-intrasite-popover>")($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});

	it('testing arb-edit-page', function() {
		var element = $compile("<arb-edit-page class='full-height' page-id='" + 1 +
			"' done-fn='doneFn(result)' layout='column'></arb-edit-page>")($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});

	it('testing arb-inline-comment', function() {
		var element = $compile($("<arb-inline-comment" +
			" lens-id='" + 1 +
			"' comment-id='" + 1 + "'></arb-inline-comment>"))($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});

	it('testing arb-discussion', function() {
		var element = $compile("<arb-discussion class='reveal-after-render' page-id='" + 1 +
			"'></arb-discussion>")($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});

	it('testing arb-group-index', function() {
		var element = $compile("<arb-group-index group-id='" + 1 +
			"' ids-map='indexPageIdsMap'></arb-group-index>")($rootScope);
		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toNotEqual("");
	});
*/
/*
	it('testing arb-index', function() {
		testPage.pageId = 3440973961008233681;

		var element = $compile("<arb-index featured-domains='featuredDomains'></arb-index>")($rootScope);

		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toContain("a");
	});
*/
/*
	it('testing arb-group-index', function() {
		var element = $compile("<arb-primary-page></arb-primary-page>")($rootScope);

		$rootScope.$digest();
		console.log(element);
		expect(element.html()).toContain("a");
	});
*/

	function compileElement(elementText) {
		console.log(testPage.text);
		var element = $compile(elementText)($rootScope);
		$rootScope.$digest();
		console.log(element.html());
		return element;
	}

	it('testing markdown', function() {
		var elementText = "<arb-markdown class='popover-text-container' page-id='1'></arb-markdown>";
		var testPage2 = {
			pageId:2,
			alias:"existentAlias",
			title:"existentAlias"
		};
		$rootScope.pageService.addPageToMap(testPage2);

		testPage.text = "[existentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("page-id")).toEqual("2");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toEqual("existentAlias");

		testPage.text = "[nonexistentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("page-id")).toEqual("nonexistentAlias");
		expect($aTag.attr("class")).toContain("red-link");
		expect($aTag.text()).toEqual("nonexistentAlias");

		testPage.text = "[existentAlias description]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("page-id")).toEqual("2");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toEqual("description");

		testPage.text = "[nonexistentAlias description]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("page-id")).toEqual("nonexistentAlias");
		expect($aTag.attr("class")).toContain("red-link");
		expect($aTag.text()).toEqual("description");

		testPage.text = "[hyphenated-alias]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[hyphenated-alias]");

		testPage.text = "[hyphenated-alias description]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[hyphenated-alias description]");

		testPage.text = "[^%@#&^!@ test]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[^%@#&^!@ test]");

		testPage.text = "[http://google.com google]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://google.com");
		expect($aTag.text()).toEqual("google");

		testPage.text = "[ text]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/edit");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[@1]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/user/1");
		expect($aTag.attr("user-id")).toEqual("1");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toEqual("title");

		testPage.text = "[@999]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/user/999");
		expect($aTag.attr("user-id")).toEqual("999");
		expect($aTag.attr("class")).toContain("red-link");
		expect($aTag.text()).toEqual("999");

		testPage.text = "[text](existentAlias)";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("text");

		testPage.text = "[text](nonexistentAlias)";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("text");

		testPage.text = "[text](http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://google.com");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[vote:existentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/pages/existentAlias/?embedVote=1");
		expect($aTag.attr("page-id")).toEqual("existentAlias");
		expect($aTag.attr("embed-vote-id")).toEqual("existentAlias");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toContain("Embedded existentAlias vote.");

		testPage.text = "[vote:nonexistentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/pages/nonexistentAlias/?embedVote=1");
		expect($aTag.attr("page-id")).toEqual("nonexistentAlias");
		expect($aTag.attr("embed-vote-id")).toEqual("nonexistentAlias");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toContain("Embedded nonexistentAlias vote.");

		testPage.text = "[todo:text]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("");

		testPage.text = "[comment:text]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("");

		testPage.text = "[summary(optional):markdown]";
		var element = compileElement(elementText);
		expect(element.text()).toEqual("");

		testPage.text = "[text](http://foo.com/blah_(wikipedia)#cite-1)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://foo.com/blah_(wikipedia)#cite-1");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text](http://www.example.com/wpstyle/?p=364)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://www.example.com/wpstyle/?p=364");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text](https://www.example.com/foo/?bar=baz&inga=42&quux)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("https://www.example.com/foo/?bar=baz&inga=42&quux");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text](http://userid:password@example.com:8080)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://userid:password@example.com:8080");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text](http://foo.bar/?q=Test%20URL-encoded%20stuff)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://foo.bar/?q=Test%20URL-encoded%20stuff");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text](http://)";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("text");

		testPage.text = "[text]()";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("text");

		testPage.text = "\\[existentAlias]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[existentAlias]");

		testPage.text = "[existentAlias\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[existentAlias]");

		testPage.text = "\\[existentAlias\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[existentAlias]");

		testPage.text = "\\\\[existentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("page-id")).toEqual("2");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toEqual("existentAlias");

		testPage.text = "[existentAlias\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[existentAlias\\]");

		testPage.text = "\\\\[existentAlias\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("\\[existentAlias\\]");

		testPage.text = "\\[vote:existentAlias]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[vote:existentAlias]");

		testPage.text = "[vote:existentAlias\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[vote:existentAlias]");

		testPage.text = "\\[vote:existentAlias\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[vote:existentAlias]");

		testPage.text = "\\\\[vote:existentAlias]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/pages/existentAlias/?embedVote=1");
		expect($aTag.attr("page-id")).toEqual("existentAlias");
		expect($aTag.attr("embed-vote-id")).toEqual("existentAlias");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toContain("Embedded existentAlias vote.");

		testPage.text = "[vote:existentAlias\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		//expect($pTag.text()).toEqual("[vote:existentAlias\\]");

		testPage.text = "\\\\[vote:existentAlias\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		//expect($pTag.text()).toEqual("\\[vote:existentAlias\\]");

		testPage.text = "\\[text](http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toEqual("http://google.com");
		//expect($aTag.text()).toEqual("http://google.com");

		testPage.text = "[text\\](http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toEqual("http://google.com");
		//expect($aTag.text()).toEqual("http://google.com");

		testPage.text = "[text]\\(http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toContain("/edit/text");
		//expect($aTag.text()).toEqual("text");

		testPage.text = "[text](http://google.com\\)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toEqual("http://google.com)");
		//expect($aTag.text()).toEqual("http://google.com)");

		testPage.text = "\\\\[text](http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toEqual("http://google.com");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[text\\\\](http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toEqual("http://google.com");
		//expect($aTag.text()).toEqual("text\\");

		testPage.text = "[text]\\\\(http://google.com)";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toContain("/edit/text");
		//expect($aTag.text()).toEqual("texthttp://google.com");

		testPage.text = "[text](http://google.com\\\\)";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		//expect($pTag.text()).toEqual("text");

		testPage.text = "\\[@1]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[@1]");

		testPage.text = "[@1\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[@1]");

		testPage.text = "\\[@1\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[@1]");

		testPage.text = "\\\\[@1]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/user/1");
		expect($aTag.attr("user-id")).toEqual("1");
		expect($aTag.attr("class")).toNotContain("red-link");
		expect($aTag.text()).toEqual("title");

		testPage.text = "[@1\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[@1\\]");

		testPage.text = "\\\\[@1\\\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("\\[@1\\]");

		testPage.text = "\\[ text]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[ text]");

		testPage.text = "[ text\\]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toContain("/edit");
		//expect($aTag.text()).toEqual("http://arbital.com/edit");

		testPage.text = "\\[ text\\]";
		var element = compileElement(elementText);
		var $pTag = $(element.html());
		expect($pTag.text()).toEqual("[ text]");

		testPage.text = "\\\\[ text]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		expect($aTag.attr("href")).toContain("/edit");
		expect($aTag.text()).toEqual("text");

		testPage.text = "[ text\\\\]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toContain("/edit");
		//expect($aTag.text()).toEqual("text\\");

		testPage.text = "\\\\[ text\\\\]";
		var element = compileElement(elementText);
		var $aTag = $(element.html()).find("a");
		//expect($aTag.attr("href")).toContain("/edit");
		//expect($aTag.text()).toEqual("text\\");
	});
});
