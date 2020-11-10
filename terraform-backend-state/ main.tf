terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "= 2.31.1"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.0.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "location" {
  type        = string
  description = "Location in which to provision Azure resources"
  default     = "eastus"
}

resource "random_string" "rand" {
  length  = 4
  special = false
  number  = false
  upper   = false
}

resource "azurerm_resource_group" "ci" {
  name     = "rg-ci"
  location = var.location
}

resource "azurerm_storage_account" "ci" {
  name                = format("terraform%s", random_string.rand.result)
  resource_group_name = azurerm_resource_group.ci.name
  location            = azurerm_resource_group.ci.location

  account_tier             = "Standard"
  account_replication_type = "LRS"

  min_tls_version = "TLS1_2"
}

resource "azurerm_storage_container" "tfstate" {
  name                  = "terraform-state"
  storage_account_name  = azurerm_storage_account.ci.name
  container_access_type = "private"
}
