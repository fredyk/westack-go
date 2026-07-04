package graphql

import (
	"testing"
)

func TestBindGraphQLOperation_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLOperation returned nil")
	}
}

func TestBindGraphQLOperationWithContext_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Error("BindGraphQLOperationWithContext returned nil")
	}
}

func TestBindGraphQLQuery_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLQuery returned nil")
	}
}

func TestBindGraphQLMutation_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLMutation returned nil")
	}
}
