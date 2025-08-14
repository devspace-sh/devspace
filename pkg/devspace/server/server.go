package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Server is listens on a given port for the ui functionality
type Server struct {
	Server *http.Server
}

// DefaultPort is the default port the ui server will listen to
const DefaultPort = 8090

// NewServer creates a new server from the given parameters
func NewServer(ctx devspacecontext.Context, host string, ignoreDownloadError bool, forcePort *int, pipeline types.Pipeline) (*Server, error) {
	path, err := downloadUI()
	if err != nil {
		if !ignoreDownloadError {
			return nil, errors.Wrap(err, "download ui")
		}

		ctx.Log().Warnf("Couldn't download ui: %v", err)
	}

	// Find an open port
	usePort := DefaultPort
	if forcePort != nil {
		usePort = *forcePort

		if host == "localhost" {
			available, err := port.IsAvailable(fmt.Sprintf(":%d", usePort))
			if !available {
				return nil, errors.Errorf("Port %d already in use: %v", usePort, err)
			}
		}
	} else {
		if host == "localhost" {
			for i := 0; i < 20; i++ {
				available, _ := port.IsAvailable(fmt.Sprintf(":%d", usePort))
				if available {
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
	handler, err := newHandler(ctx, path, pipeline)
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
	}, nil
}

// ListenAndServe implements interface
func (s *Server) ListenAndServe() error {
	return s.Server.ListenAndServe()
}

type handler struct {
	ctx      devspacecontext.Context
	pipeline types.Pipeline

	kubeContexts     map[string]string
	analyticsEnabled bool
	path             string
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

func newHandler(ctx devspacecontext.Context, path string, pipeline types.Pipeline) (*handler, error) { // Get kube config
	kubeConfig, err := kubeconfig.NewLoader().LoadRawConfig()
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

	handler := &handler{
		ctx:                  ctx,
		pipeline:             pipeline,
		mux:                  http.NewServeMux(),
		path:                 path,
		kubeContexts:         kubeContexts,
		ports:                make(map[string]*forward),
		clientCache:          make(map[string]kubectl.Client),
		terminalResizeQueues: make(map[string]TerminalResizeQueue),
	}
	handler.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(path, "index.html"))
	})
	handler.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(path, "static")))))
	handler.mux.HandleFunc("/api/ping", handler.ping)
	handler.mux.HandleFunc("/api/exclude-dependency", handler.excludeDependency)
	handler.mux.HandleFunc("/api/version", handler.version)
	handler.mux.HandleFunc("/api/command", handler.command)
	handler.mux.HandleFunc("/api/resource", handler.request)
	handler.mux.HandleFunc("/api/config", handler.returnConfig)
	handler.mux.HandleFunc("/api/forward", handler.forward)
	handler.mux.HandleFunc("/api/enter", handler.enter)
	handler.mux.HandleFunc("/api/resize", handler.resize)
	handler.mux.HandleFunc("/api/logs", handler.logs)
	return handler, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/*w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		return
	}*/

	if r.Method != "GET" && r.Method != "POST" {
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
	_, _ = w.Write(b)
}

type returnConfig struct {
	Config     *latest.Config         `yaml:"config"`
	RawConfig  map[string]interface{} `yaml:"rawConfig"`
	LocalCache localCache             `yaml:"generatedConfig"`

	AnalyticsEnabled bool              `yaml:"analyticsEnabled"`
	Profile          string            `yaml:"profile"`
	WorkingDirectory string            `yaml:"workingDirectory"`
	KubeContext      string            `yaml:"kubeContext"`
	KubeNamespace    string            `yaml:"kubeNamespace"`
	KubeContexts     map[string]string `yaml:"kubeContexts"`
}

type localCache struct {
	Vars        map[string]interface{}        `yaml:"vars,omitempty"`
	LastContext *localcache.LastContextConfig `yaml:"lastContext,omitempty"`
}

func (h *handler) returnConfig(w http.ResponseWriter, r *http.Request) {
	profile := ""
	retConfig := &returnConfig{
		AnalyticsEnabled: h.analyticsEnabled,
		Profile:          profile,
		WorkingDirectory: h.ctx.WorkingDir(),
		KubeContexts:     h.kubeContexts,
	}
	if h.ctx.Config() != nil {
		retConfig.RawConfig = h.ctx.Config().Raw()
		retConfig.Config = h.ctx.Config().Config()
		retConfig.LocalCache = localCache{
			Vars:        h.ctx.Config().Variables(),
			LastContext: h.ctx.Config().LocalCache().GetLastContext(),
		}
	}
	if h.ctx.KubeClient() != nil {
		retConfig.KubeNamespace = h.ctx.KubeClient().Namespace()
		retConfig.KubeContext = h.ctx.KubeClient().CurrentContext()
	}

	s, err := yaml.Marshal(retConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data interface{}
	if err := yamlutil.Unmarshal([]byte(s), &data); err != nil {
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
	_, _ = w.Write(b)
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
	kubeContext := h.ctx.KubeClient().CurrentContext()
	context, ok := r.URL.Query()["context"]
	if ok && len(context) == 1 && context[0] != "" {
		kubeContext = context[0]
	}

	// Namespace
	kubeNamespace := h.ctx.KubeClient().Namespace()
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
		h.ctx.Log().Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Do the request
	out, err := client.GenericRequest(r.Context(), options)
	if err != nil {
		if strings.Index(err.Error(), "request: unknown") != 0 {
			h.ctx.Log().Errorf("Error in %s: %v", r.URL.String(), err)
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(out))
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
