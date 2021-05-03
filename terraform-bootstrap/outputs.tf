
output "backend-state-account-name" {
  value = azurerm_storage_account.ci.name
}

output "backend-state-account-key" {
  value     = azurerm_storage_account.ci.primary_access_key
  sensitive = true
}

output "backend-state-container-name" {
  value = azurerm_storage_container.tfstate.name
}

output "acr-id" {
  value = azurerm_container_registry.acr.id
}
