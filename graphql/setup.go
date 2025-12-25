package graphql

import (
	"errors"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"gorm.io/gorm"
)

var gqlHTTPHandler http.Handler

// SetupGraphQL initializes gqlgen schema and a unified handler.
func SetupGraphQL(db *gorm.DB) error {
	if db == nil {
		log.Println("[GraphQL] ERROR: Database connection is nil, cannot initialize GraphQL")
		return errors.New("database connection not initialized")
	}

	log.Println("[GraphQL] Setting up schema and handler...")
	resolver := NewResolver(db)
	schema := NewExecutableSchema(Config{Resolvers: resolver})
	server := handler.NewDefaultServer(schema)

	// Single endpoint handler: GET -> Playground, others -> GraphQL server
	gqlHTTPHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			playground.Handler("GraphQL", "/graphql").ServeHTTP(w, r)
			return
		}
		server.ServeHTTP(w, r)
	})
	log.Println("[GraphQL] Handler initialized successfully!")
	return nil
}

// GetHandler returns the HTTP handler for the GraphQL endpoint.
func GetHandler() http.Handler {
	if gqlHTTPHandler == nil {
		log.Println("[GraphQL] WARNING: Handler is nil!")
	}
	return gqlHTTPHandler
}
