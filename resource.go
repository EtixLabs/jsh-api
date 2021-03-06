package jshapi

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"

	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"

	"golang.org/x/net/context"

	"github.com/EtixLabs/go-json-spec-handler"
	"github.com/EtixLabs/jsh-api/store"
)

const (
	post    = "POST"
	get     = "GET"
	delete  = "DELETE"
	patch   = "PATCH"
	head    = "HEAD"
	options = "OPTIONS"
	patID   = "/:id"
	patRoot = ""
)

// EnableClientGeneratedIDs is an option that allows consumers to allow for client generated IDs.
var EnableClientGeneratedIDs bool

// Route represents a resource route.
type Route struct {
	Method string
	Path   string
	Allow  bool
}

// String implements the Stringer interface for Route.
func (r Route) String() string {
	return fmt.Sprintf("%-7s - %s", r.Method, r.Path)
}

/*
Resource holds the necessary state for creating a REST API endpoint for a
given resource type. Will be accessible via `/<type>`.

Using NewCRUDResource you can generate a generic CRUD handler for a
JSON Specification Resource end point. If you wish to only implement a subset
of these endpoints that is also available through NewResource() and manually
registering storage handlers via .Post(), .Get(), .List(), .Patch(), and .Delete():

Besides the built in registration helpers, it isn't recommended, but you can add
your own routes using the goji.Mux API:

	func searchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		name := pat.Param(ctx, "name")
		fmt.Fprintf(w, "Hello, %s!", name)
	}

	resource := jshapi.NewCRUDResource("users", userStorage)
	// creates /users/search/:name
	resource.HandleC(pat.New("search/:name"), searchHandler)
*/
type Resource struct {
	*goji.Mux
	// The singular name of the resource type("user", "post", etc)
	Type string
	// Routes is a list of routes registered to the resource
	Routes []Route
	// Map of relationships
	Relationships map[string]Relationship
}

/*
NewResource is a resource constructor that makes no assumptions about routes
that you'd like to implement, but still provides some basic utilities for
managing routes and handling API calls.

The prefix parameter causes all routes created within the resource to be prefixed.
*/
func NewResource(resourceType string) *Resource {
	return &Resource{
		// Mux is a goji.SubMux, inherits context from parent Mux
		Mux: goji.SubMux(),
		// Type of the resource, makes no assumptions about plurality
		Type:          resourceType,
		Relationships: map[string]Relationship{},
		// A list of registered routes used for the OPTIONS HTTP method
		Routes: []Route{},
	}
}

// NewCRUDResource generates a resource
func NewCRUDResource(resourceType string, storage store.CRUD) *Resource {
	resource := NewResource(resourceType)
	resource.CRUD(storage)
	return resource
}

/*
CRUD is syntactic sugar and a shortcut for registering all JSON API CRUD
routes for a compatible storage implementation:

Registers handlers for:
	GET    /resource
	POST   /resource
	GET    /resource/:id
	DELETE /resource/:id
	PATCH  /resource/:id
*/
func (res *Resource) CRUD(storage store.CRUD) {
	res.PartialCRUD(storage, "")
}

// PartialCRUD registers all CRUD routes with OPTIONS and HEAD support.
// It provides a handler that sends a 405 response for methods contained in the disallow parameter.
// Since GET is always allowed, the supported parameters are POST,PATCH,DELETE.
func (res *Resource) PartialCRUD(storage store.CRUD, disallow string) {
	res.Options(patRoot)
	res.List(storage.List, true)
	res.Post(storage.Save, !strings.Contains(disallow, post))
	res.Options(patID)
	res.Get(storage.Get, true)
	res.Patch(storage.Update, !strings.Contains(disallow, patch))
	res.Delete(storage.Delete, !strings.Contains(disallow, delete))
}

/*
ToOne is syntactic sugar for registering all JSON API routes for a to-one relationship:

Registers handlers for:
	GET    /resource/:id/relationship
	GET    /resource/:id/relationships/relationship
	PATCH  /resource/:id/relationships/relationship

CRUD actions on a specific relationship "resourceType" object should be performed
via it's own top level /<resourceType> jsh-api handler as per JSONAPI specification.
*/
func (res *Resource) ToOne(relationship string, storage store.ToOne) {
	res.PartialToOne(relationship, storage, "")
}

