package jshapi

import (
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"

	"github.com/derekdowling/go-json-spec-handler"
	"github.com/derekdowling/go-stdlogger"
)

// Logger can be overridden with your own logger to utilize any custom features
// it might have. Interface defined here: https://github.com/derekdowling/go-stdlogger/blob/master/logger.go
var Logger std.Logger = log.New(os.Stderr, "jshapi: ", log.LstdFlags)

// SendAndLog is a jsh wrapper function that first prepares a jsh.Sendable response,
// and then handles logging 5XX errors that it encounters in the process.
func SendAndLog(ctx context.Context, w http.ResponseWriter, r *http.Request, sendable jsh.Sendable) {

	intentionalErr, isType := sendable.(*jsh.Error)
	if isType && intentionalErr.Status() >= 500 {
		Logger.Printf("Returning ISE for: %s", intentionalErr.Internal())
	}

	response, err := sendable.Prepare(r, true)

	if err != nil && response.HTTPStatus >= 500 {
		Logger.Printf("Error preparing response: %s\n", err.Internal())
	}

	sendErr := jsh.SendResponse(w, r, response)
	if sendErr != nil {
		Logger.Println(err.Error())
	}
}
