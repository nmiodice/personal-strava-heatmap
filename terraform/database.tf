# https://www.terraform.io/docs/providers/random/r/password.html
resource "random_password" "db_passwd" {
  length  = 64
  special = false
}

# https://www.terraform.io/docs/providers/azurerm/r/postgresql_server.html
resource "azurerm_postgresql_server" "dbserver" {
  name                = format("%s-dbsvr", local.prefix)
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  sku_name = var.psql_sku
  version  = 11

  storage_mb                   = 25600 # 25 GB
  backup_retention_days        = 20
  geo_redundant_backup_enabled = false # need bigger sku!
  auto_grow_enabled            = true

  administrator_login              = "psqladmin"
  administrator_login_password     = random_password.db_passwd.result
  ssl_enforcement_enabled          = true
  ssl_minimal_tls_version_enforced = "TLS1_2"
  public_network_access_enabled    = true

  tags = local.tags
}

# https://www.terraform.io/docs/providers/azurerm/r/postgresql_database.html
resource "azurerm_postgresql_database" "db" {
  name                = "main"
  resource_group_name = azurerm_resource_group.rg.name
  server_name         = azurerm_postgresql_server.dbserver.name
  charset             = "UTF8"
  collation           = "English_United States.1252"
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/postgresql_firewall_rule
resource "azurerm_postgresql_firewall_rule" "local-dev" {
  name                = "local-dev"
  resource_group_name = azurerm_resource_group.rg.name
  server_name         = azurerm_postgresql_server.dbserver.name
  start_ip_address    = chomp(data.http.myip.body)
  end_ip_address      = chomp(data.http.myip.body)
}
