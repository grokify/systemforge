//go:build ignore

// This file generates JSON Schema from the Config struct.
// Run with: go generate ./...
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/grokify/coreforge/identity/coreauth"
	"github.com/invopop/jsonschema"
)

func main() {
	// Create a reflector with custom options
	r := &jsonschema.Reflector{
		DoNotReference:             true,
		ExpandedStruct:             true,
		RequiredFromJSONSchemaTags: true,
	}

	// Generate schema from Config struct
	schema := r.Reflect(&coreauth.Config{})

	// Set schema metadata
	schema.ID = "https://github.com/grokify/coreforge/identity/coreauth/config.schema.json"
	schema.Title = "CoreAuth Configuration"
	schema.Description = "Configuration schema for CoreAuth OAuth 2.0 / OpenID Connect server"

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile("config.schema.json", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated config.schema.json")
}
