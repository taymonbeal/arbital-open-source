/* This file contains the recent changes to schemas, sorted from oldest to newest. */
alter table marks add resolvedPageId VARCHAR(32) NOT NULL;
alter table marks add column resolvedBy varchar(32) not null;
alter table marks drop column questionId;
alter table updates add column markId BIGINT NOT NULL;

alter table changeLogs modify column oldSettingsValue varchar(1024) not null;
alter table changeLogs modify column newSettingsValue varchar(1024) not null;

alter table marks add column answered BOOLEAN NOT NULL;
alter table pagePairs add column everPublished boolean not null;
update pagePairs set everPublished = 1
where
	parentId not in (select pageId from pageInfos where currentEdit <= 0) and
	childId not in (select pageId from pageInfos where currentEdit <= 0);

alter table pageInfos add column mergedInto varchar(32) not null;
alter table marks add column type varchar(32) not null;
update marks set type="query";
alter table marks add column resolvedAt datetime not null;
alter table marks add column answeredAt datetime not null;

alter table pageInfos add column likeableId bigint not null;
alter table changelogs add column likeableId bigint not null;

/* A table for keeping track of likeableIds */
CREATE TABLE likeableIds (
	/* Id of the likeable. */
	id BIGINT NOT NULL AUTO_INCREMENT,
	PRIMARY KEY(id)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
