package resolver

import (
	"github.com/aporeto-inc/kubernetes-integration/kubernetes"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/watch"
)

// NamespaceWatcher implements the policy for a specific Namespace
type NamespaceWatcher struct {
	namespace         string
	podResultChan     chan watch.Event
	policyResultChan  chan watch.Event
	namespaceStopChan chan bool
	podStopChan       chan bool
	policyStopChan    chan bool
}

// NewNamespaceWatcher initialize a new NamespaceWatcher that watches the Pod and
// Networkpolicy events on the specific namespace passed in parameter.
func NewNamespaceWatcher(client *kubernetes.Client, namespace string) *NamespaceWatcher {
	// Creating all the channels for the Subwatchers.
	namespaceWatcher := &NamespaceWatcher{
		namespace:         namespace,
		podResultChan:     make(chan watch.Event),
		policyResultChan:  make(chan watch.Event),
		namespaceStopChan: make(chan bool),
		podStopChan:       make(chan bool),
		policyStopChan:    make(chan bool),
	}

	//Launching the Pod and Policy watchers:
	go client.LocalPodWatcher(namespace, namespaceWatcher.podResultChan, namespaceWatcher.podStopChan)
	go client.PolicyWatcher(namespace, namespaceWatcher.policyResultChan, namespaceWatcher.policyStopChan)

	return namespaceWatcher
}

func (n *NamespaceWatcher) stopWatchingNamespace() {
	n.podStopChan <- true
	n.policyStopChan <- true
	n.namespaceStopChan <- true
}

func (n *NamespaceWatcher) startWatchingNamespace(
	podEventHandler func(*api.Pod, watch.EventType) error,
	networkPolicyEventHandler func(*extensions.NetworkPolicy, watch.EventType) error,
) {
	for {
		select {
		case <-n.namespaceStopChan:
			glog.V(2).Infof("Received Stop signal for Namespace: %s", n.namespace)
			return
		case req := <-n.podResultChan:
			glog.V(2).Infof("Processing PodEvent for Namespace: %s", n.namespace)
			podEventHandler(req.Object.(*api.Pod), req.Type)
		case req := <-n.policyResultChan:
			glog.V(2).Infof("Processing PolicyEvent for Namespace: %s", n.namespace)
			networkPolicyEventHandler(req.Object.(*extensions.NetworkPolicy), req.Type)
		}

	}
}