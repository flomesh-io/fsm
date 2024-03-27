package cli

import "github.com/flomesh-io/fsm/pkg/connector"

func (c *client) GetC2KContext() *connector.C2KContext {
	return c.c2kContext
}

func (c *client) GetK2CContext() *connector.K2CContext {
	return c.k2cContext
}

func (c *client) GetK2GContext() *connector.K2GContext {
	return c.k2gContext
}
