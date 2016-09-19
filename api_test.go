package jshapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/EtixLabs/go-json-spec-handler"
	"github.com/EtixLabs/go-json-spec-handler/client"
	. "github.com/smartystreets/goconvey/convey"
)

const testResourceType = "bars"

func TestAPI(t *testing.T) {

	Convey("API Tests", t, func() {
		api := New("api")
		So(api.prefix, ShouldEqual, "/api")

		server := httptest.NewServer(api)
		baseURL := server.URL + api.prefix

		Convey("->AddResource()", func() {
			resource := NewMockResource(testResourceType, 1, testObjAttrs)
			api.Add(resource)
			So(api.Resources[testResourceType], ShouldEqual, resource)

			Convey("should work with /<resource> routes", func() {
				_, resp, err := jsc.List(baseURL, testResourceType)

				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(err, ShouldBeNil)
			})

			Convey("should work with /<resource>/:id routes", func() {
				patchObj, err := jsh.NewObject("1", testResourceType, testObjAttrs)
				So(err, ShouldBeNil)

				_, resp, patchErr := jsc.Patch(baseURL, patchObj)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(patchErr, ShouldBeNil)
			})
		})

		Convey("->Action()", func() {
			handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) (*jsh.Object, jsh.ErrorType) {
				object := sampleObject("", testResourceType, testObjAttrs)
				return object, nil
			}
			api.Action("testAction", handler)

			Convey("should handle top level action", func() {
				doc, response, err := jsc.TopLevelAction(baseURL, "testAction", nil)
				So(err, ShouldBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(doc.Data, ShouldNotBeEmpty)
			})
		})
	})
}
