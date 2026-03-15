package marketplace

import (
	_ "embed"
)

//go:embed schema/marketplace.zed
var MarketplaceSchema string

// MergeSchema combines the marketplace schema with an application-specific schema.
// The app schema should define app-specific resource types that reference
// marketplace types (listing, license, etc.).
func MergeSchema(appSchema string) string {
	return MarketplaceSchema + "\n\n" + appSchema
}
