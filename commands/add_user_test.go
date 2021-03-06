package commands_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/goodmustache/pt/actions"
	. "github.com/goodmustache/pt/commands"
	"github.com/goodmustache/pt/config"
	"github.com/goodmustache/pt/tracker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/ghttp"
)

var _ = Describe("Add User", func() {
	const apiToken = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

	Describe("Help", func() {
		It("displays help for add-user", func() {
			session := runCommand("add-user", "-h")

			Eventually(session).ShouldNot(Exit(0))
			Expect(session.Err).To(Say("add-user"))
			Expect(session.Err).To(Say("--api-token"))
			Expect(session.Err).To(Say("--alias"))
		})
	})

	Context("when config does not exists", func() {
		DescribeTable("adding user",
			func(inputApiToken func(alias string) *Session, alias string) {
				tokenInfo := tracker.TokenInfomation{
					APIToken: apiToken,
					ID:       42,
					Name:     "Anand Gaitonde",
					Username: "agaitonde",
				}

				expectedConfig := config.Config{
					CurrentUserID: tokenInfo.ID,
					Users: []config.User{
						{
							APIToken: tokenInfo.APIToken,
							ID:       tokenInfo.ID,
							Name:     tokenInfo.Name,
							Username: tokenInfo.Username,
							Alias:    alias,
						},
					},
				}

				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/me"),
						VerifyHeader(http.Header{"X-TrackerToken": []string{apiToken}}),
						RespondWithJSONEncoded(http.StatusOK, tokenInfo),
					),
				)

				session := inputApiToken(alias)

				Eventually(session.Out).Should(Say("Added User! Setting %s \\(%s\\) to be the current user.", tokenInfo.Name, tokenInfo.Username))
				Eventually(session).Should(Exit(0))

				rawConfig, err := ioutil.ReadFile(path.Join(userHomeDir(), ".config", "pt", "config.json"))
				Expect(err).ToNot(HaveOccurred())

				var config config.Config
				err = json.Unmarshal(rawConfig, &config)
				Expect(err).ToNot(HaveOccurred())

				config.CurrentUserSetTime = time.Time{}
				Expect(config).To(Equal(expectedConfig))
			},

			Entry("prompts for api token if not passed", func(_ string) *Session {
				session, stdin := runCommandWithInput("add-user")
				defer stdin.Close()

				Eventually(session.Out).Should(Say(AddUserInstructions))

				Eventually(session.Out).Should(Say("API Token:"))
				inputValue(apiToken, stdin)
				return session
			}, ""),

			Entry("prompts for api token if not passed and adds alias", func(alias string) *Session {
				session, stdin := runCommandWithInput("add-user", "-a", alias)
				defer stdin.Close()

				Eventually(session.Out).Should(Say(AddUserInstructions))

				Eventually(session.Out).Should(Say("API Token:"))
				inputValue(apiToken, stdin)
				return session
			}, "banana"),

			Entry("does not prompt if api token is not passed", func(_ string) *Session {
				session := runCommand("add-user", "--api-token", apiToken)

				Eventually(session.Out).ShouldNot(Say(AddUserInstructions))
				Eventually(session.Out).ShouldNot(Say("API Token:"))
				return session
			}, ""),

			Entry("does not prompt if api token is not passed and adds alias", func(alias string) *Session {
				session := runCommand("add-user", "--api-token", apiToken, "-a", alias)

				Eventually(session.Out).ShouldNot(Say(AddUserInstructions))
				Eventually(session.Out).ShouldNot(Say("API Token:"))
				return session
			}, "banana"),
		)
	})

	Context("when the config exists", func() {
		var (
			tokenInfo tracker.TokenInfomation
			conf      config.Config
		)

		BeforeEach(func() {
			tokenInfo = tracker.TokenInfomation{
				APIToken: apiToken,
				ID:       42,
				Name:     "Anand Gaitonde",
				Username: "agaitonde",
			}

			conf = config.Config{
				CurrentUserID: 3,
				Users: []config.User{
					{
						APIToken: "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
						ID:       3,
						Name:     "Weyman Fung",
						Username: "weymanf",
						Alias:    "wf",
					},
				},
			}

			err := actions.WriteConfig(conf)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				CombineHandlers(
					VerifyRequest("GET", "/me"),
					RespondWithJSONEncoded(http.StatusOK, tokenInfo),
				),
			)
		})

		It("appends the new user to the config", func() {
			session := runCommand("add-user", "--api-token", apiToken)

			Eventually(session).Should(Exit(0))

			readConf, err := actions.ReadConfig()
			Expect(err).ToNot(HaveOccurred())

			users := readConf.Users
			Expect(users).To(HaveLen(2))
			Expect(users[0].ID).To(BeNumerically("==", 3))
			Expect(users[1].ID).To(BeNumerically("==", 42))
		})
	})
})
