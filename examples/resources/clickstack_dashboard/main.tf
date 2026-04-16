# Discover available data sources and webhooks for use in dashboards and alerts.
data "clickstack_sources" "all" {}
data "clickstack_webhooks" "all" {}

# ---------------------------------------------------------------------------
# Dashboard: API Overview
# A multi-tile dashboard monitoring API request volume, errors, and latency.
# ---------------------------------------------------------------------------
resource "clickstack_dashboard" "api_overview" {
  name = "API Overview"
  tags = ["api", "production"]

  # Optional: pre-populate the dashboard search bar
  saved_query          = "service:api-gateway"
  saved_query_language = "lucene"

  # Optional: dashboard-level filters that apply across all tiles
  filter {
    type       = "field"
    name       = "Environment"
    expression = "environment"
    source_id  = data.clickstack_sources.all.sources[0].id
  }

  # --- Row 1: KPI numbers (full width, 4 tiles across) ---
  tile {
    name = "Total Requests"
    x    = 0
    y    = 0
    w    = 6
    h    = 3
    config_json = jsonencode({
      source       = "log"
      type         = "number"
      query        = "service:api-gateway"
      field        = ""
      numberFormat = "short"
    })
  }

  tile {
    name = "Error Count"
    x    = 6
    y    = 0
    w    = 6
    h    = 3
    config_json = jsonencode({
      source       = "log"
      type         = "number"
      query        = "service:api-gateway level:error"
      field        = ""
      numberFormat = "short"
    })
  }

  tile {
    name = "P95 Latency (ms)"
    x    = 12
    y    = 0
    w    = 6
    h    = 3
    config_json = jsonencode({
      source       = "metric"
      type         = "number"
      query        = "http.server.duration"
      field        = "p95"
      numberFormat = "ms"
    })
  }

  tile {
    name = "Active Users"
    x    = 18
    y    = 0
    w    = 6
    h    = 3
    config_json = jsonencode({
      source       = "session"
      type         = "number"
      query        = ""
      field        = ""
      numberFormat = "short"
    })
  }

  # --- Row 2: Time-series charts ---
  tile {
    name = "Request Rate"
    x    = 0
    y    = 3
    w    = 12
    h    = 6
    config_json = jsonencode({
      source = "log"
      type   = "line"
      query  = "service:api-gateway"
      field  = ""
    })
  }

  tile {
    name = "Errors by Status Code"
    x    = 12
    y    = 3
    w    = 12
    h    = 6
    config_json = jsonencode({
      source  = "log"
      type    = "stacked_bar"
      query   = "service:api-gateway level:error"
      field   = ""
      groupBy = "http.status_code"
    })
  }

  # --- Row 3: Table + Pie ---
  tile {
    name = "Top Endpoints"
    x    = 0
    y    = 9
    w    = 16
    h    = 6
    config_json = jsonencode({
      source  = "log"
      type    = "table"
      query   = "service:api-gateway"
      groupBy = "http.route"
      columns = ["http.route", "count", "avg_duration"]
      sortBy  = "count"
      sortDir = "desc"
    })
  }

  tile {
    name = "Traffic by Method"
    x    = 16
    y    = 9
    w    = 8
    h    = 6
    config_json = jsonencode({
      source  = "log"
      type    = "pie"
      query   = "service:api-gateway"
      groupBy = "http.method"
    })
  }

  # --- Row 4: Log search ---
  tile {
    name = "Recent Errors"
    x    = 0
    y    = 15
    w    = 24
    h    = 8
    config_json = jsonencode({
      source  = "log"
      type    = "search"
      query   = "service:api-gateway level:error"
      columns = ["timestamp", "level", "http.route", "http.status_code", "message"]
    })
  }

  # --- Row 5: Markdown notes ---
  tile {
    name = "Runbook"
    x    = 0
    y    = 23
    w    = 24
    h    = 3
    config_json = jsonencode({
      type    = "markdown"
      content = "## Runbook\n- **High error rate**: Check your incident management tool for active incidents.\n- **Latency spike**: Check ClickHouse query performance and DB connection pool.\n- **Traffic drop**: Verify CDN health and DNS resolution."
    })
  }
}
