package jshapi

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goji.io/pat"

	"github.com/EtixLabs/go-json-spec-handler"
	"github.com/EtixLabs/go-json-spec-handler/client"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

var testObjAttrs = map[string]string{
	"foo": "bar",
}

func TestResource(t *testing.T) {
	resource := NewMockResource(testResourceType, 2, testObjAttrs)

	api := New("")
	api.Add(resource)

	server := httptest.NewServer(api)
	baseURL := server.URL

	routeCount := len(resource.Routes)
	if routeCount != 9 {
		log.Fatalf("Invalid number of base resource routes: %d", routeCount)
	}

	Convey("Resource Tests", t, func() {

		Convey("->NewResource()", func() {

			Convey("should be agnostic to plurality", func() {
				resource := NewResource("users")
				So(resource.Type, ShouldEqual, "users")

				resource2 := NewResource("user")
				So(resource2.Type, ShouldEqual, "user")
			})
		})

		Convey("->Post()", func() {
			object := sampleObject("", testResourceType, testObjAttrs)
			doc, resp, err := jsc.Post(baseURL, object)

			So(resp.StatusCode, ShouldEqual, http.StatusCreated)
			So(err, ShouldBeNil)
			So(doc.Data[0].ID, ShouldEqual, "1")
		})

		Convey("->List()", func() {
			doc, resp, err := jsc.List(baseURL, testResourceType)

			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(err, ShouldBeNil)
			So(len(doc.Data), ShouldEqual, 2)
			So(doc.Data[0].ID, ShouldEqual, "1")
		})

		Convey("->Fetch()", func() {
			doc, resp, err := jsc.Fetch(baseURL, testResourceType, "3")

			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(err, ShouldBeNil)
			So(doc.Data[0].ID, ShouldEqual, "3")
		})

		Convey("->Patch()", func() {

			Convey("should reject requests with ID mismatch", func() {
				object := sampleObject("1", testResourceType, testObjAttrs)
				request, err := jsc.PatchRequest(baseURL, object)
				So(err, ShouldBeNil)
				// Manually replace resource ID in URL to be invalid
				request.URL.Path = strings.Replace(request.URL.Path, "1", "2", 1)
				doc, resp, err := jsc.Do(request, jsh.ObjectMode)

				So(resp.StatusCode, ShouldEqual, 409)
				So(err, ShouldBeNil)
				So(doc, ShouldNotBeNil)
			})

			Convey("should accept patch requests", func() {
				object := sampleObject("1", testResourceType, testObjAttrs)
				doc, resp, err := jsc.Patch(baseURL, object)

				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(err, ShouldBeNil)
				So(doc.Data[0].ID, ShouldEqual, "1")
			})
		})

		Convey("->Delete()", func() {
			resp, err := jsc.Delete(baseURL, testResourceType, "1")

			So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
			So(err, ShouldBeNil)
		})
	})
}

func TestActionHandler(t *testing.T) {
	resource := NewMockResource(testResourceType, 2, testObjAttrs)

	// Add our custom action
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) (*jsh.Object, jsh.ErrorType) {
		id := pat.Param(ctx, "id")
		object := sampleObject(id, testResourceType, testObjAttrs)
		return object, nil
	}
	resource.Action("testAction", handler, true)

	api := New("")
	api.Add(resource)

	server := httptest.NewServer(api)
	baseURL := server.URL

	Convey("Action Handler Tests", t, func() {

		Convey("Resource State", func() {
			So(len(resource.Routes), ShouldEqual, 10)
			So(resource.Routes[len(resource.Routes)-1].String(), ShouldEqual, "POST    - /bars/:id/testAction")
		})

		Convey("->Custom()", func() {
			doc, response, err := jsc.Action(baseURL, testResourceType, "1", "testAction", nil)

			So(err, ShouldBeNil)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
			So(doc.Data, ShouldNotBeEmpty)
		})
	})
}

