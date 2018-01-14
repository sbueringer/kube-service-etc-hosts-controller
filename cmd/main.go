package main

import (
	"github.com/sbueringer/kube-service-etc-hosts-operator/informer"
)

// main starts the informer in the background and waits
// forever to keep the program running
func main() {

	go informer.CreateAndRunIngressInformer()
	go informer.CreateAndRunServiceInformer()

	// Wait forever
	select {}
}
