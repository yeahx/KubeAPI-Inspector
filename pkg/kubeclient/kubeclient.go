package kubeclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/yeahx/kubeapi-inspector/pkg/utils"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"

	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	clientset       *kubernetes.Clientset
	doc             *openapi_v2.Document
	namespace       string
	DiscoveryClient discovery.DiscoveryInterface
	AuthClient      authorizationv1client.AuthorizationV1Interface
	Rules           []rbacv1.PolicyRule
	// [accounts.space.test.io] = Resource{}, [accounts.space.test.io/status] = Resource{}
	Resources map[string]Resource
}

type APIGroup string

type Resource struct {
	GroupName    string
	GroupVersion string
	Remote       bool
	*metav1.APIResource
}

// NewKubeClient creates a Kubernetes client.
func NewKubeClient(kubeconfig, namespace string, insecureSkipTLS bool) (*KubeClient, error) {
	var config *rest.Config
	var err error
	var inCluster bool

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	if namespace == "" {
		ns, err := os.ReadFile(namespaceFile)
		if err != nil {
			namespace = "default"
		} else {
			namespace = string(ns)
		}
	}

	if kubeconfig == "" {
		//home := os.Getenv("HOME")
		//kubeconfig = fmt.Sprintf("%s/.kube/config", home)

		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		inCluster = true
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from path %s: %w", kubeconfig, err)
		}

		// Set Insecure to true (skip TLS verification) only if the flag is set
		// If the flag is not set, set Insecure to false if certificate data or files are provided
		if insecureSkipTLS {
			config.Insecure = true
		} else if config.CAData != nil || config.CertData != nil || config.CAFile != "" || config.CertFile != "" || config.KeyFile != "" {
			config.Insecure = false
		} else {
			config.Insecure = true
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	kc := &KubeClient{clientset: clientset, namespace: namespace}

	kc.DiscoveryClient = kc.clientset.Discovery()
	kc.AuthClient = kc.clientset.AuthorizationV1()

	//ssr := &authenticationv1.SelfSubjectReview{}
	//res, err := kc.clientset.AuthenticationV1().SelfSubjectReviews().Create(context.TODO(), ssr, metav1.CreateOptions{})
	//if err != nil {
	//	return nil, err
	//}

	kc.Resources = make(map[string]Resource)

	// check rule review
	if err = kc.loadRBACPolicy(); err != nil {
		return nil, err
	}

	if inCluster {
		fmt.Printf("[*] Use in-cluster mode, Host: %s, Token %s", config.Host, config.BearerToken)
	} else {
		fmt.Printf("[*] Load %s config, Host: %s, Token: %s\n", kubeconfig, config.Host, config.BearerToken)
	}

	return kc, nil
}

func (kc *KubeClient) Get(ctx context.Context, uri string) ([]byte, error) {
	return kc.clientset.RESTClient().Get().RequestURI(uri).DoRaw(ctx)
}

func (kc *KubeClient) Watch(uri string) ([]byte, error) {
	params := "?watch=true&timeoutSeconds=2"

	uri = fmt.Sprintf("%s%s", uri, params)

	b, err := kc.Get(context.TODO(), uri)
	//fmt.Printf("%s", string(b))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (kc *KubeClient) List(uri string) ([]byte, error) {
	b, err := kc.Get(context.TODO(), uri)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// DeleteCollection use deletecollection verb to test, make sure target apiserver support dryRun mode.
func (kc *KubeClient) DeleteCollection(uri string) ([]byte, error) {
	params := "?dryRun=All"
	uri = fmt.Sprintf("%s%s", uri, params)

	b, err := kc.clientset.RESTClient().Delete().RequestURI(uri).DoRaw(context.TODO())
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (kc *KubeClient) GetClientSet() *kubernetes.Clientset {
	if kc.clientset == nil {
		return nil
	}

	return kc.clientset
}

// loadRBACPolicy load current accounts rbac rules.
func (kc *KubeClient) loadRBACPolicy() error {
	sar := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: kc.namespace,
		},
	}

	res, err := kc.AuthClient.SelfSubjectRulesReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Verbs: [Get,List]; APIGroup: space.test.io; Resources: [accounts, accounts/status]
	kc.Rules = utils.ConvertToPolicyRule(res.Status)
	return nil
}

func (kc *KubeClient) FetchCRDApis() error {
	fmt.Printf("[*] Starting to discovery apis\n")
	body, err := kc.clientset.RESTClient().Get().
		AbsPath("/apis").
		SetHeader("Accept", runtime.ContentTypeJSON).
		Do(context.TODO()).
		Raw()

	if err != nil {
		return err
	}

	apiGroupList := &metav1.APIGroupList{}

	err = json.Unmarshal(body, apiGroupList)
	if err != nil {
		return err
	}

	for _, group := range apiGroupList.Groups {
		if utils.IsNativeAPI(group.Name) {
			continue
		}
		for _, version := range group.Versions {
			resourceList, err := kc.DiscoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				log.Printf("could not retrieve resource list for group version %s: %v", version.GroupVersion, err)
				continue
			}
			for _, resource := range resourceList.APIResources {
				r := Resource{
					GroupName:    group.Name,
					GroupVersion: version.GroupVersion,
					Remote:       isRemoteApi(resource),
					APIResource:  resource.DeepCopy(),
				}

				r.Version = version.Version

				combine := utils.CombineResourceGroup(r.APIResource.Name, r.GroupName)

				if _, ok := kc.Resources[combine]; !ok {
					kc.Resources[combine] = r
				}
			}
		}
	}

	apisCount := len(kc.Resources)

	fmt.Printf("[*] Discovered %d custom apis\n", apisCount)

	return nil
}

func (kc *KubeClient) DownloadOpenApiSchema() (*openapi_v2.Document, error) {
	fmt.Printf("[*] Starting to download openapi definition\n")
	doc, err := kc.clientset.OpenAPISchema()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to get openapi schema: %v", err))
	}
	kc.doc = doc
	return kc.doc, nil
}

// isRemoteApi determines whether a given API resource is served by a remote API server.
func isRemoteApi(resource metav1.APIResource) bool {
	part := strings.Split(resource.Name, "/")

	return resource.StorageVersionHash == "" && len(part) == 1
}
