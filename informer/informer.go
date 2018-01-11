package informer

import (
	"text/template"
	"os/signal"
	"net"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lextoumbourou/goodhosts"

	v1 "k8s.io/api/core/v1"

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
var clusterIPCIDR = "10.96.0.0/12"
var templatePath string
var outputPath string
var hostsPath string

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
	
	templatePath = os.Getenv("TEMPLATE_PATH")
	if templatePath == "" { templatePath = "/tmp/index.md.tpl"}

	outputPath = os.Getenv("OUTPUT_PATH")
	if outputPath == "" { outputPath = "/data/index.md"}

	hostsPath = os.Getenv("HOSTS_PATH")
	if hostsPath == "" { templatePath = "/etc/hosts"}

	if val, ok := kubeConfigByName[os.Getenv("KUBECONFIG")]; ok {
		kubeConfig = val
	}

	var err error
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

// Creates and starts an informer which watches services
// and stores them in the serviceStore
// whenever services are added or updated the respective functions are called
func CreateAndRunServiceInformer() {
	cleanHosts()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	go func(){
		<- sigchan 

		fmt.Println("Cleaning up hosts file")
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
		writeServices()
	}
}

func handleServiceUpdate(_, new interface{}) {
	if service, ok := new.(*v1.Service); ok {
		fmt.Printf("EVENT: service %s UPDATED\n", service.Name)

		addHost(service)
		writeServices()
	}
}

func handleServiceDelete(new interface{}) {
	if service, ok := new.(*v1.Service); ok {
		fmt.Printf("EVENT: service %s ADDED\n", service.Name)

		removeHost(service)
		writeServices()
	}
}

func newHosts() (goodhosts.Hosts, error) {
	hosts := goodhosts.Hosts{Path: hostsPath}

	err := hosts.Load()

	return hosts, err
}

func cleanHosts(){
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

	hosts.Add(service.Spec.ClusterIP, service.Name+"."+service.Namespace)

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

type Services struct {
	Services map[string][]*v1.Service
}

func writeServices(){
	svcs := Services{
		Services: make(map[string][]*v1.Service),
	}

	for _, item := range serviceStore.List(){
		if svc, ok := item.(*v1.Service); ok {
			ns := svc.Namespace
			svcs.Services[ns] = append(svcs.Services[ns], svc)
		}
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil { panic(err) }
	
	outputFile, err := os.Create(outputPath)
	if err != nil { panic(err) }

	err = tmpl.Execute(outputFile, svcs)
	if err != nil { panic(err) }
}