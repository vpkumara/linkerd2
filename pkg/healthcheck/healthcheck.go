package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/linkerd/linkerd2/controller/api/public"
	spclient "github.com/linkerd/linkerd2/controller/gen/client/clientset/versioned"
	healthcheckPb "github.com/linkerd/linkerd2/controller/gen/common/healthcheck"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/pkg/profiles"
	"github.com/linkerd/linkerd2/pkg/version"
	authorizationapi "k8s.io/api/authorization/v1beta1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

// Checks is an enum for the types of health checks.
type Checks int

const (
	// KubernetesAPIChecks adds a series of checks to validate that the caller is
	// configured to interact with a working Kubernetes cluster.
	KubernetesAPIChecks Checks = iota

	// KubernetesVersionChecks validate that the cluster meets the minimum version
	// requirements.
	KubernetesVersionChecks

	// LinkerdPreInstallClusterChecks adds checks to validate that the control
	// plane namespace does not already exist, and that the user can create
	// cluster-wide resources, including ClusterRole, ClusterRoleBinding, and
	// CustomResourceDefinition. This check only runs as part of the set
	// of pre-install checks.
	// This check is dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdPreInstallClusterChecks

	// LinkerdPreInstallSingleNamespaceChecks adds a check to validate that the
	// control plane namespace already exists, and that the user can create
	// namespace-scoped resources, including Role and RoleBinding. This check only
	// runs as part of the set of pre-install checks.
	// This check is dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdPreInstallSingleNamespaceChecks

	// LinkerdPreInstallChecks adds checks to validate that the user can create
	// Kubernetes objects necessary to install the control plane, including
	// Service, Deployment, and ConfigMap. This check only runs as part of the set
	// of pre-install checks.
	// This check is dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdPreInstallChecks

	// LinkerdDataPlaneExistenceChecks adds a data plane check to validate that
	// the data plane namespace exists.
	// This check is dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdDataPlaneExistenceChecks

	// LinkerdDataPlaneChecks adds a data plane check to validate that the proxy
	// containers are in the ready state.
	// This check is dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdDataPlaneChecks

	// LinkerdControlPlaneExistenceChecks adds a series of checks to validate that
	// the control plane namespace and controller pod exist.
	// These checks are dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdControlPlaneExistenceChecks

	// LinkerdAPIChecks adds a series of checks to validate that the control plane
	// is successfully serving the public API.
	// These checks are dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdAPIChecks

	// LinkerdServiceProfileChecks add a check validate any ServiceProfiles that
	// may already be installed.
	// These checks are dependent on the output of KubernetesAPIChecks, so those
	// checks must be added first.
	LinkerdServiceProfileChecks

	// LinkerdVersionChecks adds a series of checks to query for the latest
	// version, and validate the the CLI is up to date.
	LinkerdVersionChecks

	// LinkerdControlPlaneVersionChecks adds a series of checks to validate that
	// the control plane is running the latest available version.
	// These checks are dependent on `apiClient` from
	// LinkerdControlPlaneExistenceChecks and `latestVersion` from
	// LinkerdVersionChecks, so those checks must be added first.
	LinkerdControlPlaneVersionChecks

	// LinkerdDataPlaneVersionChecks adds a series of checks to validate that the
	// control plane is running the latest available version.
	// These checks are dependent on `apiClient` from
	// LinkerdControlPlaneExistenceChecks and `latestVersion` from
	// LinkerdVersionChecks, so those checks must be added first.
	LinkerdDataPlaneVersionChecks

	// KubernetesAPICategory is the string representation of KubernetesAPIChecks.
	KubernetesAPICategory = "kubernetes-api"
	// KubernetesVersionCategory is the string representation of
	// KubernetesVersionChecks.
	KubernetesVersionCategory = "kubernetes-version"
	// LinkerdPreInstallClusterCategory is the string representation of
	// LinkerdPreInstallClusterChecks.
	LinkerdPreInstallClusterCategory = "kubernetes-cluster-setup"
	// LinkerdPreInstallSingleNamespaceCategory is the string representation of
	// LinkerdPreInstallSingleNamespaceChecks.
	LinkerdPreInstallSingleNamespaceCategory = "kubernetes-single-namespace-setup"
	// LinkerdPreInstallCategory is the string representation of
	// LinkerdPreInstallChecks.
	LinkerdPreInstallCategory = "kubernetes-setup"
	// LinkerdDataPlaneExistenceCategory is the string representation of
	// LinkerdDataPlaneExistenceChecks.
	LinkerdDataPlaneExistenceCategory = "linkerd-data-plane-existence"
	// LinkerdDataPlaneCategory is the string representation of
	// LinkerdDataPlaneChecks.
	LinkerdDataPlaneCategory = "linkerd-data-plane"
	// LinkerdControlPlaneExistenceCategory is the string representation of
	// LinkerdControlPlaneExistenceChecks.
	LinkerdControlPlaneExistenceCategory = "linkerd-existence"
	// LinkerdAPICategory is the string representation of LinkerdAPIChecks.
	LinkerdAPICategory = "linkerd-api"
	// LinkerdServiceProfileCategory is the string representation of
	// LinkerdServiceProfileChecks.
	LinkerdServiceProfileCategory = "linkerd-service-profile"
	// LinkerdVersionCategory is the string representation of
	// LinkerdVersionChecks.
	LinkerdVersionCategory = "linkerd-version"
	// LinkerdControlPlaneVersionCategory is the string representation of
	// LinkerdControlPlaneVersionChecks.
	LinkerdControlPlaneVersionCategory = "linkerd-control-plane-version"
	// LinkerdDataPlaneVersionCategory is the string representation of
	// LinkerdDataPlaneVersionChecks.
	LinkerdDataPlaneVersionCategory = "linkerd-data-plane-version"
)

