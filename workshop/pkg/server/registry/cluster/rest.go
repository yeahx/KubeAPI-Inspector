package cluster

import (
	"context"
	"fmt"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
	"strings"
	"workshop/pkg/apis/workshop/v1alpha1"
)

type ClusterRest struct {
	*genericregistry.Store
}

var _ rest.StandardStorage = &ClusterRest{}

//var _ rest.StorageWithReadiness = &clusterRest{}
//var _ rest.TableConvertor = &clusterRest{}

func (s *ClusterRest) ShortNames() []string {
	return []string{"cls"}
}

func (s *ClusterRest) NamespaceScoped() bool {
	return false
}

func getServiceAccountName(fullName string) string {
	parts := strings.Split(fullName, ":")
	if len(parts) == 4 && parts[1] == "serviceaccount" {
		return parts[len(parts)-1]
	}

	return fullName
}

func predicateListOptions(userInfo user.Info, options *metainternalversion.ListOptions) *metainternalversion.ListOptions {
	for _, g := range userInfo.GetGroups() {
		if g == user.SystemPrivilegedGroup {
			return options
		}
	}

	tenant := getServiceAccountName(userInfo.GetName())
	if options == nil {
		options = &metainternalversion.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("spec.tenant", tenant),
		}
	}

	if options.FieldSelector == nil {
		options.FieldSelector = fields.OneTermEqualSelector("spec.tenant", tenant)
		return options
	}
	options.FieldSelector = fields.AndSelectors(options.FieldSelector, fields.OneTermEqualSelector("spec.tenant", tenant))
	return options
}

func (s *ClusterRest) validateObjectTenant(ctx context.Context, userInfo user.Info, name string, options *metav1.GetOptions) (runtime.Object, error) {
	tenant := getServiceAccountName(userInfo.GetName())
	obj, err := s.Store.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	cls, ok := obj.(*v1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("object is not a Cluster")
	}

	for _, g := range userInfo.GetGroups() {
		if g == user.SystemPrivilegedGroup {
			return cls, nil
		}
	}

	if !(cls.Spec.Tenant == tenant) {
		return nil, fmt.Errorf("tenant not match")
	}

	return cls, nil
}

func (s *ClusterRest) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	userInfo, ok := genericapirequest.UserFrom(ctx)

	if !ok {
		return &v1alpha1.ClusterList{}, nil
	}

	newListOptions := predicateListOptions(userInfo, options)
	return s.Store.List(ctx, newListOptions)
}

func (s *ClusterRest) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	userInfo, ok := genericapirequest.UserFrom(ctx)
	if !ok {
		return &v1alpha1.Cluster{}, nil
	}

	return s.validateObjectTenant(ctx, userInfo, name, options)
}
func (s *ClusterRest) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	userInfo, ok := genericapirequest.UserFrom(ctx)
	if !ok {
		return nil, false, nil
	}

	_, err := s.validateObjectTenant(ctx, userInfo, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	return s.Store.Delete(ctx, name, deleteValidation, options)
}

func NewClusterRest(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*ClusterRest, error) {

	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &v1alpha1.Cluster{} },
		NewListFunc:               func() runtime.Object { return &v1alpha1.ClusterList{} },
		PredicateFunc:             nil,
		DefaultQualifiedResource:  v1alpha1.SchemeGroupVersion.WithResource("clusters").GroupResource(),
		SingularQualifiedResource: v1alpha1.SchemeGroupVersion.WithResource("cluster").GroupResource(),
		CreateStrategy:            strategy,
		UpdateStrategy:            strategy,
		DeleteStrategy:            strategy,
		TableConvertor:            rest.NewDefaultTableConvertor(schema.GroupResource{Resource: "cluster"}),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}

	r := &ClusterRest{Store: store}

	return r, nil
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*v1alpha1.Cluster)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Cluster")
	}

	fieldsSet := fields.Set{
		"metadata.name": apiserver.ObjectMeta.Name,
		"spec.tenant": apiserver.Spec.Tenant,
	}

	return labels.Set(apiserver.ObjectMeta.Labels), fieldsSet, nil
}

func NewStrategy(typer runtime.ObjectTyper) ClusterStrategy {
	return ClusterStrategy{typer, names.SimpleNameGenerator}
}

type ClusterStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (ClusterStrategy) NamespaceScoped() bool {
	return false
}

func (ClusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {}

func (ClusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {}

func (ClusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList { return nil }

// WarningsOnCreate returns warnings for the creation of the given object.
func (ClusterStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string { return nil }

func (ClusterStrategy) AllowCreateOnUpdate() bool { return false }

func (ClusterStrategy) AllowUnconditionalUpdate() bool { return false }

func (ClusterStrategy) Canonicalize(obj runtime.Object) {}

func (ClusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (ClusterStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
