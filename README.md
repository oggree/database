# oggree/database

This module provides a unified, thread-safe database connection manager for the `oggree` ecosystem. It is built on top of [GORM](https://gorm.io/) and uses [oggree/config](https://github.com/oggree/config) for configuration management.

## Features

- **Multi-Database Support:** Out-of-the-box support for **PostgreSQL**, **MySQL**, and **SQLite**.
- **Connection Pooling:** Configured globally with sensible defaults (Max idle conns: 10, Max open conns: 30, Conn max lifetime: 10 mins).
- **Thread-safe Singletons:** Each named connection is instantiated exactly once using `sync.Once` and cached in a map.
- **Read/Write Splitting & Load Balancing:** Integrated with GORM's `dbresolver` plugin. Ready to adapt for source/replica architecture using a random load-balancing policy.
- **Configuration via `oggree/config`:** Loads dynamically from central configurations using `config.GetString`.
- **Integrated Logging:** Uses the custom `github.com/oggree/logger` for system logs, and GORM's default logger initialized at the `Info` level for SQL tracing.

## Usage

### 1. Configuration Setup

The module expects configuration properties to be loaded into the `oggree/config` library. For a given `connectionName` (e.g., `default`), the expected configuration keys will be:

```yaml
database:
  default:
    type: postgres # options: postgres, mysql, sqlite
    host: 127.0.0.1
    port: 5432
    database: my_database
    username: my_user
    password: my_password
    sslMode: disable
    replicas:
      - host: 10.0.0.2
        purpose: read # The default if omitted
      - host: 10.0.0.3
        purpose: write # Adds this to DBResolver as a write source
```

### 2. Getting a Database Instance

To retrieve the database instance from another package, call `database.GetInstance()` with the appropriate connection name. This will return a pointer to the `DatabaseClient`, from which you can access the underlying `*gorm.DB`.

```go
package main

import (
	"fmt"
	"github.com/oggree/database"
)

func main() {
	// Let's assume config is already initialized here with the proper variables loaded.
	
	dbClient := database.GetInstance("default")
	if dbClient == nil {
		panic("Failed to initialize the database connection")
	}
	
	// Use dbClient.DB (which is a *gorm.DB)
	var count int64
	dbClient.DB.Table("users").Count(&count)
	fmt.Printf("Total users: %d\n", count)
}
```

## AI & Integration Guidelines (For Further AI Use)

When extending this module or integrating it into other services:

- **Adding new DB providers:** Modify `GetSQLClient()` in `main.go` and append an `else if connectionType == "newdb"` block. Require the respective GORM driver in `go.mod`.
- **Handling Replicas:** Read-replicas are natively supported by supplying a list of hosts under the `replicas` key in the yaml config. GORM's `dbresolver` will automatically distribute read queries using the `RandomPolicy` to these replicas. Valid for both Postgres and MySQL.
- **SQLite Note:** When `type` is `sqlite`, the module auto-creates a directory `data/db/` in the working path and stores the file as `{database}.db`.
- **Concurrency & Scaling:** Rely on `GetInstance` for obtaining connections across multiple goroutines, as it safely scopes multiple database variants without generating overhead. Do not directly call `GetSQLClient` unless you explicitly want to instantiate a fresh connection rather than reusing the singleton.

## Module Info
- **Dependencies:** `gorm.io/gorm`, `gorm.io/driver/postgres`, `gorm.io/driver/mysql`, `gorm.io/driver/sqlite`, `github.com/oggree/config`, `github.com/oggree/logger`
- **Go Version:** `>= 1.25.0`