var (
	maxRetries        = 60
	retryWindow       = 5 * time.Second
	clusterZoneSuffix = []string{"svc", "cluster", "local"}
)

type checker struct {
	// category is one of the *Category constants defined above
	category string

	// description is the short description that's printed to the command line
	// when the check is executed
	description string

	// fatal indicates that all remaining checks should be aborted if this check
	// fails; it should only be used if subsequent checks cannot possibly succeed
	// (default false)
	fatal bool

	// warning indicates that if this check fails, it should be reported, but it
	// should not impact the overall outcome of the health check (default false)
	warning bool

	// retryDeadline establishes a deadline before which this check should be
	// retried; if the deadline has passed, the check fails (default: no retries)
	retryDeadline time.Time

	// check is the function that's called to execute the check; if the function
	// returns an error, the check fails
	check func() error

	// checkRPC is an alternative to check that can be used to perform a remote
	// check using the SelfCheck gRPC endpoint; check status is based on the value
	// of the gRPC response
	checkRPC func() (*healthcheckPb.SelfCheckResponse, error)
}

// CheckResult encapsulates a check's identifying information and output
type CheckResult struct {
	Category    string
	Description string
	Retry       bool
	Warning     bool
	Err         error
}

type checkObserver func(*CheckResult)

// Options specifies configuration for a HealthChecker.
type Options struct {
	ControlPlaneNamespace string
	DataPlaneNamespace    string
	KubeConfig            string
	KubeContext           string
	APIAddr               string
	VersionOverride       string
	RetryDeadline         time.Time
}

// HealthChecker encapsulates all health check checkers, and clients required to
// perform those checks.
type HealthChecker struct {
	checkers []*checker
	*Options

	// these fields are set in the process of running checks
	kubeAPI          *k8s.KubernetesAPI
	httpClient       *http.Client
	clientset        *kubernetes.Clientset
	spClientset      *spclient.Clientset
	kubeVersion      *k8sVersion.Info
	controlPlanePods []v1.Pod
	apiClient        pb.ApiClient
	latestVersion    string
}

