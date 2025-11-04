package integration

import (
	"Keyline/internal/commands"
	"Keyline/mediator"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	ed25519PublicKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIP31xjXWD6Vp3VqnWS8xKzeFsGhZwnV/OvPExVBXHjYQ"
	ed25519PrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\nQyNTUxOQAAACD99cY11g+lad1ap1kvMSs3hbBoWcJ1fzrzxMVQVx42EAAAAIh9iJMufYiT\nLgAAAAtzc2gtZWQyNTUxOQAAACD99cY11g+lad1ap1kvMSs3hbBoWcJ1fzrzxMVQVx42EA\nAAAEDmse+LYEH5FGuuCy9T4UIVFG+isPPii6vXXNtah36Evf31xjXWD6Vp3VqnWS8xKzeF\nsGhZwnV/OvPExVBXHjYQAAAAAAECAwQF\n-----END OPENSSH PRIVATE KEY-----\n"
)

var _ = Describe("ServiceUser flow", Ordered, func() {
	var h *harness
	var serviceUserId uuid.UUID

	BeforeAll(func() {
		h = newIntegrationTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("should create a service user successfully", func() {
		req := commands.CreateServiceUser{
			VirtualServerName: h.VirtualServer(),
			Username:          "service-user",
		}
		response, err := mediator.Send[*commands.CreateServiceUserResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		serviceUserId = response.Id
	})

	It("should associate public key with service user", func() {
		req := commands.AssociateServiceUserPublicKey{
			VirtualServerName: h.VirtualServer(),
			ServiceUserId:     serviceUserId,
			PublicKey:         ed25519PublicKey,
		}
		_, err := mediator.Send[*commands.AssociateServiceUserPublicKeyResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should remove public key from service user", func() {
		req := commands.RemoveServiceUserPublicKey{
			VirtualServerName: h.VirtualServer(),
			ServiceUserId:     serviceUserId,
			PublicKey:         ed25519PublicKey,
		}
		_, err := mediator.Send[*commands.RemoveServiceUserPublicKeyResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
	})
})
