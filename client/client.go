package client

type Client interface {
	VirtualServer() VirtualServerClient
	User() UserClient
	Oidc() OidcClient
	Project() ProjectClient
}

type client struct {
	transport *Transport
}

func NewClient(baseUrl string, virtualServer string, opts ...TransportOptions) Client {
	return &client{
		transport: NewTransport(baseUrl, virtualServer, opts...),
	}
}

func (c *client) VirtualServer() VirtualServerClient {
	return NewVirtualServerClient(c.transport)
}

func (c *client) User() UserClient {
	return NewUserClient(c.transport)
}

func (c *client) Oidc() OidcClient {
	return NewOidcClient(c.transport)
}

func (c *client) Project() ProjectClient {
	return NewProjectClient(c.transport)
}
