package utils

import (
	"encoding/json"
	"fmt"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

var (
	nativeApiGroup = map[string]interface{}{
		"apps":                         struct{}{},
		"batch":                        struct{}{},
		"policy":                       struct{}{},
		"extensions":                   struct{}{},
		"autoscaling":                  struct{}{},
		"node.k8s.io":                  struct{}{},
		"events.k8s.io":                struct{}{},
		"storage.k8s.io":               struct{}{},
		"cli.k8s.io":                   struct{}{},
		"discovery.k8s.io":             struct{}{},
		"scheduling.k8s.io":            struct{}{},
		"networking.k8s.io":            struct{}{},
		"coordination.k8s.io":          struct{}{},
		"certificates.k8s.io":          struct{}{},
		"apiextensions.k8s.io":         struct{}{},
		"authorization.k8s.io":         struct{}{},
		"authentication.k8s.io":        struct{}{},
		"apiregistration.k8s.io":       struct{}{},
		"rbac.authorization.k8s.io":    struct{}{},
		"admissionregistration.k8s.io": struct{}{},
	}
)

func MakeUri(g, v, r string) string {
	return fmt.Sprintf("/apis/%s/%s/%s", g, v, strings.TrimRight(r, "/"))
}

func IsStatusSubresource(res string) bool {
	parts := strings.Split(res, "/")
	if len(parts) == 2 {
		return parts[1] == "status"
	}

	return false
}

func IsNativeAPI(group string) bool {
	var f bool

	_, ok := nativeApiGroup[group]

	if group == "" || ok {
		f = true
	}

	return f
}

func ConvertToPolicyRule(status authorizationv1.SubjectRulesReviewStatus) []rbacv1.PolicyRule {
	ret := []rbacv1.PolicyRule{}
	for _, resource := range status.ResourceRules {
		ret = append(ret, rbacv1.PolicyRule{
			Verbs:         resource.Verbs,
			APIGroups:     resource.APIGroups,
			Resources:     resource.Resources,
			ResourceNames: resource.ResourceNames,
		})
	}

	for _, nonResource := range status.NonResourceRules {
		ret = append(ret, rbacv1.PolicyRule{
			Verbs:           nonResource.Verbs,
			NonResourceURLs: nonResource.NonResourceURLs,
		})
	}

	return ret
}

func CombineResourceGroup(resource, group string) string {
	if len(resource) == 0 {
		return ""
	}
	parts := strings.SplitN(resource, "/", 2)
	combine := parts[0]

	if group != "" {
		combine = combine + "." + group
	}

	if len(parts) == 2 {
		combine = combine + "/" + parts[1]
	}
	return combine
}

// bytesToUnstructuredObj s
func BytesToUnstructuredList(bytes []byte) (*unstructured.UnstructuredList, int, error) {
	obj := &unstructured.Unstructured{}
	l := &unstructured.UnstructuredList{}

	if err := obj.UnmarshalJSON(bytes); err != nil {
		return nil, 0, err
	}

	l, err := obj.ToList()
	if err != nil {
		return nil, 0, err
	}

	return l, len(l.Items), nil
}

func WatchResToUnstructuredList(bytes []byte) (*unstructured.UnstructuredList, int, error) {
	l := &unstructured.UnstructuredList{}
	if len(bytes) == 0 {
		return nil, 0, nil
	}
	parts := strings.Split(string(bytes), "\n")
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		item := make(map[string]interface{})

		_ = json.Unmarshal([]byte(part), &item)

		object, ok := item["object"]
		if !ok {
			continue
		}
		content, ok := object.(map[string]interface{})
		if !ok {
			continue
		}

		obj := &unstructured.Unstructured{}
		obj.SetUnstructuredContent(content)
		obj.GetObjectKind().GroupVersionKind()

		l.Items = append(l.Items, *obj)

	}

	if len(l.Items) > 0 {
		i := l.Items[0]
		gvk := i.GetObjectKind().GroupVersionKind()
		gvk.Kind = fmt.Sprintf("%sList", gvk.Kind)
		l.SetGroupVersionKind(gvk)
	}

	return l, len(l.Items), nil
}

// removeOjbectFields remove redundant fields
func RemoveObjectFields(list *unstructured.UnstructuredList, uri string) {
	_ = list.EachListItem(func(object runtime.Object) error {
		unstructuredObj, ok := object.(*unstructured.Unstructured)
		if !ok {
			return nil
		}
		unstructuredObj.SetManagedFields(nil)
		//unstructuredObj.SetAnnotations()
		unstructured.RemoveNestedField(unstructuredObj.UnstructuredContent(), "metadata",
			"annotations", "kubectl.kubernetes.io/last-applied-configuration")
		//fmt.Printf("%s, leak data: %v", uri, unstructuredObj)
		return nil
	})
}

func PrintResult(uri, verb string, object any) {
	fmt.Printf("[+] Path: %s found broken access control by %s verb.\nCommand example: %s\n", uri, verb, generateCurlExample(verb, uri))
	indent, _ := json.MarshalIndent(object, "", "    ")
	fmt.Printf("[+] leak objects: %v \n", string(indent))
}

func generateCurlExample(verb, uri string) string {
	output := ""
	switch verb {
	case "Watch":
		output = fmt.Sprintf("curl -H\"Authorization: Bearer $token\" -k https://kubernetes.default%s?watch=true&timeoutSeconds=2", uri)
	case "DeleteCollection":
		output = fmt.Sprintf("curl -H\"Authorization: Bearer $token\" -k https://kubernetes.default%s?dryRun=All", uri)
	}

	return output
}
