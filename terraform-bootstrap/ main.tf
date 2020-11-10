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