// NewHealthChecker returns an initialized HealthChecker
func NewHealthChecker(checks []Checks, options *Options) *HealthChecker {
	hc := &HealthChecker{
		checkers: make([]*checker, 0),
		Options:  options,
	}

	for _, check := range checks {
		switch check {
		case KubernetesAPIChecks:
			hc.addKubernetesAPIChecks()
		case KubernetesVersionChecks:
			hc.addKubernetesVersionChecks()
		case LinkerdPreInstallClusterChecks:
			hc.addLinkerdPreInstallClusterChecks()
		case LinkerdPreInstallSingleNamespaceChecks:
			hc.addLinkerdPreInstallSingleNamespaceChecks()
		case LinkerdPreInstallChecks:
			hc.addLinkerdPreInstallChecks()
		case LinkerdDataPlaneExistenceChecks:
			hc.addLinkerdDataPlaneExistenceChecks()
		case LinkerdDataPlaneChecks:
			hc.addLinkerdDataPlaneChecks()
		case LinkerdControlPlaneExistenceChecks:
			hc.addLinkerdControlPlaneExistenceChecks()
		case LinkerdAPIChecks:
			hc.addLinkerdAPIChecks()
		case LinkerdServiceProfileChecks:
			hc.addLinkerdServiceProfileChecks()
		case LinkerdVersionChecks:
			hc.addLinkerdVersionChecks()
		case LinkerdControlPlaneVersionChecks:
			hc.addLinkerdControlPlaneVersionChecks()
		case LinkerdDataPlaneVersionChecks:
			hc.addLinkerdDataPlaneVersionChecks()
		}
	}

	return hc
}

func (hc *HealthChecker) addKubernetesAPIChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    KubernetesAPICategory,
		description: "can initialize the client",
		fatal:       true,
		check: func() (err error) {
			hc.kubeAPI, err = k8s.NewAPI(hc.KubeConfig, hc.KubeContext)
			return
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    KubernetesAPICategory,
		description: "can query the Kubernetes API",
		fatal:       true,
		check: func() (err error) {
			hc.httpClient, err = hc.kubeAPI.NewClient()
			if err != nil {
				return
			}
			hc.kubeVersion, err = hc.kubeAPI.GetVersionInfo(hc.httpClient)
			return
		},
	})
}

func (hc *HealthChecker) addKubernetesVersionChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    KubernetesVersionCategory,
		description: "is running the minimum Kubernetes API version",
		check: func() error {
			return hc.kubeAPI.CheckVersion(hc.kubeVersion)
		},
	})
}

func (hc *HealthChecker) addLinkerdPreInstallClusterChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallClusterCategory,
		description: "control plane namespace does not already exist",
		check: func() error {
			return hc.checkNamespace(hc.ControlPlaneNamespace, false)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallClusterCategory,
		description: "can create Namespaces",
		check: func() error {
			return hc.checkCanCreate("", "", "v1", "Namespace")
		},
	})

	// TODO: refactor with LinkerdPreInstallSingleNamespaceChecks
	roleType := "ClusterRole"
	roleBindingType := "ClusterRoleBinding"

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallClusterCategory,
		description: fmt.Sprintf("can create %ss", roleType),
		check: func() error {
			return hc.checkCanCreate("", "rbac.authorization.k8s.io", "v1beta1", roleType)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallClusterCategory,
		description: fmt.Sprintf("can create %ss", roleBindingType),
		check: func() error {
			return hc.checkCanCreate("", "rbac.authorization.k8s.io", "v1beta1", roleBindingType)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallClusterCategory,
		description: "can create CustomResourceDefinitions",
		check: func() error {
			return hc.checkCanCreate(hc.ControlPlaneNamespace, "apiextensions.k8s.io", "v1beta1", "CustomResourceDefinition")
		},
	})
}

func (hc *HealthChecker) addLinkerdPreInstallSingleNamespaceChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallSingleNamespaceCategory,
		description: "control plane namespace exists",
		check: func() error {
			return hc.checkNamespace(hc.ControlPlaneNamespace, true)
		},
	})

	// TODO: refactor with LinkerdPreInstallClusterChecks
	roleType := "Role"
	roleBindingType := "RoleBinding"

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallSingleNamespaceCategory,
		description: fmt.Sprintf("can create %ss", roleType),
		check: func() error {
			return hc.checkCanCreate("", "rbac.authorization.k8s.io", "v1beta1", roleType)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallSingleNamespaceCategory,
		description: fmt.Sprintf("can create %ss", roleBindingType),
		check: func() error {
			return hc.checkCanCreate("", "rbac.authorization.k8s.io", "v1beta1", roleBindingType)
		},
	})
}

