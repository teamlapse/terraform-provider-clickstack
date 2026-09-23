# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- A `clickstack_dashboard` tile declared with `config_json` no longer plans an update when ClickStack returns its default for a key the declaration leaves out: `fillNulls: true` on a line or stacked-bar tile, `asRatio: false` on a builder line, stacked-bar or table tile, `select: ""` and `whereLanguage: "lucene"` on a search tile, and `markdown: ""` on a markdown tile. A stored value that differs from the default still shows.
- A `clickstack_alert` on a tile other than a line, stacked-bar or number tile now fails to create or update. ClickStack accepted such an alert but never evaluated it, and deleted it the next time the dashboard was saved, so the following plan created it again.
- A dashboard, alert or saved search read while a write to the same type was in flight no longer keeps serving the pre-write copy once the write completes.
- A `clickstack_dashboard` tile declared with `config_json` keeps its declared config on read when ClickStack's stored copy only lacks keys (such as a `source` name it has resolved), instead of planning an update on every run.

### Changed

- The sources and webhooks lists are fetched once per plan and shared by every data block that reads them.

### Changed

- Reads of dashboards, alerts and saved searches are served from one list per resource type per plan instead of one request per resource, so a large root no longer trips the API's rate limit while refreshing; an id the list does not carry is still fetched directly, and any write invalidates the list.

### Fixed

- A `clickstack_dashboard` tile declared with `series_json` no longer plans an update on every run: ClickStack stores it as `config` and returns only that on read, which the provider now keeps in the declared form.

## [0.0.5]

### Fixed

- Prevent irrelevant source IDs returned by the API from being stored for `clickstack_alert` resources.

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