// PartialToOne registers to-one relationships routes with OPTIONS and HEAD support.
// It provides a handler that sends a 405 response for methods contained in the disallow parameter.
// Since GET is always allowed, the only supported parameter is PATCH.
func (res *Resource) PartialToOne(relationship string, storage store.ToOne, disallow string) {
	matcher := fmt.Sprintf("%s/%s", patID, relationship)
	res.Options(matcher)
	res.GetRelated(storage.GetResource, matcher, true)

	relationshipMatcher := fmt.Sprintf("%s/relationships/%s", patID, relationship)
	res.Options(relationshipMatcher)
	res.GetRelationship(storage.Get, relationshipMatcher, true)
	res.PatchOne(storage.Update, relationshipMatcher, !strings.Contains(disallow, patch))

	res.Relationships[relationship] = ToOne
}

/*
ToMany is syntactic sugar for registering all JSON API routes for a to-many relationship:

Registers handlers for:
	GET    /resource/:id/relationship
	GET    /resource/:id/relationships/relationship
	PATCH  /resource/:id/relationships/relationship
	POST   /resource/:id/relationships/relationship
	DELETE /resource/:id/relationships/relationship

CRUD actions on a specific relationship "resourceType" object should be performed
via it's own top level /<resourceType> jsh-api handler as per JSONAPI specification.
*/
func (res *Resource) ToMany(relationship string, storage store.ToMany) {
	res.PartialToMany(relationship, storage, "")
}

// PartialToMany registers to-many relationships routes with OPTIONS and HEAD support.
// It provides a handler that sends a 405 response for methods contained in the disallow parameter.
// Since GET is always allowed, the supported parameters are POST,PATCH,DELETE.
func (res *Resource) PartialToMany(relationship string, storage store.ToMany, disallow string) {
	// GET /resources/:id/<relationship>
	matcher := fmt.Sprintf("%s/%s", patID, relationship)
	res.Options(matcher)
	res.ListRelated(storage.ListResources, matcher, true)

	// GET /resources/:id/relationships/<relationship>
	relationshipMatcher := fmt.Sprintf("%s/relationships/%s", patID, relationship)
	res.Options(relationshipMatcher)
	res.ListRelationships(storage.List, relationshipMatcher, true)
	res.PostMany(storage.Save, relationshipMatcher, !strings.Contains(disallow, post))
	res.PatchMany(storage.Update, relationshipMatcher, !strings.Contains(disallow, patch))
	res.DeleteMany(storage.Delete, relationshipMatcher, !strings.Contains(disallow, delete))

	res.Relationships[relationship] = ToMany
}

// Action adds to the resource a custom action of the form:
// POST /resources/:id/<action>
func (res *Resource) Action(action string, storage store.Action, allow bool) {
	matcher := path.Join(patID, action)

	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.actionHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Post(matcher), handler)
	res.addRoute(post, matcher, allow)
}

// Options registers a `OPTIONS /resource` handler for the resource.
func (res *Resource) Options(pattern string) {
	res.HandleFuncC(
		pat.Options(pattern),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.optionsHandler(ctx, w, r)
		},
	)

	res.addRoute(options, pattern, true)
}

// Post registers a `POST /resource` handler for the resource.
func (res *Resource) Post(storage store.Save, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.postHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Post(patRoot), handler)
	res.addRoute(post, patRoot, allow)
}

// Get registers a `GET /resource/:id` handler for the resource.
func (res *Resource) Get(storage store.Get, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.fetchHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(patID), handler)
	res.addRoute(head, patID, allow)
	res.addRoute(get, patID, allow)
}

// List registers a `GET /resource` handler for the resource.
func (res *Resource) List(storage store.List, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.listHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(patRoot), handler)
	res.addRoute(head, patRoot, allow)
	res.addRoute(get, patRoot, allow)
}

// Patch registers a `PATCH /resource/:id` handler for the resource.
func (res *Resource) Patch(storage store.Update, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.patchHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Patch(patID), handler)
	res.addRoute(patch, patID, allow)
}

