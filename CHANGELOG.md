# CHANGELOG

## 1.5.0 (February 22nd 2023)

### Change:

- Complied with latest Go binaries because of security advisory.

## 1.4.0 (May 20th, 2022)

### Feature 

- Added logic to create initial status history record, to support changes in Service Manager

## 1.3.3 (April 1st, 2022)

### Change

- Updated code to support application segregation

## 1.3.2 (January 28th, 2022)

Fix:
- Fixed prefix issue caused by API change


## 1.3.1 (September 24th, 2021)

Changes:
- Allow for Analyst recognition on various fields (h_ownerid, h_createdby, h_closedby_user_id, h_resolvedby_user_id, h_reopenedby_user_id, h_lastmodifieduserid)
- Allow for team ID mapping on various fields (h_fk_team_id, h_closedby_team_id, h_resolvedby_team_id, h_reopenedby_team_id)

Fixes:
- file attachments - there was an issue if a specific ID was too long
- ensure createdby is set correctly (instead of picking up the API Key's context user).

## 1.3.0 (September 13th, 2021)

Changes:

- Added ability to specify unique column for Analyst (owner) AnalystUniqueColumn a
- using API Key and Instance ID

## 1.2.0 (August 31st, 2021)

Changes:

- Added support for Release Request workflows to be spawned
- Added ability to specify unique column for Account (Custom Type 0) or Contact (Customer Type 1)
- Removed references to content location for file attachments, as this is no longer required
- brought codebase more up to date (entityBrowserRecords2)

## 1.1.1 (July 6th, 2021)

Change:

- Rebuilt using latest version of goApiLib, to fix possible issue with connections via a proxy

## 1.1.0 (August 25th, 2020)

Changes:

- Set h_archived column to 1 when requests are being imported in a cancelled state
- Changed BPM spawning to use processSpawn2 instead of processSpawn
- Added version checking code

## 1.0.1 (April 16th, 2020)

Changes:

- Updated code to support Core application and platform changes
- Added version flag to enable auto-build

## 1.0.0 (04/01/2017)

- Initial Release
