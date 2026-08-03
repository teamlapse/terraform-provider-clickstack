# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial open-source release.
- `clickstack_dashboard` resource for managing dashboards with tiles and filters.
- `clickstack_alert` resource for threshold-based alerts (tile and saved search sources).
- Optional consecutive-window evaluation for `clickstack_alert` via `num_consecutive_windows`.
- `clickstack_saved_search` resource for reusable search queries.
- `clickstack_sources` data source for listing available data sources.
- `clickstack_webhooks` data source for listing configured webhooks.

### Changed

- `clickstack_saved_search` now uses the ClickHouse Cloud saved-search API and supports source IDs, column selection, Lucene or SQL filters, ordering, tags, and pinned SQL filters.
