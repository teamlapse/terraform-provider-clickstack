# ---------------------------------------------------------------------------
# Saved Searches: Reusable queries that can be referenced by alerts
# ---------------------------------------------------------------------------

data "clickstack_sources" "all" {}

resource "clickstack_saved_search" "api_errors" {
  name      = "API Errors"
  source_id = data.clickstack_sources.all.sources[0].id
  select    = "Timestamp, ServiceName, Body"
  where     = "ServiceName:api-gateway AND SeverityText:ERROR"
  order_by  = "Timestamp DESC"
  tags      = ["api", "errors"]
}

resource "clickstack_saved_search" "slow_traces" {
  name           = "Slow Traces (>1s)"
  source_id      = data.clickstack_sources.all.sources[1].id
  where          = "Duration > 1000"
  where_language = "sql"
  order_by       = "Duration DESC"
  tags           = ["performance"]
}

resource "clickstack_saved_search" "auth_failures" {
  name      = "Authentication Failures"
  source_id = data.clickstack_sources.all.sources[0].id
  select    = "Timestamp, ServiceName, Body"
  where     = "ServiceName:auth AND (Body:401 OR Body:403 OR Body:\"invalid token\")"
  order_by  = "Timestamp DESC"
  tags      = ["security", "auth"]

  filters {
    condition = "ServiceName IN ('auth')"
  }
}
