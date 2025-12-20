package graphql

import (
	"gorm.io/gorm"
)

// Resolver holds dependencies for GraphQL resolvers.
// gqlgen will wire this into generated resolvers.
type Resolver struct {
	DB *gorm.DB
}

// NewResolver constructs a Resolver with the given DB.
func NewResolver(db *gorm.DB) *Resolver {
	return &Resolver{DB: db}
}
