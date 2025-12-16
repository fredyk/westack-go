package tests

import (
	wst "github.com/fredyk/westack-go/v3/common"
	"github.com/fredyk/westack-go/v3/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Global test variables
var (
	noteModel     model.Model
	accountModel  model.Model
	customerModel model.Model
	orderModel    model.Model
	storeModel    model.Model
	footerModel   model.Model
	imageModel    model.Model
	appModel      model.Model
	headerModel   model.Model

	systemContext *model.EventContext
	userId        primitive.ObjectID
	noteId        primitive.ObjectID
)

func init() {
	// System context for tests - bypasses permissions
	systemContext = &model.EventContext{
		Bearer: &model.BearerToken{
			Account: &model.BearerAccount{System: true},
		},
	}

	// Mock IDs for tests
	userId, _ = primitive.ObjectIDFromHex("5f9f1b5b9b9b9b9b9b9b9b9c")
	noteId, _ = primitive.ObjectIDFromHex("5f9f1b5b9b9b9b9b9b9b9b9b")

	// TODO: Initialize models when datasource is available
	// For now, tests will need to skip or mock
}

// CreateMockModel creates a minimal mock model for testing
func CreateMockModel(name string) model.Model {
	config := &model.Config{
		Name: name,
		Properties: map[string]model.Property{
			"title": {
				Type: "string",
			},
			"content": {
				Type: "string",
			},
		},
	}

	models := &map[string]model.Model{}
	m := model.New(config, models)
	return m.(model.Model)
}

// MockDatasource creates a simple in-memory datasource for testing
// This is a placeholder until we implement proper mocking
type MockDatasource struct {
	data map[string]map[string]wst.M
}

func NewMockDatasource() *MockDatasource {
	return &MockDatasource{
		data: make(map[string]map[string]wst.M),
	}
}

func (ds *MockDatasource) Create(collection string, doc *wst.M) (*wst.M, error) {
	if ds.data[collection] == nil {
		ds.data[collection] = make(map[string]wst.M)
	}

	// Generate ID if not present
	if (*doc)["_id"] == nil {
		(*doc)["_id"] = primitive.NewObjectID()
	}

	id := (*doc)["_id"].(primitive.ObjectID).Hex()
	ds.data[collection][id] = *doc
	return doc, nil
}