func (hc *HealthChecker) addLinkerdPreInstallChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallCategory,
		description: "can create ServiceAccounts",
		check: func() error {
			return hc.checkCanCreate(hc.ControlPlaneNamespace, "", "v1", "ServiceAccount")
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallCategory,
		description: "can create Services",
		check: func() error {
			return hc.checkCanCreate(hc.ControlPlaneNamespace, "", "v1", "Service")
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallCategory,
		description: "can create Deployments",
		check: func() error {
			return hc.checkCanCreate(hc.ControlPlaneNamespace, "extensions", "v1beta1", "Deployments")
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdPreInstallCategory,
		description: "can create ConfigMaps",
		check: func() error {
			return hc.checkCanCreate(hc.ControlPlaneNamespace, "", "v1", "ConfigMap")
		},
	})
}

func (hc *HealthChecker) addLinkerdControlPlaneExistenceChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdControlPlaneExistenceCategory,
		description: "control plane namespace exists",
		fatal:       true,
		check: func() error {
			return hc.checkNamespace(hc.ControlPlaneNamespace, true)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdControlPlaneExistenceCategory,
		description:   "controller pod is running",
		retryDeadline: hc.RetryDeadline,
		fatal:         true,
		check: func() error {
			var err error
			hc.controlPlanePods, err = hc.kubeAPI.GetPodsByNamespace(hc.httpClient, hc.ControlPlaneNamespace)
			if err != nil {
				return err
			}
			return checkControllerRunning(hc.controlPlanePods)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdControlPlaneExistenceCategory,
		description: "can initialize the client",
		fatal:       true,
		check: func() (err error) {
			if hc.APIAddr != "" {
				hc.apiClient, err = public.NewInternalClient(hc.ControlPlaneNamespace, hc.APIAddr)
			} else {
				hc.apiClient, err = public.NewExternalClient(hc.ControlPlaneNamespace, hc.kubeAPI)
			}
			return
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdControlPlaneExistenceCategory,
		description:   "can query the control plane API",
		retryDeadline: hc.RetryDeadline,
		fatal:         true,
		check: func() error {
			_, err := version.GetServerVersion(hc.apiClient)
			return err
		},
	})
}

func (hc *HealthChecker) addLinkerdAPIChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdAPICategory,
		description:   "control plane pods are ready",
		retryDeadline: hc.RetryDeadline,
		fatal:         true,
		check: func() error {
			var err error
			hc.controlPlanePods, err = hc.kubeAPI.GetPodsByNamespace(hc.httpClient, hc.ControlPlaneNamespace)
			if err != nil {
				return err
			}
			return validateControlPlanePods(hc.controlPlanePods)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdAPICategory,
		description:   "can query the control plane API",
		fatal:         true,
		retryDeadline: hc.RetryDeadline,
		checkRPC: func() (*healthcheckPb.SelfCheckResponse, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return hc.apiClient.SelfCheck(ctx, &healthcheckPb.SelfCheckRequest{})
		},
	})
}

func (hc *HealthChecker) addLinkerdServiceProfileChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdServiceProfileCategory,
		description: "no invalid service profiles",
		warning:     true,
		check: func() error {
			return hc.validateServiceProfiles()
		},
	})
}

func (hc *HealthChecker) addLinkerdDataPlaneExistenceChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdDataPlaneExistenceCategory,
		description: "data plane namespace exists",
		fatal:       true,
		check: func() error {
			return hc.checkNamespace(hc.DataPlaneNamespace, true)
		},
	})
}

