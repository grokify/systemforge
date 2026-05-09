# Installation

## Requirements

- Go 1.21 or later
- PostgreSQL 14 or later (recommended)
- Ent CLI (for code generation)

## Install SystemForge

Add SystemForge to your Go module:

```bash
go get github.com/grokify/systemforge
```

## Install Ent CLI

SystemForge uses [Ent](https://entgo.io/) for database schemas. Install the CLI:

```bash
go install entgo.io/ent/cmd/ent@latest
```

## Project Setup

### Option 1: New Project

Create a new project with SystemForge:

```bash
mkdir myapp && cd myapp
go mod init github.com/myorg/myapp

# Add SystemForge
go get github.com/grokify/systemforge

# Create your ent directory
mkdir -p ent/schema
```

### Option 2: Existing Project

Add SystemForge to an existing project:

```bash
go get github.com/grokify/systemforge
```

## Database Setup

SystemForge uses PostgreSQL by default. Create a database:

```sql
CREATE DATABASE myapp;
CREATE USER myapp WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE myapp TO myapp;
```

## Generate Ent Code

If you're extending SystemForge schemas, generate the Ent code:

```bash
cd ent
go generate ./...
```

## Verify Installation

Create a simple test to verify the installation:

```go
package main

import (
    "fmt"
    "github.com/grokify/systemforge/identity/ent"
)

func main() {
    // This should compile without errors
    _ = &ent.Client{}
    fmt.Println("SystemForge installed successfully!")
}
```

Run:

```bash
go run main.go
```

## Next Steps

- [Quick Start](quickstart.md) - Build your first SystemForge app
- [Configuration](configuration.md) - Configure SystemForge for your environment
