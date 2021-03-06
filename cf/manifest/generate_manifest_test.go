package manifest_test

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/cloudfoundry/cli/cf/manifest"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type outputs struct {
	contents []string
	cursor   int
}

var _ = Describe("generate_manifest", func() {
	var (
		m   AppManifest
		err error
	)

	BeforeEach(func() {
		_, err = os.Stat("./output.yml")
		Ω(err).To(HaveOccurred())

		m = NewGenerator()
		m.FileSavePath("./output.yml")
	})

	AfterEach(func() {
		err = os.Remove("./output.yml")
		Ω(err).ToNot(HaveOccurred())
	})

	It("creates a new file at a given path", func() {
		m.Save()

		_, err = os.Stat("./output.yml")
		Ω(err).ToNot(HaveOccurred())
	})

	It("starts the manifest with 3 dashes (---), followed by 'applications'", func() {
		m.Save()

		contents := getYamlContent("./output.yml")

		Ω(contents[0]).To(Equal("---"))
		Ω(contents[1]).To(Equal("applications:"))
	})

	It("creates entry under the given app name", func() {
		m.Memory("app1", 128)
		m.Memory("app2", 64)
		m.Save()

		//outputs.ContainSubstring assert orders
		cmdOutput := &outputs{
			contents: getYamlContent("./output.yml"),
			cursor:   0,
		}

		Ω(cmdOutput.ContainsSubstring("- name: app1")).To(BeTrue())
		Ω(cmdOutput.ContainsSubstring("  memory: 128M")).To(BeTrue())

		Ω(cmdOutput.ContainsSubstring("- name: app2")).To(BeTrue())
		Ω(cmdOutput.ContainsSubstring("  memory: 64M")).To(BeTrue())
	})

	It("prefixes each service with '-'", func() {
		m.Service("app1", "service1")
		m.Service("app1", "service2")
		m.Service("app1", "service3")
		m.Save()

		contents := getYamlContent("./output.yml")

		Ω(contents).To(ContainSubstrings(
			[]string{"  services:"},
			[]string{"- service1"},
			[]string{"- service2"},
			[]string{"- service3"},
		))
	})

	It("generates a manifest containing all the attributes", func() {
		m.Memory("app1", 128)
		m.Service("app1", "service1")
		m.EnvironmentVars("app1", "foo", "boo")
		m.HealthCheckTimeout("app1", 100)
		m.Instances("app1", 3)
		m.Domain("app1", "foo", "blahblahblah.com")
		err := m.Save()
		Ω(err).NotTo(HaveOccurred())

		Ω(getYamlContent("./output.yml")).To(ContainSubstrings(
			[]string{"- name: app1"},
			[]string{"  memory: 128M"},
			[]string{"  services:"},
			[]string{"  - service1"},
			[]string{"  env:"},
			[]string{"    foo: boo"},
			[]string{"  timeout: 100"},
			[]string{"  instances: 3"},
			[]string{"  host: foo"},
			[]string{"  domain: blahblahblah.com"},
		))
	})

})

func getYamlContent(path string) []string {
	b, err := ioutil.ReadFile(path)
	Ω(err).ToNot(HaveOccurred())

	return strings.Split(string(b), "\n")
}

func (o *outputs) ContainsSubstring(str string) bool {
	for i := o.cursor; i < len(o.contents)-1; i++ {
		if strings.Contains(o.contents[i], str) {
			o.cursor = i
			return true
		}
	}
	return false
}
