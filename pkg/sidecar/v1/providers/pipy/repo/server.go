package repo

import (
	"fmt"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	client2 "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/registry"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// ServerType is the type identifier for the ADS server
	ServerType = "pipy-Repo"

	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0

	fsmCodebaseConfig   = "config.json"
	fsmCodebaseConfigGz = "config.json.gz"
)

var (
	fsmCodebase        = "fsm-base"
	fsmSidecarCodebase = "fsm-sidecar"
	fsmCodebaseRepo    = fmt.Sprintf("/%s", fsmCodebase)
)

// NewRepoServer creates a new Aggregated Discovery Service server
func NewRepoServer(meshCatalog catalog.MeshCataloger, proxyRegistry *registry.ProxyRegistry, fsmNamespace string, cfg configurator.Configurator, certManager *certificate.Manager, kubecontroller k8s.Controller, msgBroker *messaging.Broker) *Server {
	if len(cfg.GetRepoServerCodebase()) > 0 {
		fsmCodebase = fmt.Sprintf("%s/%s", cfg.GetRepoServerCodebase(), fsmCodebase)
		fsmSidecarCodebase = fmt.Sprintf("%s/%s", cfg.GetRepoServerCodebase(), fsmSidecarCodebase)
		fsmCodebaseRepo = fmt.Sprintf("/%s", fsmCodebase)
	}

	server := Server{
		catalog:        meshCatalog,
		proxyRegistry:  proxyRegistry,
		fsmNamespace:   fsmNamespace,
		cfg:            cfg,
		certManager:    certManager,
		workQueues:     workerpool.NewWorkerPool(workerPoolSize),
		kubeController: kubecontroller,
		configVerMutex: sync.Mutex{},
		configVersion:  make(map[string]uint64),
		pluginSet:      mapset.NewSet(),
		msgBroker:      msgBroker,
		repoClient:     client2.NewRepoClient(cfg.GetRepoServerIPAddr(), uint16(cfg.GetProxyServerPort())),
	}

	prettyConfig = func() bool {
		return cfg.GetMeshConfig().Spec.FeatureFlags.EnableSidecarPrettyConfig
	}

	return &server
}

// Start starts the codebase push server
func (s *Server) Start(_ uint32, _ *certificate.Certificate) error {
	// wait until pipy repo is up
	err := wait.PollImmediate(10*time.Second, 300*time.Second, func() (bool, error) {
		success, err := s.repoClient.IsRepoUp()
		if success {
			log.Info().Msg("Repo is READY!")
			return success, nil
		}
		log.Error().Msg("Repo is not up, sleeping ...")
		return success, err
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	s.repoClient.Restore = func() error {
		_, restoreErr := s.repoClient.Batch(fmt.Sprintf("%d", 0), []client2.Batch{
			{
				Basepath: fsmCodebase,
				Items:    fsmCodebaseItems,
			},
		})
		if restoreErr != nil {
			log.Error().Err(restoreErr)
			return restoreErr
		}
		// wait until base codebase is ready
		restoreErr = wait.PollImmediate(5*time.Second, 90*time.Second, func() (bool, error) {
			success, _, codebaseErr := s.repoClient.GetCodebase(fsmCodebase)
			if success {
				log.Info().Msg("Base codebase is READY!")
				return success, nil
			}
			log.Error().Msg("Base codebase is NOT READY, sleeping ...")
			return success, codebaseErr
		})
		if restoreErr != nil {
			log.Error().Err(restoreErr)
			return restoreErr
		}
		return nil
	}

	err = s.repoClient.Restore()
	if err != nil {
		log.Error().Err(err)
		return err
	}

	// Start broadcast listener thread
	go s.broadcastListener()

	s.ready = true

	return nil
}
