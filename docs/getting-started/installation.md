# Installation

## Requirements

- Go 1.21 or later
- PostgreSQL 14 or later (recommended)
- Ent CLI (for code generation)

## Install CoreForge

Add CoreForge to your Go module:

```bash
go get github.com/grokify/coreforge
```

## Install Ent CLI

CoreForge uses [Ent](https://entgo.io/) for database schemas. Install the CLI:

```bash
go install entgo.io/ent/cmd/ent@latest
```

## Project Setup

### Option 1: New Project

Create a new project with CoreForge:

```bash
mkdir myapp && cd myapp
go mod init github.com/myorg/myapp

# Add CoreForge
go get github.com/grokify/coreforge

# Create your ent directory
mkdir -p ent/schema
```

### Option 2: Existing Project

Add CoreForge to an existing project:

```bash
go get github.com/grokify/coreforge
```

## Database Setup

CoreForge uses PostgreSQL by default. Create a database:

```sql
CREATE DATABASE myapp;
CREATE USER myapp WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE myapp TO myapp;
```

## Generate Ent Code

If you're extending CoreForge schemas, generate the Ent code:

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
    "github.com/grokify/coreforge/identity/ent"
)

func main() {
    // This should compile without errors
    _ = &ent.Client{}
    fmt.Println("CoreForge installed successfully!")
}
```

Run:

```bash
go run main.go
```

## Next Steps

- [Quick Start](quickstart.md) - Build your first CoreForge app
- [Configuration](configuration.md) - Configure CoreForge for your environment