func (hc *HealthChecker) addLinkerdDataPlaneChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdDataPlaneCategory,
		description:   "data plane proxies are ready",
		retryDeadline: hc.RetryDeadline,
		fatal:         true,
		check: func() error {
			pods, err := hc.getDataPlanePods()
			if err != nil {
				return err
			}

			return validateDataPlanePods(pods, hc.DataPlaneNamespace)
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:      LinkerdDataPlaneCategory,
		description:   "data plane proxy metrics are present in Prometheus",
		retryDeadline: hc.RetryDeadline,
		check: func() error {
			pods, err := hc.getDataPlanePods()
			if err != nil {
				return err
			}

			return validateDataPlanePodReporting(pods)
		},
	})
}

func (hc *HealthChecker) addLinkerdVersionChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdVersionCategory,
		description: "can determine the latest version",
		fatal:       true,
		check: func() (err error) {
			if hc.VersionOverride != "" {
				hc.latestVersion = hc.VersionOverride
			} else {
				// The UUID is only known to the web process. At some point we may want
				// to consider providing it in the Public API.
				uuid := "unknown"
				for _, pod := range hc.controlPlanePods {
					if strings.Split(pod.Name, "-")[0] == "web" {
						for _, container := range pod.Spec.Containers {
							if container.Name == "web" {
								for _, arg := range container.Args {
									if strings.HasPrefix(arg, "-uuid=") {
										uuid = strings.TrimPrefix(arg, "-uuid=")
									}
								}
							}
						}
					}
				}
				hc.latestVersion, err = version.GetLatestVersion(uuid, "cli")
			}
			return
		},
	})

	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdVersionCategory,
		description: "cli is up-to-date",
		warning:     true,
		check: func() error {
			return version.CheckClientVersion(hc.latestVersion)
		},
	})
}

func (hc *HealthChecker) addLinkerdControlPlaneVersionChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdControlPlaneVersionCategory,
		description: "control plane is up-to-date",
		warning:     true,
		check: func() error {
			return version.CheckServerVersion(hc.apiClient, hc.latestVersion)
		},
	})
}

func (hc *HealthChecker) addLinkerdDataPlaneVersionChecks() {
	hc.checkers = append(hc.checkers, &checker{
		category:    LinkerdDataPlaneVersionCategory,
		description: "data plane is up-to-date",
		warning:     true,
		check: func() error {
			pods, err := hc.getDataPlanePods()
			if err != nil {
				return err
			}

			for _, pod := range pods {
				if pod.ProxyVersion != hc.latestVersion {
					return fmt.Errorf("%s is running version %s but the latest version is %s",
						pod.Name, pod.ProxyVersion, hc.latestVersion)
				}
			}
			return nil
		},
	})
}

// Add adds an arbitrary checker. This should only be used for testing. For
// production code, pass in the desired set of checks when calling
// NewHeathChecker.
func (hc *HealthChecker) Add(category, description string, check func() error) {
	hc.checkers = append(hc.checkers, &checker{
		category:    category,
		description: description,
		check:       check,
	})
}

// RunChecks runs all configured checkers, and passes the results of each
// check to the observer. If a check fails and is marked as fatal, then all
// remaining checks are skipped. If at least one check fails, RunChecks returns
// false; if all checks passed, RunChecks returns true.  Checks which are
// designated as warnings will not cause RunCheck to return false, however.
func (hc *HealthChecker) RunChecks(observer checkObserver) bool {
	success := true

	for _, checker := range hc.checkers {
		if checker.check != nil {
			if !hc.runCheck(checker, observer) {
				if !checker.warning {
					success = false
				}
				if checker.fatal {
					break
				}
			}
		}

		if checker.checkRPC != nil {
			if !hc.runCheckRPC(checker, observer) {
				if !checker.warning {
					success = false
				}
				if checker.fatal {
					break
				}
			}
		}
	}

	return success
}

func (hc *HealthChecker) runCheck(c *checker, observer checkObserver) bool {
	for {
		err := c.check()
		checkResult := &CheckResult{
			Category:    c.category,
			Description: c.description,
			Warning:     c.warning,
			Err:         err,
		}

		if err != nil && time.Now().Before(c.retryDeadline) {
			checkResult.Retry = true
			observer(checkResult)
			time.Sleep(retryWindow)
			continue
		}

		observer(checkResult)
		return err == nil
	}
}

