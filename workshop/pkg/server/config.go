package server

import (
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	restclient "k8s.io/client-go/rest"
	"workshop/pkg/apis/workshop/v1alpha1"
	clusterRegistry "workshop/pkg/server/registry/cluster"
)

type Config struct {
	Apiserver *genericapiserver.Config
	Rest      *restclient.Config
}

func (c Config) Complete() (*Server, error) {

	genericServer, err := c.Apiserver.Complete(nil).New("mutli-cluster server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := NewServer(genericServer)

	clusterStorage, err := clusterRegistry.NewClusterRest(Scheme, c.Apiserver.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	workshopApiGroupIfo := genericapiserver.NewDefaultAPIGroupInfo(v1alpha1.GroupName, Scheme, ParameterCodec, Codecs)
	workshopServerResources := map[string]rest.Storage{
		"clusters": clusterStorage,
	}

	workshopApiGroupIfo.VersionedResourcesStorageMap[v1alpha1.SchemeGroupVersion.Version] = workshopServerResources

	err = s.GenericAPIServer.InstallAPIGroup(&workshopApiGroupIfo)
	if err != nil {
		return nil, err
	}

	return s, nil
}
