package app

import (
	"flag"
	"fmt"

	"github.com/yeahx/kubeapi-inspector/pkg/inspector"
	"github.com/yeahx/kubeapi-inspector/pkg/kubeclient"
	"github.com/yeahx/kubeapi-inspector/pkg/utils"
)

func Run() {
	var kubeconfig, namespace, token, server string
	var skipCheckSensitiveField, insecureSkipTLS bool

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	//flag.BoolVar(&skipNativeAPI, "skip-native-api", true, "")
	flag.StringVar(&namespace, "namespace", "", "")
	flag.StringVar(&token, "token", "", "token for access apiserver. Only required if out-of-cluster.")
	flag.StringVar(&server, "server", "", "target apiserver address.")
	flag.BoolVar(&skipCheckSensitiveField, "skipCheckSensitiveField", false, "if true skip check resource sensitive field")
	flag.BoolVar(&insecureSkipTLS, "insecure-skip-tls-verify", false, "if true, skip TLS verification for Kubernetes API server")
	flag.Parse()

	kubeClient, err := kubeclient.NewKubeClient(kubeconfig, namespace, insecureSkipTLS)
	if err != nil {
		fmt.Printf("[-] Failed to create kubeclient: %s\nmake sure kubeconfig is valided.", err)
		return
	}

	doc, err := kubeClient.DownloadOpenApiSchema()
	if err != nil {
		fmt.Printf("[-] Failed to download openapi schema: %s\n, will be skip sensitive field test.", err)
		skipCheckSensitiveField = true
	}

	err = kubeClient.FetchCRDApis()
	if err != nil {
		fmt.Printf("[-] Failed to fetch CRD apis: %s\n", err)
	}

	scan := inspector.NewInspector(kubeClient, nil)
	err = scan.ParseDocument(doc)
	if err != nil {
		fmt.Printf("[-] Failed to parse document: %s\n, will be skip sensitive field test.", err)
		skipCheckSensitiveField = true
	}

	_, err = scan.DiscoveryAPIServiceBySRV()
	if err != nil {
		fmt.Printf("[-] Failed to discovery apiservice by srv: %s\n", err)
	}

	for k, v := range kubeClient.Resources {
		if utils.IsStatusSubresource(v.Name) {
			continue
		}

		fmt.Printf("[*] Starting validation for %s, group: %s, version: %s, resource: %s,\n", k, v.GroupName, v.Version, v.Name)
		if !skipCheckSensitiveField {
			err := scan.DetectSensitiveField(v.GroupName, v.Version, v.Name)
			if err != nil {
				fmt.Printf("[-] Failed to detect sensitive field: %s\n", err)
			}
		}
		err = scan.DetectObjectLeak(v.GroupName, v.Version, v.Name)
		if err != nil {
			fmt.Printf("[-] Detect err: %v", err)
			return
		}
	}

	fmt.Println("[*] Done")
	return
}
