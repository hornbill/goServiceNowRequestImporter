# CHANGELOG

## 1.2.0 (August 31st, 2021)

Changes:

- Added support for Release Request workflows to be spawned
- Added ability to specify unique column for Account (Custom Type 0) or Contact (Customer Type 1)
- Removed references to content location for file attachments, as this is no longer required
- brought codebase more up to date (entityBrowserRecords2)

## 1.1.1 (July 6th, 2021)

Change:

- Rebuilt using latest version of goApiLib, to fix possible issue with connections via a proxy

## 1.1.0 (August 25th, 2020)

Changes:

- Set h_archived column to 1 when requests are being imported in a cancelled state
- Changed BPM spawning to use processSpawn2 instead of processSpawn
- Added version checking code

## 1.0.1 (April 16th, 2020)

Changes:

- Updated code to support Core application and platform changes
- Added version flag to enable auto-build

## 1.0.0 (04/01/2017)

- Initial Release
