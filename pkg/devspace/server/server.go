package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	yaml "gopkg.in/yaml.v2"
)

// Server is listens on a given port for the ui functionality
type Server struct {
	server *http.Server
}

// NewServer creates a new server from the given parameters
func NewServer(client *kubectl.Client, config *latest.Config, generatedConfig *generated.Config, port int, log log.Logger) (*Server, error) {
	return &Server{
		server: &http.Server{
			Addr:    "localhost:" + strconv.Itoa(port),
			Handler: newHandler(client, config, generatedConfig, log),
			// ReadTimeout:  5 * time.Second,
			// WriteTimeout: 10 * time.Second,
			// IdleTimeout:  60 * time.Second,
		},
	}, nil
}

// ListenAndServe implements interface
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

type handler struct {
	config          *latest.Config
	generatedConfig *generated.Config
	client          *kubectl.Client
	log             log.Logger
	mux             *http.ServeMux
}

func newHandler(client *kubectl.Client, config *latest.Config, generatedConfig *generated.Config, log log.Logger) *handler {
	handler := &handler{
		mux:             http.NewServeMux(),
		client:          client,
		config:          config,
		log:             log,
		generatedConfig: generatedConfig,
	}

	handler.mux.HandleFunc("/api/resource", handler.request)
	handler.mux.HandleFunc("/api/config", handler.returnConfig)
	handler.mux.HandleFunc("/api/logs", handler.logs)
	return handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.URL != nil {
		h.log.Infof("Incoming request at %s", r.URL.String())
	}
	h.mux.ServeHTTP(w, r)
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

func (h *handler) returnConfig(w http.ResponseWriter, r *http.Request) {
	s, err := yaml.Marshal(h.config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data interface{}
	if err := yaml.Unmarshal([]byte(s), &data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data = convert(data)

	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (h *handler) request(w http.ResponseWriter, r *http.Request) {
	resource, ok := r.URL.Query()["resource"]
	if !ok || len(resource) != 1 {
		http.Error(w, "resource is missing", http.StatusBadRequest)
		return
	}

	// Build request options
	options := &kubectl.GenericRequestOptions{Resource: resource[0]}

	// Namespace
	namespace, ok := r.URL.Query()["namespace"]
	if ok && len(namespace) == 1 {
		options.Namespace = namespace[0]
	}

	// Api version
	apiVersion, ok := r.URL.Query()["apiVersion"]
	if ok && len(apiVersion) == 1 {
		options.APIVersion = apiVersion[0]
	}

	// Name
	name, ok := r.URL.Query()["name"]
	if ok && len(name) == 1 {
		options.Name = name[0]
	}

	// LabelSelector
	labelSelector, ok := r.URL.Query()["labelSelector"]
	if ok && len(name) == 1 {
		options.LabelSelector = labelSelector[0]
	}

	// Do the request
	out, err := kubectl.GenericRequest(h.client, options)
	if err != nil {
		h.log.Errorf("Error in /api/resource: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(out))
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}
