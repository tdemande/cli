package organization_test

import (
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/commands/organization"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func callShowOrg(args []string, requirementsFactory *testreq.FakeReqFactory) (ui *testterm.FakeUI) {
	ui = new(testterm.FakeUI)

	token := core_config.TokenInfo{Username: "my-user"}

	spaceFields := models.SpaceFields{}
	spaceFields.Name = "my-space"

	orgFields := models.OrganizationFields{}
	orgFields.Name = "my-org"

	configRepo := testconfig.NewRepositoryWithAccessToken(token)
	configRepo.SetSpaceFields(spaceFields)
	configRepo.SetOrganizationFields(orgFields)

	cmd := NewShowOrg(ui, configRepo)
	testcmd.RunCommand(cmd, args, requirementsFactory)
	return
}

var _ = Describe("org command", func() {
	var (
		ui                  *testterm.FakeUI
		configRepo          core_config.ReadWriter
		requirementsFactory *testreq.FakeReqFactory
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		requirementsFactory = &testreq.FakeReqFactory{}
		configRepo = testconfig.NewRepositoryWithDefaults()
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCommand(NewShowOrg(ui, configRepo), args, requirementsFactory)
	}

	Describe("requirements", func() {
		It("fails when not logged in", func() {
			Expect(runCommand("whoops")).To(BeFalse())
		})

		It("fails with usage when not provided exactly one arg", func() {
			requirementsFactory.LoginSuccess = true
			runCommand("too", "much")
			Expect(ui.FailedWithUsage).To(BeTrue())
		})
	})

	Context("when logged in, and provided the name of an org", func() {
		BeforeEach(func() {
			developmentSpaceFields := models.SpaceFields{}
			developmentSpaceFields.Name = "development"
			stagingSpaceFields := models.SpaceFields{}
			stagingSpaceFields.Name = "staging"
			domainFields := models.DomainFields{}
			domainFields.Name = "cfapps.io"
			cfAppDomainFields := models.DomainFields{}
			cfAppDomainFields.Name = "cf-app.com"

			org := models.Organization{}
			org.Name = "my-org"
			org.Guid = "my-org-guid"
			org.QuotaDefinition = models.NewQuotaFields("cantina-quota", 512, 256, 2, 5, true)
			org.Spaces = []models.SpaceFields{developmentSpaceFields, stagingSpaceFields}
			org.Domains = []models.DomainFields{domainFields, cfAppDomainFields}
			org.SpaceQuotas = []models.SpaceQuota{
				{Name: "space-quota-1"},
				{Name: "space-quota-2"},
			}

			requirementsFactory.LoginSuccess = true
			requirementsFactory.Organization = org
		})

		It("shows the org with the given name", func() {
			runCommand("my-org")

			Expect(requirementsFactory.OrganizationName).To(Equal("my-org"))
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Getting info for org", "my-org", "my-user"},
				[]string{"OK"},
				[]string{"my-org"},
				[]string{"domains:", "cfapps.io", "cf-app.com"},
				[]string{"quota: ", "cantina-quota", "512M", "256M instance memory limit", "2 routes", "5 services", "paid services allowed"},
				[]string{"spaces:", "development", "staging"},
				[]string{"space quotas:", "space-quota-1", "space-quota-2"},
			))
		})

		Context("when the guid flag is provided", func() {
			It("shows only the org guid", func() {
				runCommand("--guid", "my-org")

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"my-org-guid"},
				))

				Expect(ui.Outputs).ToNot(ContainSubstrings(
					[]string{"Getting info for org", "my-org", "my-user"},
				))
			})
		})
	})
})
