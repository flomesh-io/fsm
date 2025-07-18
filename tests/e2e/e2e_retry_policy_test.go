package e2e

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

const client = "client"
const server = "server"

var meshNs = []string{client, server}

var retryStats = map[string]string{"upstream_rq_retry": "", "upstream_rq_retry_limit_exceeded": "", "upstream_rq_retry_backoff_exponential": ""}
var thresholdUintVal uint32 = 5

var _ = FSMDescribe("Test Retry Policy",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 9,
	},
	func() {
		Context("Retry policy enabled", func() {
			It("tests retry policy",
				func() {
					// Install FSM
					installOpts := Td.GetFSMInstallOpts()
					installOpts.EnablePermissiveMode = true
					installOpts.EnableRetryPolicy = true
					Expect(Td.InstallFSM(installOpts)).To(Succeed())

					// Create test NS in mesh
					for _, n := range meshNs {
						Expect(Td.CreateNs(n, nil)).To(Succeed())
						Expect(Td.AddNsToMesh(true, n)).To(Succeed())
					}

					// Get simple pod definitions for the HTTP server
					svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
						SimplePodAppDef{
							PodName:   server,
							Namespace: server,
							Image:     "ghcr.io/mccutchen/go-httpbin",
							Ports:     []int{80},
							OS:        Td.ClusterOS,
							Command:   []string{"/bin/go-httpbin"},
							Args:      []string{"-port", "80"},
						})
					Expect(err).NotTo(HaveOccurred())

					_, err = Td.CreateServiceAccount(server, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreatePod(server, podDef)
					Expect(err).NotTo(HaveOccurred())
					serverSvc, err := Td.CreateService(server, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(server, 1, nil)).To(Succeed())

					// Get simple Pod definitions for the source/client
					svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
						PodName:   client,
						Namespace: client,
						Command:   []string{"sleep", "365d"},
						Image:     "flomesh/fsm-curl",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
					Expect(err).NotTo(HaveOccurred())

					clientSvcAcct, err := Td.CreateServiceAccount(client, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					clientPod, err := Td.CreatePod(client, podDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(client, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(client, 1, nil)).To(Succeed())

					retry := &v1alpha1.Retry{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "retrypolicy",
							Namespace: client,
						},
						Spec: v1alpha1.RetrySpec{
							Source: v1alpha1.RetrySrcDstSpec{
								Kind:      "ServiceAccount",
								Name:      clientSvcAcct.Name,
								Namespace: client,
							},
							Destinations: []v1alpha1.RetrySrcDstSpec{
								{
									Kind:      "Service",
									Name:      serverSvc.Name,
									Namespace: server,
								},
							},
							RetryPolicy: v1alpha1.RetryPolicySpec{
								RetryOn:                  "5xx",
								PerTryTimeout:            &metav1.Duration{Duration: time.Duration(1 * time.Second)},
								NumRetries:               &thresholdUintVal,
								RetryBackoffBaseInterval: &metav1.Duration{Duration: time.Duration(5 * time.Second)},
							},
						},
					}
					_, err = Td.PolicyClient.PolicyV1alpha1().Retries(client).Create(context.TODO(), retry, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					time.Sleep(90 * time.Second)

					req := HTTPRequestDef{
						SourceNs:        client,
						SourcePod:       clientPod.Name,
						SourceContainer: podDef.GetName(),
						Destination:     fmt.Sprintf("%s.%s.svc.cluster.local:80/status/503", serverSvc.Name, server),
					}

					By("A request that will be retried NumRetries times then fail")
					err = wait.PollUntilContextTimeout(context.Background(), time.Second*30, time.Second*30, true, func(ctx context.Context) (bool, error) {
						defer GinkgoRecover()
						result := Td.HTTPRequest(req)
						if result.Err != nil {
							Td.T.Logf("Error making HTTP request: %v, status code: %d", result.Err, result.StatusCode)
						}

						stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/fsm"), "proxy", "get", "stats", clientPod.Name, "--namespace", client)
						if err != nil {
							Td.T.Logf("Could not get client stats: %v", stderr)
						}

						metrics, err := findRetryStats(stdout.String(), serverSvc.Name+"|80", retryStats)
						Expect(err).ToNot(HaveOccurred())

						return Expect(result.StatusCode).To(Equal(503)) &&
							// upstream_rq_retry: Total request retries
							Expect(metrics["upstream_rq_retry"]).To(Equal("5")) &&
							// upstream_rq_retry_limit_exceeded: Total requests not retried because max retries reached
							Expect(metrics["upstream_rq_retry_limit_exceeded"]).To(Equal("1")) &&
							// upstream_rq_retry_backoff_exponential: Total retries using the exponential backoff strategy
							Expect(metrics["upstream_rq_retry_backoff_exponential"]).To(Equal("5")), nil
					})
					Expect(err).ToNot(HaveOccurred())

				})
		})
		if false {
			Context("Retry policy disabled", func() {
				It("tests retry policy",
					func() {
						// Install FSM
						installOpts := Td.GetFSMInstallOpts()
						installOpts.EnablePermissiveMode = true
						installOpts.EnableRetryPolicy = false
						Expect(Td.InstallFSM(installOpts)).To(Succeed())

						sidecarClass := Td.GetSidecarClass(Td.FsmNamespace)
						if len(sidecarClass) == 0 || sidecarClass == constants.SidecarClassPipy {
							Skip("Pipy doesn't support retry policy")
						}

						// Create test NS in mesh
						for _, n := range meshNs {
							Expect(Td.CreateNs(n, nil)).To(Succeed())
							Expect(Td.AddNsToMesh(true, n)).To(Succeed())
						}

						// Get simple pod definitions for the HTTP server
						svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
							SimplePodAppDef{
								PodName:   server,
								Namespace: server,
								Image:     "flomesh/httpbin:ken",
								Ports:     []int{80},
								OS:        Td.ClusterOS,
							})
						Expect(err).NotTo(HaveOccurred())

						_, err = Td.CreateServiceAccount(server, &svcAccDef)
						Expect(err).NotTo(HaveOccurred())
						_, err = Td.CreatePod(server, podDef)
						Expect(err).NotTo(HaveOccurred())
						serverSvc, err := Td.CreateService(server, svcDef)
						Expect(err).NotTo(HaveOccurred())

						Expect(Td.WaitForPodsRunningReady(server, 1, nil)).To(Succeed())

						// Get simple Pod definitions for the source/client
						svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
							PodName:   client,
							Namespace: client,
							Command:   []string{"sleep", "365d"},
							Image:     "flomesh/fsm-curl",
							Ports:     []int{80},
							OS:        Td.ClusterOS,
						})
						Expect(err).NotTo(HaveOccurred())

						clientSvcAcct, err := Td.CreateServiceAccount(client, &svcAccDef)
						Expect(err).NotTo(HaveOccurred())
						clientPod, err := Td.CreatePod(client, podDef)
						Expect(err).NotTo(HaveOccurred())
						_, err = Td.CreateService(client, svcDef)
						Expect(err).NotTo(HaveOccurred())

						Expect(Td.WaitForPodsRunningReady(client, 1, nil)).To(Succeed())

						retry := &v1alpha1.Retry{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "retrypolicy",
								Namespace: client,
							},
							Spec: v1alpha1.RetrySpec{
								Source: v1alpha1.RetrySrcDstSpec{
									Kind:      "ServiceAccount",
									Name:      clientSvcAcct.Name,
									Namespace: client,
								},
								Destinations: []v1alpha1.RetrySrcDstSpec{
									{
										Kind:      "Service",
										Name:      serverSvc.Name,
										Namespace: server,
									},
								},
								RetryPolicy: v1alpha1.RetryPolicySpec{
									RetryOn:                  "5xx",
									PerTryTimeout:            &metav1.Duration{Duration: time.Duration(1 * time.Second)},
									NumRetries:               &thresholdUintVal,
									RetryBackoffBaseInterval: &metav1.Duration{Duration: time.Duration(5 * time.Second)},
								},
							},
						}
						_, err = Td.PolicyClient.PolicyV1alpha1().Retries(client).Create(context.TODO(), retry, metav1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						time.Sleep(30 * time.Second)

						req := HTTPRequestDef{
							SourceNs:        client,
							SourcePod:       clientPod.Name,
							SourceContainer: podDef.GetName(),
							Destination:     fmt.Sprintf("%s.%s.svc.cluster.local:80/status/503", serverSvc.Name, server),
						}

						By("A request that will be retried 0 times and then fail")
						err = wait.Poll(time.Second*30, time.Second*30, func() (bool, error) {
							defer GinkgoRecover()
							result := Td.HTTPRequest(req)

							stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/fsm"), "proxy", "get", "stats", clientPod.Name, "--namespace", client)
							if err != nil {
								Td.T.Logf("Could not get client stats: %v", stderr)
							}

							metrics, err := findRetryStats(stdout.String(), serverSvc.Name+"|80", retryStats)
							Expect(err).ToNot((HaveOccurred()))

							return Expect(result.StatusCode).To(Equal(503)) &&
								// upstream_rq_retry: Total request retries
								Expect(metrics["upstream_rq_retry"]).To(Equal("0")) &&
								// upstream_rq_retry_limit_exceeded: Total requests not retried because max retries reached
								Expect(metrics["upstream_rq_retry_limit_exceeded"]).To(Equal("0")) &&
								// upstream_rq_retry_backoff_exponential: Total retries using the exponential backoff strategy
								Expect(metrics["upstream_rq_retry_backoff_exponential"]).To(Equal("0")), nil
						})
						Expect(err).ToNot(HaveOccurred())
					})
			})
		}
	})

func findRetryStats(output, serverSvc string, retryStats map[string]string) (map[string]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		stat := scanner.Text()
		if strings.Contains(stat, serverSvc) {
			retryStats = getMetric(stat, retryStats)
		}
	}

	err := scanner.Err()
	return retryStats, err
}

func getMetric(stat string, retryStats map[string]string) map[string]string {
	for r := range retryStats {
		regR := r + "\\b"
		match, _ := regexp.MatchString(regR, stat)
		if match {
			splitStat := strings.Split(stat, ":")
			res := strings.ReplaceAll(splitStat[1], " ", "")
			retryStats[r] = res
		}
	}
	return retryStats
}
