package inspector

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/yeahx/kubeapi-inspector/pkg/kubeclient"
	"github.com/yeahx/kubeapi-inspector/pkg/utils"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"net"
)

type Inspector struct {
	client               *kubeclient.KubeClient
	schemaMap            map[string]*openapi_v2.Schema
	sensitiveInfoRegexps []*regexp.Regexp
	sensitiveCheckFunc   func(p string) bool
	pathBodyParameterMap map[string]*openapi_v2.BodyParameter    // map[pathItem.Name]*BodyParameter
	pathPathParameterMap map[string]*openapi_v2.NonBodyParameter // map[pathItem.Name]*NonBodyParameter
}

// DNSServiceInfo
type DNSServiceInfo struct {
	Target   string
	Port     uint16
	Priority uint16
	Weight   uint16
}

func NewInspector(client *kubeclient.KubeClient, sensitiveCheckFunc func(string) bool) *Inspector {
	regs := compileRegexps(sensitivePatterns)
	fmt.Printf("[*] Load %d sensitive pattern\n", len(regs))

	i := &Inspector{client: client, sensitiveInfoRegexps: regs, sensitiveCheckFunc: sensitiveCheckFunc}

	if i.sensitiveCheckFunc == nil {
		i.sensitiveCheckFunc = func(p string) bool {
			for _, re := range i.sensitiveInfoRegexps {
				if r := re.MatchString(p); r {
					return true
				}
			}
			return false
		}
	}

	return i
}

// DiscoveryAPIServiceBySRV
func (i *Inspector) DiscoveryAPIServiceBySRV() ([]DNSServiceInfo, error) {
	fmt.Printf("[*] Starting discovery api service by coredns\n")

	dnsPatterns := []string{
		"any.any.svc.cluster.local.",
		"any.any.any.svc.cluster.local.",
	}

	var results []DNSServiceInfo
	for _, dns := range dnsPatterns {
		_, srvs, err := net.LookupSRV("", "", dns)
		if err != nil {
			fmt.Printf("[*] DNS Query Eror %s: %s\n", dns, err.Error())
			continue
		}

		for _, srv := range srvs {

			results = append(results, DNSServiceInfo{
				Target:   srv.Target,
				Port:     srv.Port,
				Priority: srv.Priority,
				Weight:   srv.Weight,
			})
		}
	}

	// print result
	for _, r := range results {
		fmt.Printf("[+] Service: %s:%d\n",
			r.Target, r.Port)
	}

	return results, nil
}

func (i *Inspector) DetectObjectLeak(group, version, resource string) error {
	var errors []error

	// is subresource should be skip.
	if len(strings.Split(resource, "/")) > 1 {
		return nil
	}

	uri := utils.MakeUri(group, version, resource)

	baseRes, err := i.client.List(uri)
	if err != nil {
		// 403
		if !apierrors.IsForbidden(err) {
			errors = append(errors, err)
			fmt.Printf("[-] verb List access apiserver failed: %v", err)
		}
	}

	// diff object?
	// baseObj, baseLen, err := bytesToUnstructuredList(baseRes)
	_, baseLen, err := utils.BytesToUnstructuredList(baseRes)
	//if err != nil {
	//	errors = append(errors, err)
	//}

	watchRes, err := i.client.Watch(uri)
	if err != nil {
		if !apierrors.IsForbidden(err) {
			errors = append(errors, err)
			fmt.Printf("[-] verb Watch access apiserver failed: %v", err)
		}
	}

	watchObj, watchLen, err := utils.WatchResToUnstructuredList(watchRes)
	// diff

	if watchLen > baseLen {
		utils.RemoveObjectFields(watchObj, uri)
		utils.PrintResult(utils.MakeUri(group, version, resource), "Watch", watchObj)
		return nil
	}

	dcRes, err := i.client.DeleteCollection(uri)
	if err != nil {
		if !apierrors.IsForbidden(err) {
			errors = append(errors, err)
			fmt.Printf("[-] verb DeleteCollection access apiserver failed: %v", err)
		}
		// 403 skip
	}

	dcObj, dcLen, err := utils.BytesToUnstructuredList(dcRes)

	if dcLen > baseLen {
		utils.RemoveObjectFields(dcObj, uri)
		utils.PrintResult(utils.MakeUri(group, version, resource), "DeleteCollection", watchObj)
		return nil
	}

	// lres wres

	return nil
}

