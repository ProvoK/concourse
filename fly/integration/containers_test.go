package integration_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/fly/ui"
	"github.com/fatih/color"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Fly CLI", func() {
	var sampleContainers []atc.Container = []atc.Container{
		{
			ID:           "handle-1",
			WorkerName:   "worker-name-1",
			PipelineName: "pipeline-name",
			Type:         "check",
			ResourceName: "git-repo",
		},
		{
			ID:           "early-handle",
			WorkerName:   "worker-name-1",
			PipelineName: "pipeline-name",
			JobName:      "job-name-1",
			BuildName:    "3",
			BuildID:      123,
			Type:         "get",
			StepName:     "git-repo",
			Attempt:      "1.5",
		},
		{
			ID:           "other-handle",
			WorkerName:   "worker-name-2",
			PipelineName: "pipeline-name",
			JobName:      "job-name-2",
			BuildName:    "2",
			BuildID:      122,
			Type:         "task",
			StepName:     "unit-tests",
		},
		{
			ID:         "post-handle",
			WorkerName: "worker-name-3",
			BuildID:    142,
			Type:       "task",
			StepName:   "one-off",
		},
	}

	var sampleContainerJsonString string = `[
			{
				"id": "handle-1",
				"worker_name": "worker-name-1",
				"type": "check",
				"pipeline_name": "pipeline-name",
				"resource_name": "git-repo"
			},
			{
				"id": "early-handle",
				"worker_name": "worker-name-1",
				"type": "get",
				"step_name": "git-repo",
				"attempt": "1.5",
				"build_id": 123,
				"pipeline_name": "pipeline-name",
				"job_name": "job-name-1",
				"build_name": "3"
			},
			{
				"id": "other-handle",
				"worker_name": "worker-name-2",
				"type": "task",
				"step_name": "unit-tests",
				"build_id": 122,
				"pipeline_name": "pipeline-name",
				"job_name": "job-name-2",
				"build_name": "2"
			},
			{
				"id": "post-handle",
				"worker_name": "worker-name-3",
				"type": "task",
				"step_name": "one-off",
				"build_id": 142
			}
		]`

	Describe("containers", func() {
		var (
			flyCmd *exec.Cmd
		)

		BeforeEach(func() {
			flyCmd = exec.Command(flyPath, "-t", targetName, "containers")
		})

		Context("when containers are returned from the API", func() {
			BeforeEach(func() {
				atcServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/teams/main/containers"),
						ghttp.RespondWithJSONEncoded(200, sampleContainers)),
				)
			})

			Context("when --json is given", func() {
				BeforeEach(func() {
					flyCmd.Args = append(flyCmd.Args, "--json")
				})

				It("prints response in json as stdout", func() {
					sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(sess).Should(gexec.Exit(0))
					Expect(sess.Out.Contents()).To(MatchJSON(sampleContainerJsonString))
				})
			})

			It("lists them to the user, ordered by name", func() {
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(0))
				Expect(sess.Out).To(PrintTable(ui.Table{
					Headers: ui.TableRow{
						{Contents: "handle", Color: color.New(color.Bold)},
						{Contents: "worker", Color: color.New(color.Bold)},
						{Contents: "pipeline", Color: color.New(color.Bold)},
						{Contents: "job", Color: color.New(color.Bold)},
						{Contents: "build #", Color: color.New(color.Bold)},
						{Contents: "build id", Color: color.New(color.Bold)},
						{Contents: "type", Color: color.New(color.Bold)},
						{Contents: "name", Color: color.New(color.Bold)},
						{Contents: "attempt", Color: color.New(color.Bold)},
					},
					Data: []ui.TableRow{
						{{Contents: "early-handle"}, {Contents: "worker-name-1"}, {Contents: "pipeline-name"}, {Contents: "job-name-1"}, {Contents: "3"}, {Contents: "123"}, {Contents: "get"}, {Contents: "git-repo"}, {Contents: "1.5"}},
						{{Contents: "handle-1"}, {Contents: "worker-name-1"}, {Contents: "pipeline-name"}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "check"}, {Contents: "git-repo"}, {Contents: "n/a", Color: color.New(color.Faint)}},
						{{Contents: "other-handle"}, {Contents: "worker-name-2"}, {Contents: "pipeline-name"}, {Contents: "job-name-2"}, {Contents: "2"}, {Contents: "122"}, {Contents: "task"}, {Contents: "unit-tests"}, {Contents: "n/a", Color: color.New(color.Faint)}},
						{{Contents: "post-handle"}, {Contents: "worker-name-3"}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "none", Color: color.New(color.Faint)}, {Contents: "142"}, {Contents: "task"}, {Contents: "one-off"}, {Contents: "n/a", Color: color.New(color.Faint)}},
					},
				}))
			})
		})

		Context("the api returns an internal server error", func() {
			BeforeEach(func() {
				atcServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/teams/main/containers"),
						ghttp.RespondWith(500, ""),
					),
				)
			})

			It("writes an error message to stderr", func() {
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(1))
				Eventually(sess.Err).Should(gbytes.Say("Unexpected Response"))
			})
		})
		Context("containers for teams", func() {
			var loginATCServer *ghttp.Server

			encodedString := base64.RawStdEncoding.EncodeToString([]byte(`{
					"teams": {
						"main": ["owner"],
						"other-team": ["owner"]
					},
					"user_id": "test",
					"is_admin": true,
					"user_name": "test"
			}`))

			teams := []atc.Team{
				atc.Team{
					ID:   1,
					Name: "main",
				},
				atc.Team{
					ID:   2,
					Name: "other-team",
				},
			}
			credentials := base64.StdEncoding.EncodeToString([]byte("fly:Zmx5"))
			var teamHandler = func(teams []atc.Team) http.HandlerFunc {
				return ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/teams"),
					ghttp.VerifyHeaderKV("Authorization", "Bearer foo."+encodedString),
					ghttp.RespondWithJSONEncoded(200, teams),
				)
			}
			var adminTokenHandler = func() http.HandlerFunc {
				return ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/sky/token"),
					ghttp.VerifyHeaderKV("Content-Type", "application/x-www-form-urlencoded"),
					ghttp.VerifyHeaderKV("Authorization", fmt.Sprintf("Basic %s", credentials)),
					ghttp.VerifyFormKV("grant_type", "password"),
					ghttp.VerifyFormKV("username", "test"),
					ghttp.VerifyFormKV("password", "test"),
					ghttp.VerifyFormKV("scope", "openid profile email federated:id groups"),
					ghttp.RespondWithJSONEncoded(200, map[string]string{
						"token_type":   "Bearer",
						"access_token": "foo." + encodedString,
					}),
				)
			}

			BeforeEach(func() {
				flyCmd.Args = append(flyCmd.Args, "--team-name", "other-team")
				loginATCServer = ghttp.NewServer()
				loginATCServer.AppendHandlers(
					infoHandler(),
					adminTokenHandler(),
					teamHandler(teams),
					infoHandler(),
				)

				flyLoginCmd := exec.Command(flyPath, "-t", "some-target", "login", "-c", loginATCServer.URL(), "-n", "main", "-u", "test", "-p", "test")
				sess, err := gexec.Start(flyLoginCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gbytes.Say("logging in to team 'main'"))

				<-sess.Exited
				Expect(sess.ExitCode()).To(Equal(0))
				Expect(sess.Out).To(gbytes.Say("target saved"))
			})

			AfterEach(func() {
				loginATCServer.Close()
			})

			Context("using --team parameter", func() {
				BeforeEach(func() {
					loginATCServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/api/v1/teams/other-team/containers"),
							ghttp.RespondWithJSONEncoded(200, sampleContainers),
						),
					)
				})
				It("can list containers in 'other-team'", func() {
					flyContainerCmd := exec.Command(flyPath, "-t", "some-target", "--team-scope", "other-team", "containers", "--json")
					sess, err := gexec.Start(flyContainerCmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(sess).Should(gexec.Exit(0))
					Expect(sess.Out.Contents()).To(MatchJSON(sampleContainerJsonString))
				})
			})
			Context("using --all-teams parameter", func() {
				BeforeEach(func() {
					loginATCServer.AppendHandlers(
						teamHandler(teams),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/api/v1/teams/main/containers"),
							ghttp.RespondWithJSONEncoded(200, sampleContainers),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/api/v1/teams/other-team/containers"),
							ghttp.RespondWithJSONEncoded(200, []atc.Container{}),
						),
					)
				})
				It("can list all the containers of all the teams", func() {
					flyContainerCmd := exec.Command(flyPath, "-t", "some-target", "--all-teams", "containers", "--json")
					sess, err := gexec.Start(flyContainerCmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(sess).Should(gexec.Exit(0))
					Expect(sess.Out.Contents()).To(MatchJSON(sampleContainerJsonString))
				})
			})
		})
	})
})