func TestToOne(t *testing.T) {
	resource := NewMockResource(testResourceType, 2, testObjAttrs)
	relResourceType := "bars"
	toOne := &MockToOneStorage{
		ResourceType:       relResourceType,
		ResourceAttributes: testObjAttrs,
	}
	resource.ToOne("bar", toOne)

	api := New("")
	api.Add(resource)

	server := httptest.NewServer(api)
	baseURL := server.URL

	Convey("Relationship ToOne Tests", t, func() {

		Convey("Resource State", func() {

			Convey("should track sub-resources properly", func() {
				So(len(resource.Relationships), ShouldEqual, 1)
				So(len(resource.Routes), ShouldEqual, 16)
			})
		})

		Convey("->ToOne()", func() {

			Convey("->GetResource()", func() {
				doc, resp, err := jsc.FetchRelated(baseURL, testResourceType, "1", "bar")

				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(doc.Data[0].ID, ShouldEqual, "1")
				So(doc.Data[0].Type, ShouldEqual, relResourceType)
				So(doc.Data[0].Attributes, ShouldNotBeEmpty)
			})

			Convey("->Get()", func() {
				doc, resp, err := jsc.FetchRelationship(baseURL, testResourceType, "1", "bar")

				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(doc.Data[0].ID, ShouldEqual, "1")
				So(doc.Data[0].Type, ShouldEqual, relResourceType)
				So(doc.Data[0].Attributes, ShouldBeEmpty)
			})

			Convey("->Patch()", func() {

				Convey("should accept a single object", func() {
					object := jsh.NewIDObject(relResourceType, "1")
					doc, resp, err := jsc.PatchOne(baseURL, testResourceType, "1", "bar", object)

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})

				Convey("should accept null data", func() {
					doc, resp, err := jsc.PatchOne(baseURL, testResourceType, "1", "bar", nil)

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})
			})
		})
	})
}

func TestToMany(t *testing.T) {
	resource := NewMockResource(testResourceType, 2, testObjAttrs)

	relResourceType := "bars"
	toMany := &MockToManyStorage{
		ResourceType:       relResourceType,
		ResourceAttributes: testObjAttrs,
		ListCount:          1,
	}
	resource.ToMany(relResourceType, toMany)

	api := New("")
	api.Add(resource)

	server := httptest.NewServer(api)
	baseURL := server.URL

	Convey("Relationship ToMany Tests", t, func() {

		Convey("Resource State", func() {

			Convey("should track sub-resources properly", func() {
				So(len(resource.Relationships), ShouldEqual, 1)
				So(len(resource.Routes), ShouldEqual, 18)
			})
		})

		Convey("->ToMany()", func() {

			Convey("->ListResources()", func() {
				doc, resp, err := jsc.FetchRelated(baseURL, testResourceType, "1", relResourceType)

				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(len(doc.Data), ShouldEqual, 1)
				So(doc.Data[0].ID, ShouldEqual, "1")
				So(doc.Data[0].Type, ShouldEqual, relResourceType)
				So(doc.Data[0].Attributes, ShouldNotBeEmpty)
			})

			Convey("->List()", func() {
				doc, resp, err := jsc.FetchRelationship(baseURL, testResourceType, "1", relResourceType)

				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(len(doc.Data), ShouldEqual, 1)
				So(doc.Data[0].ID, ShouldEqual, "1")
				So(doc.Data[0].Type, ShouldEqual, relResourceType)
				So(doc.Data[0].Attributes, ShouldBeEmpty)
			})

			Convey("->Post()", func() {

				Convey("should accept a list", func() {
					object := jsh.NewIDObject(relResourceType, "1")
					doc, resp, err := jsc.PostMany(baseURL, testResourceType, "1", "bars", jsh.IDList{object})

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})

				Convey("should reject an empty list", func() {
					doc, resp, err := jsc.PostMany(baseURL, testResourceType, "1", "bars", nil)

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(doc, ShouldNotBeNil)
				})
			})

			Convey("->Patch()", func() {

				Convey("should accept a list", func() {
					object := jsh.NewIDObject(relResourceType, "1")
					doc, resp, err := jsc.PatchMany(baseURL, testResourceType, "1", "bars", jsh.IDList{object})

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})

				Convey("should accept an empty list", func() {
					doc, resp, err := jsc.PatchMany(baseURL, testResourceType, "1", "bars", nil)

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})
			})

			Convey("->Delete()", func() {
				Convey("should accept a list", func() {
					object := jsh.NewIDObject(relResourceType, "1")
					doc, resp, err := jsc.DeleteMany(baseURL, testResourceType, "1", "bars", jsh.IDList{object})

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusNoContent)
					So(doc, ShouldBeNil)
				})

				Convey("should reject an empty list", func() {
					doc, resp, err := jsc.DeleteMany(baseURL, testResourceType, "1", "bars", nil)

					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(doc, ShouldNotBeNil)
				})
			})
		})
	})
}
