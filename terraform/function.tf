# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_insights
resource "azurerm_application_insights" "ai" {
  name                = format("%s-ai", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  application_type    = "other"
}

# # https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/app_service_plan
# resource "azurerm_app_service_plan" "function-asp" {
#   name                = format("%s-asp", local.prefix)
#   location            = azurerm_resource_group.rg.location
#   resource_group_name = azurerm_resource_group.rg.name
#   kind                = "linux"
#   reserved = true

#   sku {
#     tier = "PremiumV3"
#     size = "P1v3"
#   }
# }

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/app_service_plan
resource "azurerm_app_service_plan" "function-asp" {
  name                = format("%s-asp", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  kind                = "FunctionApp"
  reserved            = true

  sku {
    tier = "Dynamic"
    size = "Y1"
  }
}

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/function_app
resource "azurerm_function_app" "func" {
  name                       = format("%s-queue-function", local.prefix)
  location                   = azurerm_resource_group.rg.location
  resource_group_name        = azurerm_resource_group.rg.name
  app_service_plan_id        = azurerm_app_service_plan.function-asp.id
  storage_account_name       = azurerm_storage_account.sa.name
  storage_account_access_key = azurerm_storage_account.sa.primary_access_key
  os_type                    = "linux"
  version                    = "~3"

  site_config {
    linux_fx_version = "PYTHON|3.8"
  }

  app_settings = {
    APPINSIGHTS_INSTRUMENTATIONKEY : azurerm_application_insights.ai.instrumentation_key
    FUNCTIONS_WORKER_PROCESS_COUNT : 1 # workers per host
    FUNCTIONS_WORKER_RUNTIME : "python"
    STORAGE_QUEUE_CONNECTION_STRING : azurerm_storage_account.sa.primary_connection_string
    STORAGE_CONTAINER_NAME : azurerm_storage_container.sc.name
    UPLOAD_STORAGE_CONTAINER_NAME : azurerm_storage_container.sc-public.name
    STORAGE_ACCOUNT_NAME : azurerm_storage_account.sa.name
    STORAGE_ACCOUNT_KEY : azurerm_storage_account.sa.primary_access_key
    STORAGE_MAX_WORKERS : 16
    DB_HOST : azurerm_postgresql_server.dbserver.fqdn
    DB_NAME : azurerm_postgresql_database.db.name
    DB_USER : azurerm_postgresql_server.dbserver.administrator_login
    DB_PASS : random_password.db_passwd.result
  }
}