// Delete registers a `DELETE /resource/:id` handler for the resource.
func (res *Resource) Delete(storage store.Delete, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.deleteHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Delete(patID), handler)
	res.addRoute(delete, patID, allow)
}

// ToOne relationship

// GetRelated registers a `GET /resources/:id/<relationship>` handler for the resource relationship.
func (res *Resource) GetRelated(storage store.Get, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.fetchHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(matcher), handler)
	res.addRoute(head, matcher, allow)
	res.addRoute(get, matcher, allow)
}

// GetRelationship registers a `GET /resources/:id/relationships/<relationship>` handler for the resource relationship.
func (res *Resource) GetRelationship(storage store.ToOneGet, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.fetchIDHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(matcher), handler)
	res.addRoute(head, matcher, allow)
	res.addRoute(get, matcher, allow)
}

// PatchOne registers a `PATCH /resources/:id/relationships/<relationship>` handler for the resource relationship.
func (res *Resource) PatchOne(storage store.ToOneUpdate, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.patchOneHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Patch(matcher), handler)
	res.addRoute(patch, matcher, allow)
}

// ToMany relationship

// ListRelated registers a `GET /resources/:id/<relationship>` handler for the resource relationships.
func (res *Resource) ListRelated(storage store.ToManyListResources, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.listManyHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(matcher), handler)
	res.addRoute(head, matcher, allow)
	res.addRoute(get, matcher, allow)
}

// ListRelationships registers a `GET /resources/:id/relationships/<relationship>` handler for the resource relationships.
func (res *Resource) ListRelationships(storage store.ToManyList, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.listIDHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Get(matcher), handler)
	res.addRoute(head, matcher, allow)
	res.addRoute(get, matcher, allow)
}

// PatchMany registers a `PATCH /resources/:id/relationships/<relationship>` handler for the resource relationships.
func (res *Resource) PatchMany(storage store.ToManyUpdate, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.patchManyHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Patch(matcher), handler)
	res.addRoute(patch, matcher, allow)
}

// PostMany registers a `POST /resources/:id/relationships/<relationship>` handler for the resource relationships.
func (res *Resource) PostMany(storage store.ToManyUpdate, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.updateManyHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Post(matcher), handler)
	res.addRoute(post, matcher, allow)
}

// DeleteMany registers a `DELETE /resources/:id/relationships/<relationship>` handler for the resource relationships.
func (res *Resource) DeleteMany(storage store.ToManyUpdate, matcher string, allow bool) {
	var handler = res.notAllowedHandler
	if allow {
		handler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			res.updateManyHandler(ctx, w, r, storage)
		}
	}

	res.HandleFuncC(pat.Delete(matcher), handler)
	res.addRoute(delete, matcher, allow)
}

// notAllowedHandler returns a 405 response with the Allow header.
func (res *Resource) notAllowedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Allow", res.allowHeader(ctx, r))
	w.Header().Add("Content-Type", jsh.ContentType)
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// OPTIONS
func (res *Resource) optionsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Allow", res.allowHeader(ctx, r))
	w.Header().Add("Content-Type", jsh.ContentType)
	w.WriteHeader(http.StatusOK)
}

// POST /resources
func (res *Resource) postHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Save) {
	parsedObject, parseErr := jsh.ParseObject(r)
	if parseErr != nil && reflect.ValueOf(parseErr).IsNil() == false {
		SendHandler(ctx, w, r, parseErr)
		return
	}

	if !EnableClientGeneratedIDs && parsedObject.ID != "" {
		SendHandler(ctx, w, r, jsh.ForbiddenError("Client-generated IDs are unsupported"))
		return
	}

	object, err := storage(ctx, parsedObject)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, object)
}

// GET /resources/:id
func (res *Resource) fetchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Get) {
	id := pat.Param(ctx, "id")

	object, err := storage(ctx, id)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, object)
}

// GET /resources
func (res *Resource) listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.List) {
	list, err := storage(ctx)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, list)
}

