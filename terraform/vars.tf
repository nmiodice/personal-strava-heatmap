variable "create_env_file" {
  type    = bool
  default = true

  description = "Create a .env file in the module directory with variables set to the configuration values."
}

variable "service_name" {
  type        = string
  default     = "stvahm"
  description = "Resource prefix"
}

variable "env" {
  type        = string
  default     = "dev"
  description = "Environment (i.e., dev, int, prod)"
}

variable "location" {
  type    = string
  default = "eastus"
}

variable "psql_sku" {
  type        = string
  description = "The SKU to use for the provisioned PSQL instance"
  default     = "B_Gen5_1"
}

variable "strava_client_id" {
  type        = string
  description = "Strava application client ID"
}

variable "strava_client_secret" {
  type        = string
  description = "Strava application client Secret"
}

variable "google_maps_api_key" {
  type        = string
  description = "Google Maps API key"
}

variable "acr_id" {
  type        = string
  description = "ID of the ACR that will host docker images"
}
