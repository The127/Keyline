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

// systemProjectSlug must match repositories.NewSystemProject's hard-coded
// "system" slug; encoded here so a rename of that constant trips the test
// instead of silently passing.
const systemProjectSlug = "system"

// findRoleId locates a role in the given project by name via the role
// listing endpoint. The harness's default service user holds the `admin`
// role (VirtualServerAdmin) and so has RoleView in any project.
func findRoleId(h *harness, projectSlug, roleName string) uuid.UUID {
	roles, err := h.Client().Project().Role(projectSlug).List(h.Ctx(), client.ListRoleParams{
		Page: 0,
		Size: 100,
	})
	Expect(err).ToNot(HaveOccurred())
	for _, r := range roles.Items {
		if r.Name == roleName {
			return r.Id
		}
	}
	Fail("role '" + roleName + "' not found in project '" + projectSlug + "'")
	return uuid.Nil
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("PatchRole / DeleteRole system-project guard ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				// The harness's default service user has only the system
				// project's `admin` role -- exactly the attacker profile
				// for this bug: VirtualServerAdmin trying to mutate a
				// privileged role-name.
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses PatchRole on a system-project role for a non-system caller (the bypass)", func() {
				adminRoleId := findRoleId(h, systemProjectSlug, "admin")

				err := h.Client().Project().Role(systemProjectSlug).Patch(
					h.Ctx(),
					adminRoleId,
					api.PatchRoleRequestDto{Name: utils.Ptr("system-admin")},
				)
				Expect(err).To(HaveOccurred())

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected client.ApiError, got %T: %v", err, err)
				Expect(apiErr.Code).To(Equal(401),
					"PatchRole on system project must be rejected with 401 -- "+
						"otherwise an admin can rename `admin` to `system-admin` "+
						"and self-promote to SystemAdmin via the JWT roles claim")

				// Verify the database state was not mutated: the role's
				// name must still be `admin`. (If the rename had taken
				// effect, the harness's service user would have started
				// claiming system:system-admin in its JWTs and gained
				// VirtualServerCreate.)
				role, err := h.Client().Project().Role(systemProjectSlug).Get(h.Ctx(), adminRoleId)
				Expect(err).ToNot(HaveOccurred())
				Expect(role.Name).To(Equal("admin"))
			})

			It("refuses DeleteRole on a system-project role for a non-system caller", func() {
				adminRoleId := findRoleId(h, systemProjectSlug, "admin")

				err := h.Client().Project().Role(systemProjectSlug).Delete(h.Ctx(), adminRoleId)
				Expect(err).To(HaveOccurred())

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected client.ApiError, got %T: %v", err, err)
				Expect(apiErr.Code).To(Equal(401))

				// Role must still exist.
				role, err := h.Client().Project().Role(systemProjectSlug).Get(h.Ctx(), adminRoleId)
				Expect(err).ToNot(HaveOccurred())
				Expect(role.Name).To(Equal("admin"))
			})

			It("still allows PatchRole on a non-system project role (regression check)", func() {
				// Create a fresh project + role belonging to it. The harness
				// admin has ProjectCreate / RoleCreate / RoleUpdate.
				projResp, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug:        "non-system-project-" + backend.name,
					Name:        "Non System Project",
					Description: "Non-system project for the PatchRole guard regression test.",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(projResp.Id).ToNot(BeZero())

				roleResp, err := h.Client().Project().Role("non-system-project-" + backend.name).Create(h.Ctx(), api.CreateRoleRequestDto{
					Name:        "rename-me",
					Description: "Role for the rename regression check.",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(roleResp.Id).ToNot(BeZero())

				err = h.Client().Project().Role("non-system-project-" + backend.name).Patch(
					h.Ctx(),
					roleResp.Id,
					api.PatchRoleRequestDto{Name: utils.Ptr("renamed")},
				)
				Expect(err).ToNot(HaveOccurred(),
					"the system-project guard must NOT fire on non-system projects")

				role, err := h.Client().Project().Role("non-system-project-" + backend.name).Get(h.Ctx(), roleResp.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(role.Name).To(Equal("renamed"))
			})

			It("still allows DeleteRole on a non-system project role (regression check)", func() {
				roleResp, err := h.Client().Project().Role("non-system-project-" + backend.name).Create(h.Ctx(), api.CreateRoleRequestDto{
					Name:        "delete-me",
					Description: "Role for the delete regression check.",
				})
				Expect(err).ToNot(HaveOccurred())

				err = h.Client().Project().Role("non-system-project-" + backend.name).Delete(h.Ctx(), roleResp.Id)
				Expect(err).ToNot(HaveOccurred(),
					"the system-project guard must NOT fire on non-system projects")
			})
		})
	}
}
