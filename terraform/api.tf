# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/app_service_plan
resource "azurerm_app_service_plan" "api-asp" {
  name                = format("%s-api-asp", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  kind                = "Linux"
  reserved            = true

  sku {
    tier = "Basic"
    size = "B1"
  }
}

locals {
  acr_id_split = split("/", var.acr_id)
  acr_name     = element(local.acr_id_split, length(local.acr_id_split) - 1)
}

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/app_service
resource "azurerm_app_service" "api" {
  name                = format("%s-api-app", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  app_service_plan_id = azurerm_app_service_plan.api-asp.id

  identity {
    type = "SystemAssigned"
  }

  site_config {
    linux_fx_version = "DOCKER|"
  }

  app_settings = {
    PORT : 8080
    APPINSIGHTS_INSTRUMENTATIONKEY : azurerm_application_insights.ai.instrumentation_key
    WEBSITES_ENABLE_APP_SERVICE_STORAGE : false
    HTTP_CLIENT_TIMEOUT_SECONDS : "5s"

    STRAVA_CLIENT_ID : var.strava_client_id
    STRAVA_CLIENT_SECRET : var.strava_client_secret
    STRAVA_MAX_DOWNLOAD_WORKERS : 1

    DB_HOST : azurerm_postgresql_server.dbserver.fqdn
    DB_NAME : azurerm_postgresql_database.db.name
    DB_USER : azurerm_postgresql_server.dbserver.administrator_login
    DB_PASS : random_password.db_passwd.result
    DB_PORT : 5432
    DB_SSLMODE : "require"

    STORAGE_CONTAINER_NAME : azurerm_storage_container.sc.name
    UPLOAD_STORAGE_CONTAINER_NAME : azurerm_storage_container.sc-public.name
    STORAGE_ACCOUNT_NAME : azurerm_storage_account.sa.name
    STORAGE_ACCOUNT_KEY : azurerm_storage_account.sa.primary_access_key
    STORAGE_MAX_WORKERS : 16
    STORAGE_QUEUE_NAME : azurerm_storage_queue.sq.name
    QUEUE_BATCH_SIZE : 250

    MIN_TILE_ZOOM : 2
    MAX_TILE_ZOOM : 20
    GOOGLE_MAPS_API_KEY : var.google_maps_api_key

    DOCKER_REGISTRY_SERVER_URL : format("https://%s.azurecr.io", local.acr_name)
    DOCKER_REGISTRY_SERVER_USERNAME : format("@Microsoft.KeyVault(SecretUri=%s)", azurerm_key_vault_secret.acr-pull-sp.id)
    DOCKER_REGISTRY_SERVER_PASSWORD : format("@Microsoft.KeyVault(SecretUri=%s)", azurerm_key_vault_secret.acr-pull-passwd.id)
  }

  lifecycle {
    ignore_changes = [
      site_config[0].linux_fx_version
    ]
  }
}

resource "azurerm_monitor_diagnostic_setting" "api-log-settings" {
  name               = "logs-and-metrics-capture"
  target_resource_id = azurerm_app_service.api.id
  storage_account_id = azurerm_storage_account.sa.id

  log {
    category = "AppServiceHTTPLogs"
    retention_policy {
      enabled = true
      days    = 180
    }
  }

  log {
    category = "AppServiceConsoleLogs"
    retention_policy {
      enabled = true
      days    = 180
    }
  }

  log {
    category = "AppServiceAppLogs"
    retention_policy {
      enabled = true
      days    = 180
    }
  }

  metric {
    category = "AllMetrics"

    retention_policy {
      enabled = true
      days    = 180
    }
  }
}
