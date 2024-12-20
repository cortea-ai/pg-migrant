# Pg-migrant

```
A cli utility for db migrations

Usage:
  pg-migrant [command]

Available Commands:
  apply               Apply pending migrations
  check               Check need for rebasing and no gaps in version numbering. Requires GITHUB_TOKEN.
  clean               Clean existing database schema. Requires `allow_db_clean=true`.
  completion          Generate the autocompletion script for the specified shell
  db-last-migration   Get the last migration version of the db
  diff                Diff the current schema against the db
  help                Help about any command
  pending-migrations  Print the version for each pending migration
  repo-last-migration Get the last migration version commited to the repo
  squash              Squash pending migrations into a single migration. Requires GITHUB_TOKEN.
  version             Print the version number of pg-migrant
```
