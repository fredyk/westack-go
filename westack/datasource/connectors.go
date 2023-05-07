package datasource

import "context"

type PersistedConnector interface {
	// GetName Returns the name of the connector
	GetName() string
	// Connect Connects to the datasource
	Connect(parentContext context.Context) error
	// Disconnect Disconnects from the datasource
	Disconnect() error
	// Ping Pings the datasource
	Ping(parentCtx context.Context) error
	// GetClient Returns the client for the datasource
	GetClient() interface{}
}
