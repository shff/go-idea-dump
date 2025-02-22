package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/graphql-go/graphql"
)

func catchallResolver(p graphql.ResolveParams) (interface{}, error) {
	// Use the field name as part of the dynamic resolution
	fieldName := p.Info.FieldName
	return fmt.Sprintf("Resolved: %s", fieldName), nil
}

func main() {
	// Catchall logic for dynamic field names
	fields := graphql.Fields{}
	for _, fieldName := range []string{"bla", "ble", "blu"} {
		fields[fieldName] = &graphql.Field{
			Type:    graphql.String,
			Resolve: catchallResolver,
		}
	}

	// Create the Query type dynamically
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: fields,
	})

	// Create the schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
	})
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Execute a query
	query := `{ __schema { types { name fields { name } } } }`
	params := graphql.Params{
		Schema:        schema,
		RequestString: query,
	}

	// Execute the query
	result := graphql.Do(params)
	if len(result.Errors) > 0 {
		log.Fatalf("Query failed: %+v", result.Errors)
	}

	// Print the result as JSON
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonResult))
}

// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/graph-gophers/graphql-go"
// )

// // Define a resolver for the dynamic field
// type Resolver struct{}

// func (r *Resolver) DynamicField() *string {
// 	result := "Hello, World!"

// 	return &result
// }

// func (r *Resolver) DynamicMut() *string {
// 	result := "Hello, World?"

// 	return &result
// }

// type DynamicResolver struct{}

// func (r *DynamicResolver) Resolve(ctx context.Context, args struct{ Field string }) (string, error) {
// 	// Simulate a dynamic response based on the field name
// 	return fmt.Sprintf("Resolved: %s", args.Field), nil
// }

// func main() {
// 	// Define the schema in SDL format
// 	schemaStr := `
// 		type Query {
// 			dynamicField: String
// 		}
// 		type Mutation {
// 			dynamicMut: String
// 		}
// 	`

// 	// Parse the schema using the SDL format and link it to the resolver
// 	schema, err := graphql.ParseSchema(schemaStr, &DynamicResolver{})
// 	if err != nil {
// 		log.Fatalf("Failed to parse schema: %v", err)
// 	}

// 	// Execute a query
// 	query := `mutation { dynamicMut }`
// 	result := schema.Exec(context.TODO(), query, "", nil)

// 	// Check for errors
// 	if len(result.Errors) > 0 {
// 		log.Printf("Query execution errors: %+v", result.Errors)
// 		return
// 	}

// 	// Output the result
// 	fmt.Printf("%s\n", result.Data)
// }
