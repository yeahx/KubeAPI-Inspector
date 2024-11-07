package options

import (
	"fmt"
	apiextensionsserver "k8s.io/apiextensions-apiserver/pkg/apiserver"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	utilversion "k8s.io/apiserver/pkg/util/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
	"net"
	generatedopenapi "workshop/pkg/apis/workshop/v1alpha1"
	"workshop/pkg/server"
	//"sigs.k8s.io/metrics-server/pkg/api"
	//generatedopenapi "sigs.k8s.io/metrics-server/pkg/api/generated/openapi"
	//"sigs.k8s.io/metrics-server/pkg/server"
)

type Options struct {
	// genericoptions.RecomendedOptions - EtcdOptions
	GenericServerRunOptions *genericoptions.ServerRunOptions
	SecureServing           *genericoptions.SecureServingOptionsWithLoopback
	Authentication          *genericoptions.DelegatingAuthenticationOptions
	Authorization           *genericoptions.DelegatingAuthorizationOptions
	Etcd                    *genericoptions.EtcdOptions
	Logging                 *logs.Options

	Kubeconfig string

	// Only to be used to for testing
	DisableAuthForTesting bool
}

func (o *Options) Validate() []error {
	var errors []error
	err := logsapi.ValidateAndApply(o.Logging, nil)
	if err != nil {
		errors = append(errors, err)
	}
	if errs := o.GenericServerRunOptions.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	return errors
}

func (o *Options) Flags() (fs flag.NamedFlagSets) {
	msfs := fs.FlagSet("mutlicluster-server")
	msfs.StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)")

	o.GenericServerRunOptions.AddUniversalFlags(fs.FlagSet("generic"))
	o.SecureServing.AddFlags(fs.FlagSet("apiserver secure serving"))
	o.Authentication.AddFlags(fs.FlagSet("apiserver authentication"))
	o.Authorization.AddFlags(fs.FlagSet("apiserver authorization"))
	o.Etcd.AddFlags(fs.FlagSet("etcd"))
	logsapi.AddFlags(o.Logging, fs.FlagSet("logging"))

	return fs
}

// NewOptions constructs a new set of default options for metrics-server.
func NewOptions() *Options {
	return &Options{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		SecureServing:           genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication:          genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:           genericoptions.NewDelegatingAuthorizationOptions(),
		Etcd:                    genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig("/workshop/test", nil)),
		Logging:                 logs.NewOptions(),
	}
}

func (o Options) ServerConfig() (*server.Config, error) {
	apiserver, err := o.ApiserverConfig()
	if err != nil {
		return nil, err
	}
	restConfig, err := o.restConfig()
	if err != nil {
		return nil, err
	}
	return &server.Config{
		Apiserver: apiserver,
		Rest:      restConfig,
	}, nil
}

func (o Options) ApiserverConfig() (*genericapiserver.Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(server.Codecs)

	if err := o.GenericServerRunOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if !o.DisableAuthForTesting {
		if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
			return nil, err
		}
		if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
			return nil, err
		}
	}

	if err := o.Etcd.ApplyWithStorageFactoryTo(serverstorage.NewDefaultStorageFactory(
		o.Etcd.StorageConfig,
		o.Etcd.DefaultStorageMediaType,
		server.Codecs,
		serverstorage.NewDefaultResourceEncodingConfig(server.Scheme),
		apiextensionsserver.DefaultAPIResourceConfigSource(),
		nil,
	), serverConfig); err != nil {
		return nil, err
	}

	// versionGet := version.Get()
	// enable OpenAPI schemas
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(server.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(server.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "mutlicluster-server"
	serverConfig.OpenAPIV3Config.Info.Title = "mutlicluster-server"
	serverConfig.OpenAPIConfig.Info.Version = "1"
	serverConfig.OpenAPIV3Config.Info.Version = "1"
	serverConfig.EffectiveVersion = utilversion.DefaultKubeEffectiveVersion()

	return serverConfig, nil
}

func (o Options) restConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error
	if len(o.Kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: o.Kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		config, err = loader.ClientConfig()
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to construct lister client config: %v", err)
	}
	// Use protobufs for communication with apiserver
	config.ContentType = "application/vnd.kubernetes.protobuf"
	err = rest.SetKubernetesDefaults(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