func (hc *HealthChecker) runCheckRPC(c *checker, observer checkObserver) bool {
	checkRsp, err := c.checkRPC()
	observer(&CheckResult{
		Category:    c.category,
		Description: c.description,
		Warning:     c.warning,
		Err:         err,
	})
	if err != nil {
		return false
	}

	for _, check := range checkRsp.Results {
		var err error
		if check.Status != healthcheckPb.CheckStatus_OK {
			err = fmt.Errorf(check.FriendlyMessageToUser)
		}
		observer(&CheckResult{
			Category:    fmt.Sprintf("%s[%s]", c.category, check.SubsystemName),
			Description: check.CheckDescription,
			Warning:     c.warning,
			Err:         err,
		})
		if err != nil {
			return false
		}
	}

	return true
}

// PublicAPIClient returns a fully configured public API client. This client is
// only configured if the KubernetesAPIChecks and LinkerdAPIChecks are
// configured and run first.
func (hc *HealthChecker) PublicAPIClient() pb.ApiClient {
	return hc.apiClient
}

func (hc *HealthChecker) checkNamespace(namespace string, shouldExist bool) error {
	exists, err := hc.kubeAPI.NamespaceExists(hc.httpClient, namespace)
	if err != nil {
		return err
	}
	if shouldExist && !exists {
		return fmt.Errorf("The \"%s\" namespace does not exist", namespace)
	}
	if !shouldExist && exists {
		return fmt.Errorf("The \"%s\" namespace already exists", namespace)
	}
	return nil
}

