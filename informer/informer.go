// Copyright © 2018 Stefan Bueringer
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package informer

import (
	"io/ioutil"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/lextoumbourou/goodhosts"

	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset
var serviceStore cache.Store
var serviceController cache.Controller
var ingressStore cache.Store
var ingressController cache.Controller
var clusterIPCIDR = "10.96.0.0/12"
var aliasMappingPath string
var templatePath string
var outputPath string
var hostsPath string
var defaultIngressHost string
var aliasMappings *AliasMappings

type AliasMapping struct {
	Source string `json:"source"`
	Targets []string `json:"targets"`
}

type AliasMappings struct {
	Mappings []AliasMapping `json:"mappings"`
}

type KubeConfig int

const (
	CLUSTER KubeConfig = iota
	LOCAL
)

var kubeConfigByName = map[string]KubeConfig{
	"CLUSTER": CLUSTER,
	"LOCAL":   LOCAL,
}

var kubeConfig = CLUSTER

// parses the environment variable KUBECONFIG and creates a client based on the
// retrieved configuration. Configuration is retrieved from
// either LOCAL: $HOME\.kube\config
// or CLUSTER: from the configuration added into every pod (environment variables..)
func init() {

	aliasMappingPath = os.Getenv("ALIAS_MAPPING_PATH")
	if aliasMappingPath == "" {
		aliasMappingPath= "/alias/mappings.yaml"
	}
	templatePath = os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "/tmp/index.md.tpl"
	}

	outputPath = os.Getenv("OUTPUT_PATH")
	if outputPath == "" {
		outputPath = "/data/index.md"
	}

	hostsPath = os.Getenv("HOSTS_PATH")
	if hostsPath == "" {
		hostsPath = "/etc/hosts"
	}

	defaultIngressHost = os.Getenv("DEFAULT_INGRESS_HOST")
	if defaultIngressHost == "" {
		defaultIngressHost = "istio"
	}

	if val, ok := kubeConfigByName[os.Getenv("KUBECONFIG")]; ok {
		kubeConfig = val
	}

	yamlFile, err := ioutil.ReadFile(aliasMappingPath)
	if err != nil { panic(err) }

	err = yaml.Unmarshal(yamlFile, &aliasMappings)
	if err != nil { panic(err) }

	var config *rest.Config

	switch kubeConfig {
	case LOCAL:
		// uses the current context in kubeconfig
		kubeconfig := flag.String("kubeconfig", os.Getenv("HOME")+string(os.PathSeparator)+".kube"+string(os.PathSeparator)+"config", "absolute path to the kubeconfig file")
		flag.Parse()
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	case CLUSTER:
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		panic(err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
}

func CreateAndRunIngressInformer() {
	ingressStore, ingressController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (runtime.Object, error) {
				return clientset.ExtensionsV1beta1().Ingresses("").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				lo.Watch = true
				return clientset.ExtensionsV1beta1().Ingresses("").Watch(lo)
			},
		},
		&v1beta1.Ingress{},
		300*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handleIngressAdd,
			UpdateFunc: handleIngressUpdate,
			DeleteFunc: handleIngressDelete,
		},
	)
	ingressController.Run(wait.NeverStop)
}

func CreateAndRunServiceInformer() {
	cleanHosts()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)
	signal.Notify(sigchan, syscall.SIGINT)
	go func() {
		<-sigchan

		cleanHosts()
		os.Exit(0)
	}()

	serviceStore, serviceController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Services("").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				lo.Watch = true
				return clientset.CoreV1().Services("").Watch(lo)
			},
		},
		&v1.Service{},
		300*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handleServiceAdd,
			UpdateFunc: handleServiceUpdate,
			DeleteFunc: handleServiceDelete,
		},
	)
	serviceController.Run(wait.NeverStop)
}

func handleServiceAdd(new interface{}) {
	if service, ok := new.(*v1.Service); ok {
		fmt.Printf("EVENT: service %s ADDED\n", service.Name)

		addHost(service)
		writeOutput()
	}
}

