# ---------------------------------------------------------------------------
# Alerts: Tile-based and Saved-Search-based
# ---------------------------------------------------------------------------

# Look up available webhooks (Slack, PagerDuty, etc.)
data "clickstack_webhooks" "all" {}

# --- Alert from a dashboard tile ---
# Fires when the error count tile exceeds 100 errors in a 5-minute window.
resource "clickstack_alert" "high_error_rate" {
  name           = "High Error Rate"
  message        = "Error count exceeded 100 in the last 5 minutes. Check the API Overview dashboard."
  source         = "tile"
  threshold      = 100
  threshold_type = "above"
  interval       = "5m"
  dashboard_id   = clickstack_dashboard.api_overview.id
  tile_id        = clickstack_dashboard.api_overview.tile[1].id # Error Count tile

  channel {
    type             = "webhook"
    webhook_id       = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service  = "slack"
    slack_channel_id = "C0123456789" # #incidents channel
  }
}

# --- Alert from a saved search ---
# Fires when more than 5 OOM kills are detected in a 15-minute window.
resource "clickstack_saved_search" "oom_kills" {
  name   = "OOM Kill Events"
  query  = "\"Out of memory\" OR \"OOMKilled\""
  source = "log"
  tags   = ["infrastructure", "alerts"]
}

resource "clickstack_alert" "oom_kills" {
  name            = "OOM Kill Detected"
  message         = "OOM kill events detected — check container resource limits."
  source          = "saved_search"
  saved_search_id = clickstack_saved_search.oom_kills.id
  threshold       = 5
  threshold_type  = "above"
  interval        = "15m"

  channel {
    type            = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service = "pagerduty_api"
    severity        = "critical"
  }
}

# --- Alert with email notification ---
resource "clickstack_alert" "latency_degradation" {
  name           = "P95 Latency Degradation"
  message        = "P95 latency has exceeded 500ms."
  source         = "tile"
  threshold      = 500
  threshold_type = "above"
  interval       = "15m"
  dashboard_id   = clickstack_dashboard.api_overview.id
  tile_id        = clickstack_dashboard.api_overview.tile[2].id # P95 Latency tile
  group_by       = "service"

  channel {
    type             = "email"
    email_recipients = ["oncall@example.com", "platform-team@example.com"]
  }
}
