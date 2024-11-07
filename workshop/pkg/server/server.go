package server

import (
	metainternal "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	genericapiserver "k8s.io/apiserver/pkg/server"
	workshopinternal "workshop/pkg/apis/workshop"
	"workshop/pkg/apis/workshop/v1alpha1"
)

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	_ = v1alpha1.Install(Scheme)
	_ = workshopinternal.Install(Scheme)
	_ = metainternal.AddToScheme(Scheme)
	metav1.AddToGroupVersion(Scheme, metav1.SchemeGroupVersion)
	internalGroupVersion := schema.GroupVersion{Group: "", Version: "v1"}
	metav1.AddToGroupVersion(Scheme, internalGroupVersion)

}

// Server contains state for a Kubernetes cluster master/api server.
type Server struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

// NewServer returns a new instance of Server from the given config.
func NewServer(apiserver *genericapiserver.GenericAPIServer) *Server {
	return &Server{
		GenericAPIServer: apiserver,
	}
}

func (s *Server) RunUntil(stopCh <-chan struct{}) error {
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	// Start informers
	//go s.nodes.Run(stopCh)
	//go s.pods.Run(stopCh)

	// Ensure cache is up to date
	//ok := cache.WaitForCacheSync(stopCh, s.nodes.HasSynced)
	//if !ok {
	//	return nil
	//}

	return s.GenericAPIServer.PrepareRun().RunWithContext(wait.ContextForChannel(stopCh))
}