func handleServiceUpdate(_, new interface{}) {
	if service, ok := new.(*v1.Service); ok {
		fmt.Printf("EVENT: service %s UPDATED\n", service.Name)

		addHost(service)
		writeOutput()
	}
}

func handleServiceDelete(new interface{}) {
	if service, ok := new.(*v1.Service); ok {
		fmt.Printf("EVENT: service %s DELETED\n", service.Name)

		removeHost(service)
		writeOutput()
	}
}

func handleIngressAdd(new interface{}) {
	if service, ok := new.(*v1beta1.Ingress); ok {
		fmt.Printf("EVENT: ingress %s ADDED\n", service.Name)
		writeOutput()
	}
}

func handleIngressUpdate(_, new interface{}) {
	if service, ok := new.(*v1beta1.Ingress); ok {
		fmt.Printf("EVENT: ingress %s UPDATED\n", service.Name)
		writeOutput()
	}
}

func handleIngressDelete(new interface{}) {
	if service, ok := new.(*v1beta1.Ingress); ok {
		fmt.Printf("EVENT: ingress %s DELETED\n", service.Name)
		writeOutput()
	}
}

func newHosts() (goodhosts.Hosts, error) {
	hosts := goodhosts.Hosts{Path: hostsPath}

	err := hosts.Load()

	return hosts, err
}

func cleanHosts() {
	fmt.Printf("Cleaning hosts file: %s\n", hostsPath)
	hosts, err := newHosts()
	if err != nil {
		panic(err)
	}

	_, ipNet, err := net.ParseCIDR(clusterIPCIDR)
	if err != nil {
		panic(err)
	}

	for _, hostLine := range hosts.Lines {
		if ipNet.Contains(net.ParseIP(hostLine.IP)) && !hostLine.IsComment() {
			hosts.Remove(hostLine.IP, hostLine.Hosts...)
		}
	}
	if err := hosts.Flush(); err != nil {
		panic(err)
	}
}

func addHost(service *v1.Service) {
	hosts, err := newHosts()
	if err != nil {
		panic(err)
	}

	alias := service.Name + "." + service.Namespace
	hosts.Add(service.Spec.ClusterIP, alias)
	fmt.Printf("\t%s: Added %s %s\n", hostsPath, service.Spec.ClusterIP, alias)

	for _, aliasMapping := range aliasMappings.Mappings {
		if alias == aliasMapping.Source {
			for _, target := range aliasMapping.Targets {
				hosts.Add(service.Spec.ClusterIP, target)
			}
			fmt.Printf("\t%s: Added %s %s\n", hostsPath, service.Spec.ClusterIP, aliasMapping.Targets)
		}
	}

	if err := hosts.Flush(); err != nil {
		panic(err)
	}
}

func removeHost(service *v1.Service) {
	hosts, err := newHosts()
	if err != nil {
		panic(err)
	}

	hosts.Remove(service.Spec.ClusterIP, service.Name+"."+service.Namespace)

	if err := hosts.Flush(); err != nil {
		panic(err)
	}
}

type Data struct {
	Services  map[string][]*v1.Service
	Ingresses map[string][]*v1beta1.Ingress
	DefaultIngressHost string
}

func writeOutput() {
	data := Data{
		Services:  make(map[string][]*v1.Service),
		Ingresses: make(map[string][]*v1beta1.Ingress),
		DefaultIngressHost: defaultIngressHost,
	}

	for _, item := range serviceStore.List() {
		if svc, ok := item.(*v1.Service); ok {
			ns := svc.Namespace
			data.Services[ns] = append(data.Services[ns], svc)
		}
	}

	for _, item := range ingressStore.List() {
		if ingress, ok := item.(*v1beta1.Ingress); ok {
			ns := ingress.Namespace
			data.Ingresses[ns] = append(data.Ingresses[ns], ingress)
		}
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(outputFile, data)
	if err != nil {
		panic(err)
	}
}