func (hc *HealthChecker) getDataPlanePods() ([]*pb.Pod, error) {
	req := &pb.ListPodsRequest{}
	if hc.DataPlaneNamespace != "" {
		req.Namespace = hc.DataPlaneNamespace
	}

	resp, err := hc.apiClient.ListPods(context.Background(), req)
	if err != nil {
		return nil, err
	}

	pods := make([]*pb.Pod, 0)
	for _, pod := range resp.GetPods() {
		if pod.ControllerNamespace == hc.ControlPlaneNamespace {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (hc *HealthChecker) checkCanCreate(namespace, group, version, resource string) error {
	if hc.clientset == nil {
		var err error
		hc.clientset, err = kubernetes.NewForConfig(hc.kubeAPI.Config)
		if err != nil {
			return err
		}
	}

	auth := hc.clientset.AuthorizationV1beta1()

	sar := &authorizationapi.SelfSubjectAccessReview{
		Spec: authorizationapi.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationapi.ResourceAttributes{
				Namespace: namespace,
				Verb:      "create",
				Group:     group,
				Version:   version,
				Resource:  resource,
			},
		},
	}

	response, err := auth.SelfSubjectAccessReviews().Create(sar)
	if err != nil {
		return err
	}

	if !response.Status.Allowed {
		if len(response.Status.Reason) > 0 {
			return fmt.Errorf("Missing permissions to create %s: %v", resource, response.Status.Reason)
		}
		return fmt.Errorf("Missing permissions to create %s", resource)
	}
	return nil
}

func (hc *HealthChecker) validateServiceProfiles() error {
	if hc.clientset == nil {
		var err error
		hc.clientset, err = kubernetes.NewForConfig(hc.kubeAPI.Config)
		if err != nil {
			return err
		}
	}

	if hc.spClientset == nil {
		var err error
		hc.spClientset, err = spclient.NewForConfig(hc.kubeAPI.Config)
		if err != nil {
			return err
		}
	}

	svcProfiles, err := hc.spClientset.LinkerdV1alpha1().ServiceProfiles(hc.ControlPlaneNamespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, p := range svcProfiles.Items {
		nameParts := strings.Split(p.Name, ".")
		if len(nameParts) != 2+len(clusterZoneSuffix) {
			return fmt.Errorf("ServiceProfile \"%s\" has invalid name (must be \"<service>.<namespace>.svc.cluster.local\")", p.Name)
		}
		for i, part := range nameParts[2:] {
			if part != clusterZoneSuffix[i] {
				return fmt.Errorf("ServiceProfile \"%s\" has invalid name (must be \"<service>.<namespace>.svc.cluster.local\")", p.Name)
			}
		}
		service := nameParts[0]
		namespace := nameParts[1]
		_, err := hc.clientset.Core().Services(namespace).Get(service, meta_v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("ServiceProfile \"%s\" has unknown service: %s", p.Name, err)
		}
		for _, route := range p.Spec.Routes {
			if route.Name == "" {
				return fmt.Errorf("ServiceProfile \"%s\" has a route with no name", p.Name)
			}
			if route.Condition == nil {
				return fmt.Errorf("ServiceProfile \"%s\" has a route with no condition", p.Name)
			}
			err = profiles.ValidateRequestMatch(route.Condition)
			if err != nil {
				return fmt.Errorf("ServiceProfile \"%s\" has a route with an invalid condition: %s", p.Name, err)
			}
			for _, rc := range route.ResponseClasses {
				if rc.Condition == nil {
					return fmt.Errorf("ServiceProfile \"%s\" has a response class with no condition", p.Name)
				}
				err = profiles.ValidateResponseMatch(rc.Condition)
				if err != nil {
					return fmt.Errorf("ServiceProfile \"%s\" has a response class with an invalid condition: %s", p.Name, err)
				}
			}
		}
	}
	return nil
}

func getPodStatuses(pods []v1.Pod) map[string][]v1.ContainerStatus {
	statuses := make(map[string][]v1.ContainerStatus)

	for _, pod := range pods {
		if pod.Status.Phase == v1.PodRunning {
			parts := strings.Split(pod.Name, "-")
			name := strings.Join(parts[1:len(parts)-2], "-")
			if _, found := statuses[name]; !found {
				statuses[name] = make([]v1.ContainerStatus, 0)
			}
			statuses[name] = append(statuses[name], pod.Status.ContainerStatuses...)
		}
	}

	return statuses
}

func validateControlPlanePods(pods []v1.Pod) error {
	statuses := getPodStatuses(pods)

	names := []string{"controller", "prometheus", "web", "grafana"}
	if _, found := statuses["ca"]; found {
		names = append(names, "ca")
	}
	if _, found := statuses["proxy-injector"]; found {
		names = append(names, "proxy-injector")
	}

	for _, name := range names {
		containers, found := statuses[name]
		if !found {
			return fmt.Errorf("No running pods for \"linkerd-%s\"", name)
		}
		for _, container := range containers {
			if !container.Ready {
				return fmt.Errorf("The \"linkerd-%s\" pod's \"%s\" container is not ready", name,
					container.Name)
			}
		}
	}

	return nil
}

func checkControllerRunning(pods []v1.Pod) error {
	statuses := getPodStatuses(pods)
	if _, ok := statuses["controller"]; !ok {
		return errors.New("No running pods for \"linkerd-controller\"")
	}
	return nil
}

func validateDataPlanePods(pods []*pb.Pod, targetNamespace string) error {
	if len(pods) == 0 {
		msg := fmt.Sprintf("No \"%s\" containers found", k8s.ProxyContainerName)
		if targetNamespace != "" {
			msg += fmt.Sprintf(" in the \"%s\" namespace", targetNamespace)
		}
		return fmt.Errorf(msg)
	}

	for _, pod := range pods {
		if pod.Status != "Running" {
			return fmt.Errorf("The \"%s\" pod is not running",
				pod.Name)
		}

		if !pod.ProxyReady {
			return fmt.Errorf("The \"%s\" container in the \"%s\" pod is not ready",
				k8s.ProxyContainerName, pod.Name)
		}
	}

	return nil
}

func validateDataPlanePodReporting(pods []*pb.Pod) error {
	notInPrometheus := []string{}

	for _, p := range pods {
		// the `Added` field indicates the pod was found in Prometheus
		if !p.Added {
			notInPrometheus = append(notInPrometheus, p.Name)
		}
	}

	errMsg := ""
	if len(notInPrometheus) > 0 {
		errMsg = fmt.Sprintf("Data plane metrics not found for %s.", strings.Join(notInPrometheus, ", "))
	}

	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}

	return nil
}
