package client

type Client interface {
	Application() ApplicationClient
}

type client struct {
	transport *Transport
}

func NewClient(baseUrl string, virtualServer string, opts ...TransportOptions) Client {
	return &client{
		transport: NewTransport(baseUrl, virtualServer, opts...),
	}
}

func (c *client) Application() ApplicationClient {
	return NewApplicationClient(c.transport)
}
