package datasource

import (
	"context"
	"github.com/fredyk/westack-go/westack/memorykv"
)

// MemoryKVConnector implements the PersistedConnector interface
type MemoryKVConnector struct {
	db    memorykv.MemoryKvDb
	dsKey string
}

func (connector *MemoryKVConnector) GetName() string {
	return "memorykv"
}

func (connector *MemoryKVConnector) Connect(parentContext context.Context) error {
	connector.db = memorykv.NewMemoryKvDb(memorykv.Options{
		Name: connector.dsKey,
	})
	return nil
}

func (connector *MemoryKVConnector) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Ping(parentCtx context.Context) error {
	// We don't need to ping memorykv
	return nil
}

func (connector *MemoryKVConnector) GetClient() interface{} {
	return connector.db
}

// NewMemoryKVConnector Factory method for MemoryKVConnector
func NewMemoryKVConnector(dsKey string) PersistedConnector {
	return &MemoryKVConnector{
		dsKey: dsKey,
	}
}
