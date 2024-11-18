variable "postgres_user" {
  default = "postgres"
}
variable "postgres_password" {
  default = "postgres"
}
variable "postgres_host" {
  default = "localhost"
}
variable "postgres_port" {
  default = "5432"
}
variable "postgres_dbname" {
  default = "cortea"
}

locals {
  // Order matters!
  schema_files = [
    "example/schema/extensions.sql",
    "example/schema/contacts.sql",
    "example/schema/users.sql",
  ]
  github_config = {
    owner = "cortea-ai"
    repo = "pg-migrant"
    target_branch = "main"
  }
}

env "dev" {
  schema_files = local.schema_files
  migration_dir = "example/migrations"
  db_url = "postgres://${var.postgres_user}:${var.postgres_password}@${var.postgres_host}:${var.postgres_port}/${var.postgres_dbname}?search_path=public&sslmode=disable"
  github_config = local.github_config
}

env "prod" {
  schema_files = local.schema_files
  migration_dir = "example/migrations"
  db_url = "postgres://${var.postgres_user}:${var.postgres_password}@${var.postgres_host}:${var.postgres_port}/${var.postgres_dbname}?search_path=public&sslmode=disable"
  github_config = local.github_config
}
