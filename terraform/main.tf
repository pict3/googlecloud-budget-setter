provider "google" {
  project = var.project_id
}

data "google_cloud_run_locations" "default" { }

resource "google_cloud_run_service" "default" {
  for_each = toset([for location in data.google_cloud_run_locations.default.locations : location if can(regex("us-(?:west|central|east)1", location))])

  name     = "${var.name}--${each.value}"
  location = each.value
  project  = var.project_id

  autogenerate_revision_name = true

  template {
    spec {
      containers {
        image = var.image
        resources {
          limits = { "memory" : "1024Mi" }
        }
      }
      service_account_name = google_service_account.default.email
    }
  }
}

resource "google_cloud_run_service_iam_member" "default" {
  for_each = toset([for location in data.google_cloud_run_locations.default.locations : location if can(regex("us-(?:west|central|east)1", location))])

  location = google_cloud_run_service.default[each.key].location
  project  = google_cloud_run_service.default[each.key].project
  service  = google_cloud_run_service.default[each.key].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_service_account" "default" {
  account_id = "budget-setter-sa"
  display_name = "budget-setter-sa"
}

resource "google_project_iam_member" "default" {
  role = "roles/billing.costsManager"
  member = "serviceAccount:${google_service_account.default.email}"
}

resource "google_compute_region_network_endpoint_group" "default" {
  for_each = toset([for location in data.google_cloud_run_locations.default.locations : location if can(regex("us-(?:west|central|east)1", location))])

  name                  = "${var.name}--neg--${each.key}"
  network_endpoint_type = "SERVERLESS"
  region                = google_cloud_run_service.default[each.key].location
  cloud_run {
    service = google_cloud_run_service.default[each.key].name
  }
}

resource "google_compute_ssl_policy" "upper_1_2_policy" {
  name            = "upper-1-2-policy"
  min_tls_version = "TLS_1_2"
}

module "lb-http" {
  source            = "GoogleCloudPlatform/lb-http/google//modules/serverless_negs"
  version           = "~> 4.5"

  project = var.project_id
  name    = var.name

  managed_ssl_certificate_domains = [var.domain]
  ssl                             = true
  https_redirect                  = true
  ssl_policy                      = google_compute_ssl_policy.upper_1_2_policy.name

  backends = {
    default = {
      description            = null
      enable_cdn             = false
      custom_request_headers = null

      log_config = {
        enable      = true
        sample_rate = 1.0
      }

      groups = [
        for neg in google_compute_region_network_endpoint_group.default:
        {
          group = neg.id
        }
      ]

      iap_config = {
        enable               = false
        oauth2_client_id     = null
        oauth2_client_secret = null
      }
      security_policy = null
    }
  }
}

output "url" {
  value = "http://${module.lb-http.external_ip}"
}