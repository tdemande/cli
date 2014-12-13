package plugin_test

import (
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cloudfoundry/cli/cf/command"
	testCommand "github.com/cloudfoundry/cli/cf/command/fakes"
	"github.com/cloudfoundry/cli/cf/command_metadata"
	"github.com/cloudfoundry/cli/cf/configuration/plugin_config"
	testconfig "github.com/cloudfoundry/cli/cf/configuration/plugin_config/fakes"
	"github.com/cloudfoundry/cli/plugin"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/commands/plugin"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Install", func() {
	var (
		ui                  *testterm.FakeUI
		requirementsFactory *testreq.FakeReqFactory
		config              *testconfig.FakePluginConfiguration

		coreCmds   map[string]command.Command
		pluginFile *os.File
		homeDir    string
		pluginDir  string
		curDir     string

		test_1                    string
		test_2                    string
		test_curDir               string
		test_with_help            string
		test_with_push            string
		test_with_push_short_name string
		aliasConflicts            string
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		requirementsFactory = &testreq.FakeReqFactory{}
		config = &testconfig.FakePluginConfiguration{}
		coreCmds = make(map[string]command.Command)

		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		test_1 = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "test_1.exe")
		test_2 = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "test_2.exe")
		test_curDir = filepath.Join("test_1.exe")
		test_with_help = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "test_with_help.exe")
		test_with_push = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "test_with_push.exe")
		test_with_push_short_name = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "test_with_push_short_name.exe")
		aliasConflicts = filepath.Join(dir, "..", "..", "..", "fixtures", "plugins", "alias_conflicts.exe")

		rpc.DefaultServer = rpc.NewServer()

		homeDir, err = ioutil.TempDir(os.TempDir(), "plugins")
		Expect(err).ToNot(HaveOccurred())

		pluginDir = filepath.Join(homeDir, ".cf", "plugins")
		config.GetPluginPathReturns(pluginDir)

		curDir, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		pluginFile, err = ioutil.TempFile("./", "test_plugin")
		Expect(err).ToNot(HaveOccurred())

		if runtime.GOOS != "windows" {
			err = os.Chmod(test_1, 0700)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		os.Remove(filepath.Join(curDir, pluginFile.Name()))
		os.Remove(homeDir)
	})

	runCommand := func(args ...string) bool {
		cmd := NewPluginInstall(ui, config, coreCmds)
		return testcmd.RunCommand(cmd, args, requirementsFactory)
	}

	Describe("requirements", func() {
		It("fails with usage when not provided a path to the plugin executable", func() {
			Expect(runCommand()).ToNot(HavePassedRequirements())
		})
	})

	Describe("failures", func() {
		Context("when the plugin contains a 'help' command", func() {
			It("fails", func() {
				runCommand(test_with_help)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Command `help` in the plugin being installed is a native CF command/alias.  Rename the `help` command in the plugin being installed in order to enable its installation and use."},
					[]string{"FAILED"},
				))
			})
		})

		Context("when the plugin's command conflicts with a core command", func() {
			It("fails if is shares a command name", func() {
				coreCmds["push"] = &testCommand.FakeCommand{}
				runCommand(test_with_push)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Command `push` in the plugin being installed is a native CF command/alias.  Rename the `push` command in the plugin being installed in order to enable its installation and use."},
					[]string{"FAILED"},
				))
			})

			It("fails if it shares a command short name", func() {
				push := &testCommand.FakeCommand{}
				push.MetadataReturns(command_metadata.CommandMetadata{
					ShortName: "p",
				})

				coreCmds["push"] = push
				runCommand(test_with_push_short_name)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Command `p` in the plugin being installed is a native CF command/alias.  Rename the `p` command in the plugin being installed in order to enable its installation and use."},
					[]string{"FAILED"},
				))
			})
		})

		Context("when the plugin's alias conflicts with a core command/alias", func() {
			It("fails if is shares a command name", func() {
				coreCmds["conflict-alias"] = &testCommand.FakeCommand{}
				runCommand(aliasConflicts)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Alias `conflict-alias` in the plugin being installed is a native CF command/alias.  Rename the `conflict-alias` command in the plugin being installed in order to enable its installation and use."},
					[]string{"FAILED"},
				))
			})

			It("fails if it shares a command short name", func() {
				push := &testCommand.FakeCommand{}
				push.MetadataReturns(command_metadata.CommandMetadata{
					ShortName: "conflict-alias",
				})

				coreCmds["push"] = push
				runCommand(aliasConflicts)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Alias `conflict-alias` in the plugin being installed is a native CF command/alias.  Rename the `conflict-alias` command in the plugin being installed in order to enable its installation and use."},
					[]string{"FAILED"},
				))
			})
		})

		Context("when the plugin's alias conflicts with other installed plugin", func() {
			It("fails if it shares a command name", func() {
				pluginsMap := make(map[string]plugin_config.PluginMetadata)
				pluginsMap["AliasCollision"] = plugin_config.PluginMetadata{
					Location: "location/to/config.exe",
					Commands: []plugin.Command{
						{
							Name:     "conflict-alias",
							HelpText: "Hi!",
						},
					},
				}
				config.PluginsReturns(pluginsMap)

				runCommand(aliasConflicts)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Alias `conflict-alias` is a command/alias in plugin 'AliasCollision'.  You could try uninstalling plugin 'AliasCollision' and then install this plugin in order to invoke the `conflict-alias` command.  However, you should first fully understand the impact of uninstalling the existing 'AliasCollision' plugin."},
					[]string{"FAILED"},
				))
			})

			It("fails if it shares a command alias", func() {
				pluginsMap := make(map[string]plugin_config.PluginMetadata)
				pluginsMap["AliasCollision"] = plugin_config.PluginMetadata{
					Location: "location/to/alias.exe",
					Commands: []plugin.Command{
						{
							Name:     "non-conflict-cmd",
							Alias:    "conflict-alias",
							HelpText: "Hi!",
						},
					},
				}
				config.PluginsReturns(pluginsMap)

				runCommand(aliasConflicts)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Alias `conflict-alias` is a command/alias in plugin 'AliasCollision'.  You could try uninstalling plugin 'AliasCollision' and then install this plugin in order to invoke the `conflict-alias` command.  However, you should first fully understand the impact of uninstalling the existing 'AliasCollision' plugin."},
					[]string{"FAILED"},
				))
			})
		})

		Context("when the plugin's command conflicts with other installed plugin", func() {
			It("fails if it shares a command name", func() {
				pluginsMap := make(map[string]plugin_config.PluginMetadata)
				pluginsMap["Test1Collision"] = plugin_config.PluginMetadata{
					Location: "location/to/config.exe",
					Commands: []plugin.Command{
						{
							Name:     "test_1_cmd1",
							HelpText: "Hi!",
						},
					},
				}
				config.PluginsReturns(pluginsMap)

				runCommand(test_1)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Command `test_1_cmd1` is a command/alias in plugin 'Test1Collision'.  You could try uninstalling plugin 'Test1Collision' and then install this plugin in order to invoke the `test_1_cmd1` command.  However, you should first fully understand the impact of uninstalling the existing 'Test1Collision' plugin."},
					[]string{"FAILED"},
				))
			})

			It("fails if it shares a command alias", func() {
				pluginsMap := make(map[string]plugin_config.PluginMetadata)
				pluginsMap["AliasCollision"] = plugin_config.PluginMetadata{
					Location: "location/to/alias.exe",
					Commands: []plugin.Command{
						{
							Name:     "non-conflict-cmd",
							Alias:    "conflict-cmd",
							HelpText: "Hi!",
						},
					},
				}
				config.PluginsReturns(pluginsMap)

				runCommand(aliasConflicts)

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Command `conflict-cmd` is a command/alias in plugin 'AliasCollision'.  You could try uninstalling plugin 'AliasCollision' and then install this plugin in order to invoke the `conflict-cmd` command.  However, you should first fully understand the impact of uninstalling the existing 'AliasCollision' plugin."},
					[]string{"FAILED"},
				))
			})
		})

		Context("Locating binary file", func() {

			Context("first tries to locate binary file at local path", func() {
				It("will not try downloading from internet if file is found locally", func() {
					runCommand("./install_plugin.go")

					Expect(ui.Outputs).ToNot(ContainSubstrings(
						[]string{"Attempting to download binary file from internet"},
					))
				})
			})

			Context("tries to download binary from net if file is not found locally", func() {
				It("informs users when binary is not downloadable from net", func() {
					runCommand("path/to/not/a/thing.exe")

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Download attempt failed"},
						[]string{"Unable to install"},
						[]string{"FAILED"},
					))
				})
			})

		})

		It("if plugin name is already taken", func() {
			config.PluginsReturns(map[string]plugin_config.PluginMetadata{"Test1": plugin_config.PluginMetadata{}})
			runCommand(test_1)

			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Plugin name", "Test1", "is already taken"},
				[]string{"FAILED"},
			))
		})

		Context("io", func() {
			BeforeEach(func() {
				err := os.MkdirAll(pluginDir, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("if a file with the plugin name already exists under ~/.cf/plugin/", func() {
				config.PluginsReturns(map[string]plugin_config.PluginMetadata{"useless": plugin_config.PluginMetadata{}})
				config.GetPluginPathReturns(curDir)

				runCommand(filepath.Join(curDir, pluginFile.Name()))
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Installing plugin"},
					[]string{"The file", pluginFile.Name(), "already exists"},
					[]string{"FAILED"},
				))
			})
		})
	})

	Describe("success", func() {
		BeforeEach(func() {
			err := os.MkdirAll(pluginDir, 0700)
			Expect(err).ToNot(HaveOccurred())
			config.GetPluginPathReturns(pluginDir)
		})

		It("finds plugin in the current directory without having to specify `./`", func() {
			curDir, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			err = os.Chdir("../../../fixtures/plugins")
			Expect(err).ToNot(HaveOccurred())

			runCommand(test_curDir)
			_, err = os.Stat(filepath.Join(pluginDir, "test_1.exe"))
			Expect(err).ToNot(HaveOccurred())

			err = os.Chdir(curDir)
			Expect(err).ToNot(HaveOccurred())
		})

		It("copies the plugin into directory <FAKE_HOME_DIR>/.cf/plugins/PLUGIN_FILE_NAME", func() {
			runCommand(test_1)

			_, err := os.Stat(test_1)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filepath.Join(pluginDir, "test_1.exe"))
			Expect(err).ToNot(HaveOccurred())
		})

		if runtime.GOOS != "windows" {
			It("Chmods the plugin so it is executable", func() {
				runCommand(test_1)

				fileInfo, err := os.Stat(filepath.Join(pluginDir, "test_1.exe"))
				Expect(err).ToNot(HaveOccurred())
				Expect(int(fileInfo.Mode())).To(Equal(0700))
			})
		}

		It("populate the configuration with plugin metadata", func() {
			runCommand(test_1)

			pluginName, pluginMetadata := config.SetPluginArgsForCall(0)

			Expect(pluginName).To(Equal("Test1"))
			Expect(pluginMetadata.Location).To(Equal(filepath.Join(pluginDir, "test_1.exe")))
			Expect(pluginMetadata.Commands[0].Name).To(Equal("test_1_cmd1"))
			Expect(pluginMetadata.Commands[0].HelpText).To(Equal("help text for test_1_cmd1"))
			Expect(pluginMetadata.Commands[1].Name).To(Equal("test_1_cmd2"))
			Expect(pluginMetadata.Commands[1].HelpText).To(Equal("help text for test_1_cmd2"))
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Installing plugin", test_1},
				[]string{"OK"},
				[]string{"Plugin", "Test1", "successfully installed"},
			))
		})

		It("installs multiple plugins with no aliases", func() {
			Expect(runCommand(test_1)).To(Equal(true))
			Expect(runCommand(test_2)).To(Equal(true))
		})
	})
})
