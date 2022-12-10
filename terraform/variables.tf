variable "project_id" {
  description = "Google Cloud Project ID."
  type        = string
}

variable "region" {
  description = "Region name."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "Zone name."
  type        = string
  default     = "us-central1-f"
}

variable "name" {
  description = "Web server name."
  type        = string
  default     = "budget-setter-server"
}

variable "machine_type" {
  description = "GCE VM instance machine type."
  type        = string
  default     = "f1-micro"
}


variable "image" {
  description = "container image to deploy"
}

variable "domain" {
  description = "domain of site."
  type        = string
}
