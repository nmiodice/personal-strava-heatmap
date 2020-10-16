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
