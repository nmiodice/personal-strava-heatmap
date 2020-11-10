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

terraform {
  backend "azurerm" {
    key = "terraform.tfstate"
  }
}

# used as a random slug for each resource name
resource "random_string" "rand" {
  length  = 4
  special = false
  number  = false
  upper   = false
}

locals {
  prefix = format("%s-%s-%s", var.service_name, var.env, random_string.rand.result)
  tags = {
    environment = var.env
    service     = var.service_name
  }
}

# Create a resource group
resource "azurerm_resource_group" "rg" {
  name     = format("%s-rg", local.prefix)
  location = var.location
}