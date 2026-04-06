# Changelog

All notable changes to this project will be documented in this file.

## [0.6.1] - 2026-03-10
### Fixed
- 🐛 Fixed TLV logger error

### Changed 
- TLV package initialization changed

## [0.6.0] - 2026-02-10
### Added
- ✨ Added MakeFile for project build
- ✨ Added roaming MSISDN generation

### Fixed
- 🐛 Fixed TLV logger error

## [0.5.7] - 2025-10-13
### Added
- ✨ Added custom configurations

### Fixed
- 🐛 Fixed error when rate parameter was specified only once

## [0.5.6] - 2025-07-04
### Added
- ✨ Added network connection close handler from client side
- ✨ Added context to TLV connection
- ✨ Added Diameter reconnection (reconnect)
- ✨ Added offline CDR generation when no active connection is present

### Fixed
- 🐛 Fixed subscriber B number generation
- 🐛 Fixed new thread creation

## [0.5.5] - 2025-03-05
### Changed
- 📦 InfluxDB moved to a separate package (implemented via package)

## [0.5.4] - 2025-02-28
### Added
- ✨ TLV implemented via separate package

### Fixed
- 🐛 Fixed crash due to insufficient permissions to write offline CDR
- 🐛 Fixed crash when encountering invalid lines in CSV Pool

## [0.5.3] - 2025-01-31
### Added
- ✨ Added proper shutdown for Linux

## [0.5.2] - 2025-01-27
### Added
- ✨ Experimental feature (single write stream to Camel)

### Fixed
- 🐛 Fixed context handling

## [0.5.1] - 2024-11-17
### Fixed
- 🐛 Fixed CAMEL connection
- 🐛 Fixed sending time delay

## [0.5.0] - 2024
### Added
- ✨ Added CELL and LAC area pool creation

## [0.4.6] - 2024
### Added
- ✨ Added UDP protocol for InfluxDB v1

### Fixed
- 🐛 Fixed InfluxDB v2 connection

## [0.4.5] - 2024
### Added
- ✨ Added Location MSC generation using CELL and LAC pool

### Fixed
- 🐛 Fixed connection termination

## [0.4.4] - 2024
### Added
- ✨ Added pool re-reading

### Fixed
- 🐛 Minor fixes

## [0.4.3] - 2024
### Added
- ✨ Added offline CDR generation upon receiving CONFIRM with Charge 00
- ✨ Added `thread` parameter to enable starting new threads
- ✨ Added random MSISDN B generation
- ✨ Added `-rm` flag to remove CDR files from temporary directory

## [0.4.2] - 2024
### Added
- ✨ Added InfluxDB statistics logging for Diameter and Camel
- ✨ Changed log writing in main process
- ✨ Added MO and MT call type generation
- ✨ Removed separate Diameter log

### Fixed
- 🐛 Fixed SetReadDeadline timeout for Linux OS

## [0.4.1] - 2024
### Added
- ✨ `CDR_Pattern` parameter moved from array to top level
- ✨ Added Camel speed monitoring
- ✨ Improved function for working with large receive buffers (SCP server)

### Fixed
- 🐛 Fixed error when sending write channel to files
- 🐛 Fixed Diameter sending when connection was not initiated

## [0.4.0] - 2024
### Added
- ✨ Added CAMEL SCP

## [0.3.1] - 2024
### Added
- ✨ Added InfluxDB statistics logging

## [0.3.0] - 2024
### Added
- ✨ Added switch operation emulation with 4011 response (no credit control needed)

### Fixed
- 🐛 Removed variable in Diameter connection initialization loop

## [0.2.1] - 2024
### Fixed
- 🐛 Bug fixes

## [0.2.0] - 2024
### Added
- ✨ Added Diameter connection to Nexign NWM products (3GPP Diameter Credit-Control Application)
- ✨ Support for CCR/CCA "Event" type requests

## [0.1.0] - 2024
### Added
- ✨ Offline load functionality