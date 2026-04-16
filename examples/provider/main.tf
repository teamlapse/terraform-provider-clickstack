terraform {
  required_providers {
    clickstack = {
      source = "registry.terraform.io/teamlapse/clickstack"
    }
  }
}

# Configure via environment variables (recommended):
#   CLICKSTACK_ORGANIZATION_ID
#   CLICKSTACK_SERVICE_ID
#   CLICKSTACK_API_KEY_ID
#   CLICKSTACK_API_KEY_SECRET
provider "clickstack" {
  organization_id = var.clickstack_organization_id
  service_id      = var.clickstack_service_id
  api_key_id      = var.clickstack_api_key_id
  api_key_secret  = var.clickstack_api_key_secret
}

variable "clickstack_organization_id" {
  type        = string
  description = "ClickHouse Cloud organization ID."
}

variable "clickstack_service_id" {
  type        = string
  description = "ClickStack service ID on ClickHouse Cloud."
}

variable "clickstack_api_key_id" {
  type        = string
  sensitive   = true
  description = "ClickHouse Cloud API key ID."
}

variable "clickstack_api_key_secret" {
  type        = string
  sensitive   = true
  description = "ClickHouse Cloud API key secret."
}