// PATCH /resources/:id
func (res *Resource) patchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Update) {
	parsedObject, parseErr := jsh.ParseObject(r)
	if parseErr != nil && reflect.ValueOf(parseErr).IsNil() == false {
		SendHandler(ctx, w, r, parseErr)
		return
	}

	id := pat.Param(ctx, "id")
	if id != parsedObject.ID {
		SendHandler(ctx, w, r, jsh.ConflictError("", parsedObject.ID))
		return
	}

	object, err := storage(ctx, parsedObject)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, object)
}

// DELETE /resources/:id
func (res *Resource) deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Delete) {
	id := pat.Param(ctx, "id")

	err := storage(ctx, id)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /resources/:id/<action>
func (res *Resource) actionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Action) {
	response, err := storage(ctx, w, r)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	// NOTE: Explicitly set status to 200 to avoid automatically setting it to 201 (default for POST)
	if response != nil && response.Status == 0 {
		response.Status = 200
	}
	SendHandler(ctx, w, r, response)
}

// PATCH /resources/:id/relationships/<relationship> for a to-one relationship
func (res *Resource) patchOneHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToOneUpdate) {
	relationship, parseErr := jsh.ParseRelationship(r)
	if parseErr != nil {
		SendHandler(ctx, w, r, parseErr)
		return
	}

	id := pat.Param(ctx, "id")
	relationship, err := storage(ctx, id, relationship)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, relationship)
}

// GET /resources/:id/relationships/<relationship>
func (res *Resource) fetchIDHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToOneGet) {
	id := pat.Param(ctx, "id")

	object, err := storage(ctx, id)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, object)
}

// GET /resources/:id/<relationship>
func (res *Resource) listManyHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToManyListResources) {
	id := pat.Param(ctx, "id")

	list, err := storage(ctx, id)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, list)
}

// GET /resources/:id/relationships/<relationship>
func (res *Resource) listIDHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToManyList) {
	id := pat.Param(ctx, "id")

	list, err := storage(ctx, id)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, list)
}

// PATCH /resources/:id/relationships/<relationship> for a to-many relationship
func (res *Resource) patchManyHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToManyUpdate) {
	list, parseErr := jsh.ParseRelationshipList(r)
	if parseErr != nil {
		SendHandler(ctx, w, r, parseErr)
		return
	}

	id := pat.Param(ctx, "id")
	list, err := storage(ctx, id, list)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, list)
}

// POST/DELETE /resources/:id/relationships/<relationship> for a to-many relationship
func (res *Resource) updateManyHandler(ctx context.Context, w http.ResponseWriter,
	r *http.Request, storage store.ToManyUpdate) {
	list, parseErr := jsh.ParseRelationshipList(r)
	if parseErr != nil {
		SendHandler(ctx, w, r, parseErr)
		return
	}

	if len(list) == 0 {
		SendHandler(ctx, w, r, jsh.BadRequestError("Invalid document", "Missing description of changes"))
		return
	}

	id := pat.Param(ctx, "id")
	list, err := storage(ctx, id, list)
	if err != nil && reflect.ValueOf(err).IsNil() == false {
		SendHandler(ctx, w, r, err)
		return
	}

	SendHandler(ctx, w, r, list)
}

// addRoute adds the new method and route to a route Tree for debugging and
// informational purposes.
func (res *Resource) addRoute(method string, route string, allow bool) {
	res.Routes = append(res.Routes, Route{
		Method: method,
		Path:   fmt.Sprintf("/%s%s", res.Type, route),
		Allow:  allow,
	})
}

// RouteTree prints a recursive route tree based on what the resource, and
// all subresources have registered
func (res *Resource) RouteTree() string {
	var routes string
	for _, route := range res.Routes {
		routes = fmt.Sprintf("%s\n%s", routes, route)
	}
	return routes
}

// allowHeader generates the Allow header value for the resource at the given request path.
func (res *Resource) allowHeader(ctx context.Context, r *http.Request) string {
	var methods, sep string
	for _, route := range res.Routes {
		ctx = pattern.SetPath(ctx, r.URL.Path)
		if route.Allow && pat.New(route.Path).Match(ctx, r) != nil {
			methods = fmt.Sprint(methods, sep, route.Method)
			sep = ","
		}
	}
	return methods
}
