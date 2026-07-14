# k8s-support-archive-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
- [#34] add german translation for usage.md

## [v1.1.2] - 2026-06-08
### Fixed
- [#31] Remove hardcoded cluster domain

## [v1.1.1] - 2026-02-17
### Security
- [#29] Fix Golang stdlib CVE-2025-68121
- Update nginx to 1.29.5 to fix OpenSSL CVE-2025-15467

## [v1.1.0] - 2026-01-06
### Changed
- [#27] Make storageclass configurable with value `webserver.pvc.storageClassName` and keep the default storageclass as default.

## [v1.0.1] - 2025-09-26
### Fixed
- [#25] Update helm dependency for crd to fix component installation error.

## [v1.0.0] - 2025-09-25

### Added
- [#15] Garbage-collect old support archives
- [#14] Collect secrets information and add it to support archive
- [#16] Collect node information from prometheus
- [#20] Collect system state
- [#13] Collect events 
- [#23] Collect logs

## [v0.3.0] - 2025-08-07
### Added
- [#5] Adds initial logic for the operator
    - Archive creation
    - Finalizer for cleanup
    - Collector to fetch data
    - Metadata for state recognition
    - Nginx sidecar to expose create archives
- [#9] Collect volume information from prometheus
- Regularly sync archives with cluster state to avoid finalizers
- [#10] Add network policies to all deny ingress

## [v0.2.0] - 2025-07-18
### Added
- [#7] add metadata mapping for logLevel

## [v0.1.2] - 2025-05-06

### Changed
- [#3] Set sensible resource requests and limits

## [v0.1.0] - 2025-03-31

### Added
- [#1] Initialize operator
  - Basic reconciler skeleton
  - Helm chart
  - CI/CD
  - \+ other necessary scaffolding