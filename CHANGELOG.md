# Changelog

## [1.4.0] - 23-09-2026

* Added dashboard-managed SSH public-key access.
* Validates ED25519 public keys and reports their SHA256 fingerprints.
* Atomically installs or removes only the marked Edgebox key while preserving all other authorized keys.
* Never generates, reads, stores, logs, or returns SSH private keys.

## [1.3.2] - 08-12-2024

* Fix to Browser Dev feature:
    * Checking browser dev url when internet_accessible was checking the incorrect path. This is now fixed.

## [1.3.1] - 08-12-2024

* Fixes to Browser Dev feature:
    * Now edgeboxctl also fetches or generates the browser dev environment url and saves it into an option both when starting, and every time the browser dev status is fetched.

## [1.3.0] - 05-12-2024

* Added Edgebox Browser Development Environment Feature Support
    * Added tasks for handling browser development environment into tasks.go
    * Added executable tasks to ExecuteTask and scheduled ones to ExecuteSchedules
* Other bug fixes and improvements.

### Missing Past Releases

Release notes for past versions are not available in this file. Please refer to the [GitHub releases](https://hithub.com/edgebox-iot/edgeboxctl/releases) for more information. Feel free to contribute to this file by adding missing release notes.
