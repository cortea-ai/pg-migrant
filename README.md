# Pg-migrant

```
A cli utility for db migrations

Usage:
  pg-migrant [command]

Available Commands:
  apply              Apply pending migrations
  check              Check need for rebasing and no gaps in version numbering. Requires GITHUB_TOKEN.
  completion         Generate the autocompletion script for the specified shell
  current-version    Get the current migration version of the db
  diff               Diff the current schema against the db
  help               Help about any command
  pending-migrations Print the version for each pending migration
  squash             Squash pending migrations into a single migration. Requires GITHUB_TOKEN.
```
