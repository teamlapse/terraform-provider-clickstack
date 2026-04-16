# ---------------------------------------------------------------------------
# Data Sources: Discover available sources and webhooks
# ---------------------------------------------------------------------------

# List all ClickStack data sources (log, trace, metric, session).
# Use these IDs when configuring dashboard filters.
data "clickstack_sources" "all" {}

output "sources" {
  value = data.clickstack_sources.all.sources
}

# List all configured webhooks (Slack, PagerDuty, etc.).
# Use these IDs when configuring alert notification channels.
data "clickstack_webhooks" "all" {}

output "webhooks" {
  value     = data.clickstack_webhooks.all.webhooks
  sensitive = true
}
