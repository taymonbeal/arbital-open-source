'use strict';

import app from './angular.ts';
import {arraysSortFn} from './util.ts';

// arb-project directive displays the project page
app.directive('arbProject', function($http, $mdMedia, arb) {
	return {
		templateUrl: versionUrl('static/html/projectPage.html'),
		controller: function($scope) {
			$scope.arb = arb;
			$scope.isTinyScreen = !$mdMedia('gt-xs');

			// Slack stuff
			$scope.showJoinSlackInput = false;
			$scope.showJoinSlackButton = arb.userService.user && !arb.userService.user.isSlackMember;
			if (Cookies.getJSON('isSlackMember')) {
				$scope.showJoinSlackButton = false;
			}

			$scope.slackInvite = {email: ''};

			$scope.joinSlack = function() {
				$scope.showJoinSlackInput = true;
				$scope.slackInvite.email = arb.userService.user.email;
			};

			$scope.joinSlackSubmit = function() {
				arb.stateService.postDataWithoutProcessing('/json/sendSlackInvite/', $scope.slackInvite, function() {
					arb.userService.user.isSlackMember = true;
					Cookies.set('isSlackMember', true);
				});
				arb.userService.user.isSlackMember = true;
			};

			// Project stuff
			$scope.expandedProjectPages = {};
			$scope.toggleProjectTodos = function(pageId) {
				$scope.expandedProjectPages[pageId] = !$scope.expandedProjectPages[pageId];
			};

			arb.stateService.postData('/json/project/', {projectPageId: '5wy'},
				function(response) {
					// Store the number of pages at each quality level. Since we want these to be sorted,
					// we'll use an array.
					$scope.qualityCounts = [
						{key: '4yl', name: 'Featured', count: 0, weight: 1},
						{key: '4yf', name: 'A-Class', count: 0, weight: 1},
						{key: '4yd', name: 'B-Class', count: 0, weight: 1},
						{key: '4y7', name: 'C-Class', count: 0, weight: 0.75},
						{key: '3rk', name: 'Start', count: 0, weight: 0.1},
						{key: '72', name: 'Stub', count: 0, weight: 0.05},
						{key: '', name: 'unwritten', count: 0, weight: 0},
					];
					let incrementQualityCount = function(key) {
						for (let n = 0; n < $scope.qualityCounts.length; n++) {
							if ($scope.qualityCounts[n].key === key) {
								$scope.qualityCounts[n].count++;
								break;
							}
						}
					};

					// Compute rows to display all the pages in the project and number
					// of pages in various categories
					var aliasRows = response.result.projectData.aliasRows.map(function(aliasRow) {
						incrementQualityCount('');
						return {isRedLink: true, alias: aliasRow.alias};
					});
					var pageRows = response.result.projectData.pageIds.map(function(pageId) {
						var page = arb.stateService.getPage(pageId);
						page.qualityTag = arb.pageService.getQualityTagId(page.tagIds);
						incrementQualityCount(page.qualityTag);
						arb.pageService.computeTodos(page);
						$scope.expandedProjectPages[page.pageId] = false;
						return page;
					});
					$scope.projectPageRows = pageRows.concat(aliasRows);
					$scope.projectPageRows.sort(arraysSortFn(function(row) {
						// If the user is not logged in, reverse the sort order
						let s = arb.userService.userIsLoggedIn() ? 1 : -1;
						let array = [s * (row.isRedLink ? 0 : 1)];
						if (row.pageId) {
							array = array.concat([
								s * (row.tagIds.includes('4yl') ? 1 : 0),
								s * (row.tagIds.includes('4yf') ? 1 : 0),
								s * (row.tagIds.includes('4yd') ? 1 : 0),
								s * (row.tagIds.includes('4y7') ? 1 : 0),
								s * (row.tagIds.includes('3rk') ? 1 : 0),
								s * (row.tagIds.includes('72') ? 1 : 0),
							]);
						}
						return array;
					}));

					// Compute percent complete
					$scope.percentComplete = 0;
					let qualityStrings = [];
					for (let n = 0; n < $scope.qualityCounts.length; n++) {
						let quality = $scope.qualityCounts[n];
						$scope.percentComplete += quality.count * quality.weight;
						if (quality.count <= 0) continue;
						let qualityStr = quality.count + ' ' + quality.name + ' page';
						if (quality.count != 1) qualityStr += 's';
						qualityStrings.push(qualityStr);
					}
					$scope.percentComplete = +Math.floor(($scope.percentComplete * 100) / (aliasRows.length + pageRows.length));
					$scope.projectStatusText = $scope.percentComplete + '% complete: ' + qualityStrings.join(', ');

					// Compute recent changes rows
					$scope.changeLogModeRows = [];
					let acceptedChangeLogTypes = {newEditProposal: true, newEdit: true, deletePage: true, revertEdit: true};
					for (let n = 0; n < response.result.projectData.pageIds.length; n++) {
						let page = arb.stateService.pageMap[response.result.projectData.pageIds[n]];
						for (let i = 0; i < page.changeLogs.length; i++) {
							let changeLog = page.changeLogs[i];
							if (!acceptedChangeLogTypes[changeLog.type]) continue;
							$scope.changeLogModeRows.push({
								rowType: changeLog.type,
								activityDate: changeLog.createdAt,
								changeLog: changeLog,
							});
						}
					}

					// Compute "X changes by Y authors in last week" text
					let changeCountLastWeek = 0;
					let authorIdsSet = {};
					let now = moment.utc();
					for (let n = 0; n < $scope.changeLogModeRows.length; n++) {
						let changeLog = $scope.changeLogModeRows[n].changeLog;
						if (now.diff(moment.utc(changeLog.createdAt), 'days') <= 7) {
							authorIdsSet[changeLog.userId] = true;
							changeCountLastWeek++;
						}
					}
					let authorCountLastWeek = Object.keys(authorIdsSet).length;
					$scope.changesCountText = '' + changeCountLastWeek;
					if (changeCountLastWeek == 1) {
						$scope.changesCountText += ' change';
					} else {
						$scope.changesCountText += ' changes';
					}
					$scope.changesCountText += ' by ' + authorCountLastWeek;
					if (authorCountLastWeek == 1) {
						$scope.changesCountText += ' author';
					} else {
						$scope.changesCountText += ' authors';
					}
					$scope.changesCountText += ' last week';
				});
		},
	};
});
