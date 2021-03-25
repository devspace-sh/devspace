package server

import (
	"encoding/json"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/port"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Server is listens on a given port for the ui functionality
type Server struct {
	Server *http.Server
	log    log.Logger
}

// DefaultPort is the default port the ui server will listen to
const DefaultPort = 8090

// NewServer creates a new server from the given parameters
func NewServer(config config.Config, host string, ignoreDownloadError bool, defaultContext, defaultNamespace string, forcePort *int, log log.Logger) (*Server, error) {
	path, err := downloadUI()
	if err != nil {
		if !ignoreDownloadError {
			return nil, errors.Wrap(err, "download ui")
		}

		log.Warnf("Couldn't download ui: %v", err)
	}

	// Find an open port
	usePort := DefaultPort
	if forcePort != nil {
		usePort = *forcePort

		if host == "localhost" {
			unused, err := port.CheckHostPort(host, usePort)
			if unused == false {
				return nil, errors.Errorf("Port %d already in use: %v", usePort, err)
			}
		}
	} else {
		if host == "localhost" {
			for i := 0; i < 20; i++ {
				unused, err := port.CheckHostPort(host, usePort)
				if unused {
					break
				}

				usePort++
				if i+1 == 20 {
					return nil, err
				}
			}
		}
	}

	// Create handler
	handler, err := newHandler(config, defaultContext, defaultNamespace, path, log)
	if err != nil {
		return nil, err
	}

	return &Server{
		Server: &http.Server{
			Addr:    host + ":" + strconv.Itoa(usePort),
			Handler: handler,
			// ReadTimeout:  5 * time.Second,
			// WriteTimeout: 10 * time.Second,
			// IdleTimeout:  60 * time.Second,
		},
		log: log,
	}, nil
}

// ListenAndServe implements interface
func (s *Server) ListenAndServe() error {
	return s.Server.ListenAndServe()
}

type handler struct {
	config           *latest.Config
	generatedConfig  *generated.Config
	defaultContext   string
	defaultNamespace string
	rawConfig        map[interface{}]interface{}
	kubeContexts     map[string]string
	workingDirectory string
	analyticsEnabled bool
	path             string
	log              log.Logger
	mux              *http.ServeMux

	clientCache      map[string]kubectl.Client
	clientCacheMutex sync.Mutex

	terminalResizeQueues      map[string]TerminalResizeQueue
	terminalResizeQueuesMutex sync.Mutex

	ports      map[string]*forward
	portsMutex sync.Mutex
}

type forward struct {
	portForwarder     *portforward.PortForwarder
	portForwarderStop chan struct{}
	portForwarderPort int

	podUUID string
}

func newHandler(config config.Config, defaultContext, defaultNamespace, path string, log log.Logger) (*handler, error) { // Get kube config
	kubeConfig, err := kubeconfig.NewLoader().LoadConfig().RawConfig()
	if err != nil {
		return nil, errors.Wrap(err, "load kube config")
	}

	kubeContexts := map[string]string{}
	for name, context := range kubeConfig.Contexts {
		namespace := context.Namespace
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}

		kubeContexts[name] = namespace
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get working directory")
	}

	handler := &handler{
		mux:                  http.NewServeMux(),
		path:                 path,
		defaultContext:       defaultContext,
		defaultNamespace:     defaultNamespace,
		kubeContexts:         kubeContexts,
		workingDirectory:     cwd,
		log:                  log,
		generatedConfig:      config.Generated(),
		ports:                make(map[string]*forward),
		clientCache:          make(map[string]kubectl.Client),
		terminalResizeQueues: make(map[string]TerminalResizeQueue),
	}

	// Load raw config
	if config != nil {
		handler.rawConfig = config.Raw()
		handler.config = config.Config()
	}

	handler.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(path, "index.html"))
	})
	handler.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(path, "static")))))
	handler.mux.HandleFunc("/api/version", handler.version)
	handler.mux.HandleFunc("/api/command", handler.command)
	handler.mux.HandleFunc("/api/resource", handler.request)
	handler.mux.HandleFunc("/api/config", handler.returnConfig)
	handler.mux.HandleFunc("/api/forward", handler.forward)
	handler.mux.HandleFunc("/api/enter", handler.enter)
	handler.mux.HandleFunc("/api/resize", handler.resize)
	handler.mux.HandleFunc("/api/logs", handler.logs)
	handler.mux.HandleFunc("/api/logs-multiple", handler.logsMultiple)
	return handler, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/*w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		return
	}*/

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.mux.ServeHTTP(w, r)
}

// UIServerVersion is the struct that is returned by the /api/version request
type UIServerVersion struct {
	Version  string `json:"version"`
	DevSpace bool   `json:"devSpace"`
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	version := upgrade.GetVersion()
	b, err := json.Marshal(&UIServerVersion{
		Version:  version,
		DevSpace: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

type returnConfig struct {
	Config          *latest.Config              `yaml:"config"`
	RawConfig       map[interface{}]interface{} `yaml:"rawConfig"`
	GeneratedConfig *generated.Config           `yaml:"generatedConfig"`

	AnalyticsEnabled bool              `yaml:"analyticsEnabled"`
	Profile          string            `yaml:"profile"`
	WorkingDirectory string            `yaml:"workingDirectory"`
	KubeContext      string            `yaml:"kubeContext"`
	KubeNamespace    string            `yaml:"kubeNamespace"`
	KubeContexts     map[string]string `yaml:"kubeContexts"`
}

func (h *handler) returnConfig(w http.ResponseWriter, r *http.Request) {
	profile := ""
	if h.generatedConfig != nil {
		profile = h.generatedConfig.GetActiveProfile()
	}

	s, err := yaml.Marshal(&returnConfig{
		AnalyticsEnabled: h.analyticsEnabled,
		Config:           h.config,
		GeneratedConfig:  h.generatedConfig,
		Profile:          profile,
		RawConfig:        h.rawConfig,
		WorkingDirectory: h.workingDirectory,
		KubeContexts:     h.kubeContexts,
		KubeContext:      h.defaultContext,
		KubeNamespace:    h.defaultNamespace,
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

	data = yamlutil.Convert(data)

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

	// check client cache
	client, err := h.getClientFromCache(kubeContext, kubeNamespace)
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Do the request
	out, err := client.GenericRequest(options)
	if err != nil {
		if strings.Index(err.Error(), "request: unknown") != 0 {
			h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(out))
}

func (h *handler) getClientFromCache(kubeContext, kubeNamespace string) (kubectl.Client, error) {
	key := kubeNamespace + ":" + kubeContext

	h.clientCacheMutex.Lock()
	defer h.clientCacheMutex.Unlock()

	var err error
	client, ok := h.clientCache[key]
	if !ok {
		client, err = kubectl.NewClientFromContext(kubeContext, kubeNamespace, false, kubeconfig.NewLoader())
		if err != nil {
			return nil, err
		}

		h.clientCache[key] = client
	}

	return client, nil
}
