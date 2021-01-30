data "azurerm_client_config" "current" {}

# https://www.terraform.io/docs/providers/azurerm/r/key_vault.html
resource "azurerm_key_vault" "kv" {
  name                = format("%s-kv", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
}

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy
resource "azurerm_key_vault_access_policy" "sp-manage" {
  key_vault_id            = azurerm_key_vault.kv.id
  tenant_id               = data.azurerm_client_config.current.tenant_id
  object_id               = data.azurerm_client_config.current.object_id
  key_permissions         = ["create", "delete", "get", "list", "update"]
  secret_permissions      = ["set", "delete", "get", "list"]
  certificate_permissions = ["create", "delete", "get", "list"]
}

resource "azurerm_key_vault_access_policy" "api-get" {
  key_vault_id       = azurerm_key_vault.kv.id
  tenant_id          = data.azurerm_client_config.current.tenant_id
  object_id          = azurerm_app_service.api.identity.0.principal_id
  secret_permissions = ["get"]
}


# https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/application
resource "azuread_application" "acr" {
  name = format("acr-pull-%s", random_string.rand.result)
}

# https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal
resource "azuread_service_principal" "acr" {
  application_id = azuread_application.acr.application_id
}

resource "random_password" "acr" {
  length  = 35
  upper   = true
  lower   = true
  special = false
}

# https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal_password
resource "azuread_service_principal_password" "acr" {
  service_principal_id = azuread_service_principal.acr.id
  value                = random_password.acr.result
  end_date_relative    = "2400h"
}

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment
resource "azurerm_role_assignment" "acr_pull" {
  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = azuread_service_principal.acr.id
}

# https://www.terraform.io/docs/providers/azurerm/r/key_vault_secret.html
resource "azurerm_key_vault_secret" "acr-pull-sp" {
  depends_on   = [azurerm_key_vault_access_policy.sp-manage]
  name         = "acr-pull-sp"
  value        = azuread_service_principal.acr.application_id
  key_vault_id = azurerm_key_vault.kv.id
}

resource "azurerm_key_vault_secret" "acr-pull-passwd" {
  depends_on   = [azurerm_key_vault_access_policy.sp-manage]
  name         = "acr-pull-passwd"
  value        = random_password.acr.result
  key_vault_id = azurerm_key_vault.kv.id
}