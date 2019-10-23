package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/port"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Server is listens on a given port for the ui functionality
type Server struct {
	Server *http.Server
	log    log.Logger
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

// DefaultPort is the default port the ui server will listen to
const DefaultPort = 8090

// NewServer creates a new server from the given parameters
func NewServer(client *kubectl.Client, config *latest.Config, generatedConfig *generated.Config, ignoreDownloadError bool, log log.Logger) (*Server, error) {
	path, err := downloadUI()
	if err != nil {
		if !ignoreDownloadError {
			return nil, errors.Wrap(err, "download ui")
		}

		log.Warnf("Couldn't download ui: %v", err)
	}

	// Find an open port
	usePort := DefaultPort
	for {
		unused, _ := port.Check(usePort)
		if unused {
			break
		}

		usePort++
	}

	return &Server{
		Server: &http.Server{
			Addr:    "localhost:" + strconv.Itoa(usePort),
			Handler: newHandler(client, config, generatedConfig, path, log),
			// ReadTimeout:  5 * time.Second,
			// WriteTimeout: 10 * time.Second,
			// IdleTimeout:  60 * time.Second,
		},
		log: log,
	}, nil
}

// ListenAndServe implements interface
func (s *Server) ListenAndServe() error {
	s.log.Infof("Start listening on %s", s.Server.Addr)

	return s.Server.ListenAndServe()
}

type handler struct {
	config          *latest.Config
	path            string
	generatedConfig *generated.Config
	client          *kubectl.Client
	log             log.Logger
	mux             *http.ServeMux
}

func newHandler(client *kubectl.Client, config *latest.Config, generatedConfig *generated.Config, path string, log log.Logger) *handler {
	handler := &handler{
		mux:             http.NewServeMux(),
		path:            path,
		client:          client,
		config:          config,
		log:             log,
		generatedConfig: generatedConfig,
	}

	handler.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(path, "index.html"))
	})
	handler.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(path, "static")))))
	handler.mux.HandleFunc("/api/resource", handler.request)
	handler.mux.HandleFunc("/api/config", handler.returnConfig)
	handler.mux.HandleFunc("/api/enter", handler.enter)
	handler.mux.HandleFunc("/api/logs", handler.logs)
	handler.mux.HandleFunc("/api/logs-multiple", handler.logsMultiple)
	return handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// if r.URL != nil {
	//	h.log.Infof("Incoming request at %s", r.URL.String())
	// }
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

type returnConfig struct {
	Config          *latest.Config    `yaml:"config"`
	GeneratedConfig *generated.Config `yaml:"generatedConfig"`

	Profile       string `yaml:"profile"`
	KubeContext   string `yaml:"kubeContext"`
	KubeNamespace string `yaml:"kubeNamespace"`
}

func (h *handler) returnConfig(w http.ResponseWriter, r *http.Request) {
	s, err := yaml.Marshal(&returnConfig{
		Config:          h.config,
		GeneratedConfig: h.generatedConfig,
		Profile:         h.generatedConfig.GetActiveProfile(),
		KubeContext:     h.client.CurrentContext,
		KubeNamespace:   h.client.Namespace,
	})
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
