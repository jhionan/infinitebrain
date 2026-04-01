// Atlas project configuration.
// Source of truth for schema is db/schema/*.hcl
// Run `atlas schema apply --env local` to apply changes.
// Run `atlas schema diff --env local` to inspect drift.

variable "database_url" {
  type    = string
  default = getenv("DATABASE_URL")
}

variable "test_database_url" {
  type    = string
  default = getenv("TEST_DATABASE_URL")
}

env "local" {
  src = "file://db/schema"
  url = var.database_url
  // Dev database used by Atlas for temporary state during diff computation.
  dev = "docker://postgres/18/atlas_dev?search_path=public"
}

env "test" {
  src = "file://db/schema"
  url = var.test_database_url
  dev = "docker://postgres/18/atlas_dev?search_path=public"
}
