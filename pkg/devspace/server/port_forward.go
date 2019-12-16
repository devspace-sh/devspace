package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/port"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const minPort = 2048
const maxPort = 40000

func (h *handler) forward(w http.ResponseWriter, r *http.Request) {
	// Kube Context
	kubeContext := h.defaultContext
	context, ok := r.URL.Query()["context"]
	if ok && len(context) == 1 && context[0] != "" {
		kubeContext = context[0]
	}

	// Namespace
	kubeNamespace := h.defaultNamespace
	namespace, ok := r.URL.Query()["namespace"]
	if ok && len(namespace) == 1 && namespace[0] != "" {
		kubeNamespace = namespace[0]
	}

	name, ok := r.URL.Query()["name"]
	if !ok || len(name) != 1 {
		http.Error(w, "name is missing", http.StatusBadRequest)
		return
	}

	targetPort, ok := r.URL.Query()["port"]
	if !ok || len(targetPort) != 1 {
		http.Error(w, "port is missing", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s/%s:%s", kubeContext, kubeNamespace, name[0], targetPort[0])

	// Check if exists
	h.portsMutex.Lock()
	defer h.portsMutex.Unlock()

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(kubeContext, kubeNamespace, false)
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pod, err := client.KubeClient().CoreV1().Pods(kubeNamespace).Get(name[0], metav1.GetOptions{})
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.ports[key] != nil {
		// Check if the pod is the same
		if h.ports[key].podUUID == string(pod.UID) {
			w.Write([]byte(strconv.Itoa(h.ports[key].portForwarderPort)))
			return
		}

		close(h.ports[key].portForwarderStop)
		delete(h.ports, key)
	}

	// Find open port
	checkPort := rand.Intn(maxPort-minPort) + minPort
	for {
		unused, _ := port.Check(checkPort)
		if unused {
			break
		}

		checkPort = rand.Intn(maxPort-minPort) + minPort
	}

	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	errorChan := make(chan error)
	ports := []string{strconv.Itoa(checkPort) + ":" + targetPort[0]}

	pf, err := client.NewPortForwarder(pod, ports, []string{"127.0.0.1"}, stopChan, readyChan, nil)

	go func(key string, port int) {
		defer h.log.Infof("Stop listening on on %d", port)
		err := pf.ForwardPorts()
		if err != nil {
			h.log.Warnf("Error forwarding ports: %v", err)
		}

		h.portsMutex.Lock()
		defer h.portsMutex.Unlock()

		delete(h.ports, key)
	}(key, checkPort)

	go func(key string) {
		err := <-errorChan
		if err != nil {
			h.portsMutex.Lock()
			delete(h.ports, key)
			h.portsMutex.Unlock()
		}

		pf.Close()
	}(key)

	// Wait till forwarding is ready
	select {
	case <-readyChan:
		h.log.Infof("Port forwarding started on %s", strings.Join(ports, ","))
		h.ports[key] = &forward{
			portForwarder:     pf,
			portForwarderPort: checkPort,
			portForwarderStop: stopChan,
			podUUID:           string(pod.UID),
		}

		w.Write([]byte(strconv.Itoa(h.ports[key].portForwarderPort)))
		return
	case <-time.After(10 * time.Second):
		h.log.Errorf("Error in %s: %v", r.URL.String(), "Timeout waiting for port forwarding to start")
		http.Error(w, "Timeout waiting for port forwarding to start", http.StatusInternalServerError)
		return
	}
}
