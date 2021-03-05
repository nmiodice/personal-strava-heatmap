output "queue-function-name" {
  value = azurerm_function_app.func.name
}

output "api-endpoint" {
  value = format("https://%s", azurerm_app_service.api.default_site_hostname)
}
