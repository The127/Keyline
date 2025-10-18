package client

type UserClient interface {
}

func NewUserClient(transport *Transport) UserClient {
	return &userClient{
		transport: transport,
	}
}

type userClient struct {
	transport *Transport
}
