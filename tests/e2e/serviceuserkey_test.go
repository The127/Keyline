//go:build e2e

package e2e

import (
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Service user key endpoints ["+backend.name+"]", Ordered, func() {
			var h *harness
			var serviceUserId = serviceUserUsername

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("associates a public key with a caller-supplied kid and then removes it", func() {
				// Create a fresh service user to attach keys to, so we do not clobber the
				// harness's default service user.
				suId, err := h.Client().User().CreateServiceUser(h.Ctx(), "key-flow-user-"+backend.name)
				Expect(err).ToNot(HaveOccurred())
				Expect(suId).ToNot(BeZero())

				wantKid := "e2e-caller-kid-" + backend.name
				resp, err := h.Client().User().AssociateServiceUserPublicKey(h.Ctx(), suId, api.AssociateServiceUserPublicKeyRequestDto{
					PublicKey: "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAX3J/Yilw4CTcsOVW0BBasQwY9wuYwcJZkJliqAhNa5s=\n-----END PUBLIC KEY-----\n",
					Kid:       utils.Ptr(wantKid),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Kid).To(Equal(wantKid))

				Expect(h.Client().User().RemoveServiceUserPublicKey(h.Ctx(), suId, resp.Kid)).To(Succeed())

				_ = serviceUserId
			})

			It("associates a public key without a kid and server generates one", func() {
				suId, err := h.Client().User().CreateServiceUser(h.Ctx(), "key-flow-autokid-user-"+backend.name)
				Expect(err).ToNot(HaveOccurred())

				resp, err := h.Client().User().AssociateServiceUserPublicKey(h.Ctx(), suId, api.AssociateServiceUserPublicKeyRequestDto{
					PublicKey: "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAX3J/Yilw4CTcsOVW0BBasQwY9wuYwcJZkJliqAhNa5t=\n-----END PUBLIC KEY-----\n",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Kid).ToNot(BeEmpty())
			})
		})
	}
}
