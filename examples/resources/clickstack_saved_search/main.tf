# ---------------------------------------------------------------------------
# Saved Searches: Reusable queries that can be referenced by alerts
# ---------------------------------------------------------------------------

resource "clickstack_saved_search" "api_errors" {
  name    = "API Errors"
  query   = "service:api-gateway level:error"
  source  = "log"
  tags    = ["api", "errors"]
  columns = ["timestamp", "level", "http.route", "http.status_code", "message"]

  sort {
    field = "timestamp"
    order = "desc"
  }
}

resource "clickstack_saved_search" "slow_traces" {
  name   = "Slow Traces (>1s)"
  query  = "duration:>1000"
  source = "trace"
  tags   = ["performance"]

  sort {
    field = "duration"
    order = "desc"
  }
}

resource "clickstack_saved_search" "auth_failures" {
  name    = "Authentication Failures"
  query   = "service:auth (\"401\" OR \"403\" OR \"invalid token\")"
  source  = "log"
  tags    = ["security", "auth"]
  columns = ["timestamp", "level", "http.route", "user_id", "message"]

  sort {
    field = "timestamp"
    order = "desc"
  }
}
