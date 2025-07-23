# k8s-support-archive-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- [#10] add network policies to allow ingress from k8s-ces-control (and deny everything else)

### Added
- [#5] Adds initial logic for the operator
    - Archive creation
    - Finalizer for cleanup
    - Collector to fetch data
    - Metadata for state recognition
    - Nginx sidecar to expose create archives

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