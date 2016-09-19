// Package store is a collection of composable interfaces that are can be implemented
// in order to build a storage driver
package store

import (
	"net/http"

	"github.com/EtixLabs/go-json-spec-handler"
	"golang.org/x/net/context"
)

// CRUD is a resource controller interface.
type CRUD interface {
	Save(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType)
	Get(ctx context.Context, id string) (*jsh.Object, jsh.ErrorType)
	List(ctx context.Context) (jsh.List, jsh.ErrorType)
	Update(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType)
	Delete(ctx context.Context, id string) jsh.ErrorType
}

// Save a new resource to storage.
type Save func(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType)

// Get a specific instance of a resource by id from storage.
type Get func(ctx context.Context, id string) (*jsh.Object, jsh.ErrorType)

// List all instances of a resource from storage.
type List func(ctx context.Context) (jsh.List, jsh.ErrorType)

// Update an existing object in storage.
type Update func(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType)

// Delete an object from storage by id.
type Delete func(ctx context.Context, id string) jsh.ErrorType

// Action is a handler that performs a specific action on a resource.
type Action func(ctx context.Context, w http.ResponseWriter, r *http.Request) (*jsh.Object, jsh.ErrorType)

// ToOne is a to-one resource relationship controller interface.
type ToOne interface {
	GetResource(ctx context.Context, id string) (*jsh.Object, jsh.ErrorType)
	Get(ctx context.Context, id string) (*jsh.IDObject, jsh.ErrorType)
	Update(ctx context.Context, id string, relationship *jsh.IDObject) (*jsh.IDObject, jsh.ErrorType)
}

// Get the relationship of a resource from storage.
type ToOneGet func(ctx context.Context, id string) (*jsh.IDObject, jsh.ErrorType)

// Update an existing relationship in storage.
type ToOneUpdate func(ctx context.Context, id string, relationship *jsh.IDObject) (*jsh.IDObject, jsh.ErrorType)

// ToMany is a to-many resource relationship controller interface.
type ToMany interface {
	ListResources(ctx context.Context, id string) (jsh.List, jsh.ErrorType)
	List(ctx context.Context, id string) (jsh.IDList, jsh.ErrorType)
	Save(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType)
	Update(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType)
	Delete(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType)
}

// List all resources related to a resource from storage.
type ToManyListResources func(ctx context.Context, id string) (jsh.List, jsh.ErrorType)

// List all relationships of a resource from storage.
type ToManyList func(ctx context.Context, id string) (jsh.IDList, jsh.ErrorType)

// Update existing relationships in storage.
type ToManyUpdate func(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType)
