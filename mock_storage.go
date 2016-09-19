package jshapi

import (
	"log"
	"strconv"

	"github.com/EtixLabs/go-json-spec-handler"
	"golang.org/x/net/context"
)

// MockStorage allows you to mock out APIs really easily.
// It is also used internally for testing the API layer.
type MockStorage struct {
	// ResourceType is the name of the resource you are mocking i.e. "user", "comment"
	ResourceType string
	// ResourceAttributes a sample set of attributes a resource object should have
	// used by GET /resources and GET /resources/:id
	ResourceAttributes interface{}
	// ListCount is the number of sample objects to return in a GET /resources request
	ListCount int
}

// Save assigns a URL of 1 to the object
func (m *MockStorage) Save(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType) {
	object.ID = "1"
	return object, nil
}

// Get returns a resource with ID as specified by the request
func (m *MockStorage) Get(ctx context.Context, id string) (*jsh.Object, jsh.ErrorType) {
	return m.SampleObject(id), nil
}

// List returns a sample list
func (m *MockStorage) List(ctx context.Context) (jsh.List, jsh.ErrorType) {
	return m.SampleList(m.ListCount), nil
}

// Update does nothing
func (m *MockStorage) Update(ctx context.Context, object *jsh.Object) (*jsh.Object, jsh.ErrorType) {
	return object, nil
}

// Delete does nothing
func (m *MockStorage) Delete(ctx context.Context, id string) jsh.ErrorType {
	return nil
}

// SampleObject builds an object based on provided resource specifications
func (m *MockStorage) SampleObject(id string) *jsh.Object {
	object, err := jsh.NewObject(id, m.ResourceType, m.ResourceAttributes)
	if err != nil {
		log.Fatal(err.Error())
	}
	return object
}

// SampleList generates a sample list of resources that can be used for/against the
// mock API
func (m *MockStorage) SampleList(length int) jsh.List {
	list := jsh.List{}
	for id := 1; id <= length; id++ {
		list = append(list, m.SampleObject(strconv.Itoa(id)))
	}
	return list
}

// MockToOneStorage allows you to mock out APIs to-one relationships really easily. \
// It is also used internally for testing the API layer.
type MockToOneStorage MockStorage

// Get returns the to-one relationship resource with ID as specified by the request
func (m *MockToOneStorage) GetResource(ctx context.Context, id string) (*jsh.Object, jsh.ErrorType) {
	return m.SampleObject(id), nil
}

// GetRelationship returns the to-one relationship ID object as specified by the request
func (m *MockToOneStorage) Get(ctx context.Context, id string) (*jsh.IDObject, jsh.ErrorType) {
	return m.SampleIDObject(id), nil
}

// Update does nothing
func (m *MockToOneStorage) Update(ctx context.Context, id string,
	relationship *jsh.IDObject) (*jsh.IDObject, jsh.ErrorType) {
	return nil, nil
}

// SampleObject builds an object based on provided resource specifications
func (m *MockToOneStorage) SampleObject(id string) *jsh.Object {
	object, err := jsh.NewObject(id, m.ResourceType, m.ResourceAttributes)
	if err != nil {
		log.Fatal(err.Error())
	}
	return object
}

// SampleIDObject builds an ID object based on provided resource specifications
func (m *MockToOneStorage) SampleIDObject(id string) *jsh.IDObject {
	return jsh.NewIDObject(m.ResourceType, id)
}

// MockToManyStorage allows you to mock out APIs to-many relationships really easily. \
// It is also used internally for testing the API layer.
type MockToManyStorage MockStorage

// List returns the to-many relationship resources with ID as specified by the request
func (m *MockToManyStorage) ListResources(ctx context.Context, id string) (jsh.List, jsh.ErrorType) {
	return m.SampleList(id), nil
}

// ListRelationships returns the to-many relationship ID objects as specified by the request
func (m *MockToManyStorage) List(ctx context.Context, id string) (jsh.IDList, jsh.ErrorType) {
	return m.SampleIDList(id), nil
}

// Save does nothing
func (m *MockToManyStorage) Save(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType) {
	return nil, nil
}

// Update does nothing
func (m *MockToManyStorage) Update(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType) {
	return nil, nil
}

// Delete does nothing
func (m *MockToManyStorage) Delete(ctx context.Context, id string, list jsh.IDList) (jsh.IDList, jsh.ErrorType) {
	return nil, nil
}

// SampleObject builds an object based on provided resource specifications
func (m *MockToManyStorage) SampleList(id string) jsh.List {
	object, err := jsh.NewObject(id, m.ResourceType, m.ResourceAttributes)
	if err != nil {
		log.Fatal(err.Error())
	}
	return jsh.List{object}
}

// SampleIDObject builds an ID object based on provided resource specifications
func (m *MockToManyStorage) SampleIDList(id string) jsh.IDList {
	return jsh.IDList{jsh.NewIDObject(m.ResourceType, id)}
}
