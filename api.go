package jshapi

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"

	"golang.org/x/net/context"

	"goji.io"
	"goji.io/pat"

	"github.com/EtixLabs/jsh-api/store"
	"github.com/derekdowling/go-stdlogger"
	"github.com/derekdowling/goji2-logger"
)

// API is used to direct HTTP requests to resources
type API struct {
	*goji.Mux
	prefix    string
	Resources map[string]*Resource
	Debug     bool
}

/*
SendHandler allows the customization of how API responses are sent and logged. This
is used by all jshapi.Resource objects.
*/
var SendHandler = DefaultSender(log.New(os.Stderr, "jshapi: ", log.LstdFlags))

/*
New initializes a new top level API Resource without doing any additional setup.
*/
func New(prefix string) *API {
	// ensure that our top level prefix is "/" prefixed
	if !strings.HasPrefix(prefix, "/") {
		prefix = fmt.Sprintf("/%s", prefix)
	}

	// create our new API
	return &API{
		Mux:       goji.NewMux(),
		prefix:    prefix,
		Resources: map[string]*Resource{},
	}
}

/*
Default builds a new top-level API with a few out of the box additions to get people
started without needing to add a lot of extra functionality.

The most basic implementation is:

	// create a logger, the std log package works, as do most other loggers
	// std.Logger interface defined here:
	// https://github.com/derekdowling/go-stdlogger/blob/master/logger.go
	logger := log.New(os.Stderr, "jshapi: ", log.LstdFlags)

	// create the API. Specify a http://yourapi/<prefix>/ if required
	api := jshapi.Default("<prefix>", false, logger)
	api.Add(yourResource)

*/
func Default(prefix string, debug bool, logger std.Logger) *API {
	api := New(prefix)
	SendHandler = DefaultSender(logger)

	// register logger middleware
	gojilogger := gojilogger.New(logger, debug)
	api.UseC(gojilogger.Middleware)

	return api
}

// Add implements mux support for a given resource which is effectively handled as:
// pat.New("/(prefix/)resource.Plu*)
func (a *API) Add(resource *Resource) {
	// track our associated resources, will enable auto-generation docs later
	a.Resources[resource.Type] = resource

	// Because of how prefix matches work:
	// https://godoc.org/github.com/goji/goji/pat#hdr-Prefix_Matches
	// We need two separate routes,
	// /(prefix/)resources
	matcher := path.Join(a.prefix, resource.Type)
	a.Mux.HandleC(pat.New(matcher), resource)

	// And:
	// /(prefix/)resources/*
	idMatcher := path.Join(a.prefix, resource.Type, "*")
	a.Mux.HandleC(pat.New(idMatcher), resource)
}

func (a *API) Action(action string, storage store.Action) {
	matcher := path.Join(a.prefix, action)

	a.Mux.HandleFuncC(
		pat.Post(matcher),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			a.actionHandler(ctx, w, r, storage)
		},
	)
}

// POST /<action>
func (a *API) actionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, storage store.Action) {
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

// RouteTree prints out all accepted routes for the API that use jshapi implemented
// ways of adding routes through resources.
func (a *API) RouteTree() string {
	var routes string

	for _, resource := range a.Resources {
		routes = strings.Join([]string{routes, resource.RouteTree()}, "")
	}

	return routes
}