func (i *Inspector) DetectSensitiveField(group, version, resource string) error {
	uri := utils.MakeUri(group, version, resource)
	sensitiveFields := make(map[string]bool)
	path := []string{"$"}

	bodyParameter, ok := i.pathBodyParameterMap[uri]
	if !ok {
		return errors.New(fmt.Sprintf("%s not found body parameter", uri))
	}
	refName := strings.TrimPrefix(bodyParameter.GetSchema().XRef, "#/definitions/")
	refSchema, exists := i.schemaMap[refName]
	if !exists {
		return errors.New(fmt.Sprintf("%s not found ref %s schema", uri, refName))
	}

	resolveSchema(refSchema, i.schemaMap, path, sensitiveFields, i.sensitiveCheckFunc)
	//schemaMap := make(map[string]*openapi_v2.Schema)

	return nil
}

func (i *Inspector) ParseDocument(doc *openapi_v2.Document) error {
	i.schemaMap = make(map[string]*openapi_v2.Schema)
	i.pathBodyParameterMap = make(map[string]*openapi_v2.BodyParameter)
	i.pathPathParameterMap = make(map[string]*openapi_v2.NonBodyParameter)

	if doc.GetDefinitions() == nil {
		return errors.New("openapi document definitions is nil")
	}

	if properties := doc.GetDefinitions().GetAdditionalProperties(); properties != nil {
		for _, p := range doc.GetDefinitions().GetAdditionalProperties() {
			i.schemaMap[p.GetName()] = p.GetValue()
		}
	}

	if doc.GetPaths() == nil {
		return errors.New("openapi document paths is nil")
	}

	for _, pathItem := range doc.GetPaths().GetPath() {
		if pathItem.GetValue() != nil && pathItem.GetValue().GetPost() != nil {
			bodyParam, err := getPostBodyParameter(pathItem.GetValue().GetPost().GetParameters())
			if err != nil {
				continue
			}
			i.pathBodyParameterMap[pathItem.Name] = bodyParam
		}

	}

	fmt.Printf("[*] Parse openapi schema success.\n")

	return nil
}

// resolveSchema 解析schema，处理可能的$xref引用
func resolveSchema(schema *openapi_v2.Schema, definitions map[string]*openapi_v2.Schema,
	path []string, sensitiveFields map[string]bool, checkFunc func(ppName string) bool) {
	if schema == nil {
		return
	}

	// 解析properties
	if schema.Properties != nil {
		for _, pair := range schema.Properties.AdditionalProperties {
			propertyName := pair.Name
			propertySchema := pair.Value

			// 更新路径
			currentPath := append(path, propertyName)

			// 检查是否为敏感字段
			if checkFunc(propertyName) {
				fullPath := strings.Join(currentPath, ".")
				fmt.Printf("[+] sensitive field found: %s\n", fullPath)
				sensitiveFields[fullPath] = true
			}

			// check schema has xref
			if propertySchema.XRef != "" {
				refName := strings.TrimPrefix(propertySchema.XRef, "#/definitions/")
				refSchema, exists := definitions[refName]
				if exists {
					resolveSchema(refSchema, definitions, currentPath, sensitiveFields, checkFunc)
				}
			} else {
				// 递归解析非引用的schema
				resolveSchema(propertySchema, definitions, currentPath, sensitiveFields, checkFunc)
			}
		}
	}
}

func getPostBodyParameter(items []*openapi_v2.ParametersItem) (*openapi_v2.BodyParameter, error) {
	if items == nil {
		return nil, errors.New("ParametersItem is nil")
	}

	for _, item := range items {
		if item.GetParameter() != nil && item.GetParameter().GetBodyParameter() != nil &&
			item.GetParameter().GetBodyParameter().GetName() == "body" {
			return item.GetParameter().GetBodyParameter(), nil
		} else {
			print(item.GetParameter())
		}
	}

	return nil, nil
}

func compileRegexps(parttens []string) []*regexp.Regexp {
	var regexps []*regexp.Regexp
	for _, partten := range parttens {
		re, err := regexp.Compile(partten)
		if err != nil {

		}
		regexps = append(regexps, re)
	}

	return regexps
}
