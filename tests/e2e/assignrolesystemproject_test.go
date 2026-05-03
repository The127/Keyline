//go:build e2e

package e2e

import (
	"errors"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("AssignRoleToUser system-project guard ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				// The harness's default service user has only the system
				// project's `admin` role -- VirtualServerAdmin perms
				// (which include RoleAssign) but NOT SystemUser. That is
				// exactly the attacker profile for this bug: a VS admin
				// trying to grant a user a privileged system-project
				// role-name.
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses AssignRoleToUser on a system-project role for a non-system caller (the bypass)", func() {
				adminRoleId := findRoleId(h, systemProjectSlug, "admin")

				// The caller is the harness service user. We don't even
				// need its UserId here because the gate must short-circuit
				// before the user-existence check; an arbitrary uuid is
				// enough to prove the policy fires.
				err := h.Client().Project().Role(systemProjectSlug).Assign(
					h.Ctx(),
					adminRoleId,
					uuid.New(),
				)
				Expect(err).To(HaveOccurred())

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected client.ApiError, got %T: %v", err, err)
				Expect(apiErr.Code).To(Equal(401),
					"AssignRoleToUser on system project must be rejected with 401 -- "+
						"otherwise an admin can grant `system-admin` (or any "+
						"other system-project role) to themselves and self-"+
						"promote to SystemAdmin via the JWT roles claim")
			})

			It("still allows AssignRoleToUser on a non-system project role (regression check)", func() {
				// Set up a fresh project + role that the harness admin can
				// own. The harness admin has ProjectCreate / RoleCreate /
				// RoleAssign / UserCreate.
				projSlug := "non-system-project-assign-" + backend.name
				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug:        projSlug,
					Name:        "Non System Project Assign",
					Description: "Non-system project for the AssignRole guard regression test.",
				})
				Expect(err).ToNot(HaveOccurred())

				roleResp, err := h.Client().Project().Role(projSlug).Create(h.Ctx(), api.CreateRoleRequestDto{
					Name:        "assign-me",
					Description: "Role for the AssignRole regression check.",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(roleResp.Id).ToNot(BeZero())

				userResp, err := h.Client().User().Create(h.Ctx(), api.CreateUserRequestDto{
					Username:      "assign-target-" + backend.name,
					DisplayName:   "Assign Target",
					Email:         "assign-target-" + backend.name + "@example.com",
					EmailVerified: utils.Ptr(true),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(userResp.Id).ToNot(BeZero())

				err = h.Client().Project().Role(projSlug).Assign(
					h.Ctx(),
					roleResp.Id,
					userResp.Id,
				)
				Expect(err).ToNot(HaveOccurred(),
					"the system-project guard must NOT fire on non-system projects")
			})
		})
	}
}
