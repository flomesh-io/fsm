package v2

import (
	"context"
	"net"
	"net/http"
	"path"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/configurator"
	xnetworkClientset "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
	"github.com/flomesh-io/fsm/pkg/xnetwork"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/util"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/volume"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

// NewXNetConfigServer creates a new xnetwork config Service server
func NewXNetConfigServer(ctx context.Context,
	cfg configurator.Configurator,
	xnetworkController xnetwork.Controller,
	KubeClient kubernetes.Interface,
	kubeController k8s.Controller,
	xnetworkClient xnetworkClientset.Interface,
	msgBroker *messaging.Broker,
	nodeName, cniBridge4, cniBridge6 string) *Server {
	server := &Server{
		nodeName:           nodeName,
		ctx:                ctx,
		cfg:                cfg,
		xnetworkController: xnetworkController,
		kubeClient:         KubeClient,
		kubeController:     kubeController,
		xnetworkClient:     xnetworkClient,
		msgBroker:          msgBroker,
		workQueues:         workerpool.NewWorkerPool(workerPoolSize),
		cniBridge4:         cniBridge4,
		cniBridge6:         cniBridge6,
		xnatCache:          make(map[string]*XNat),
	}
	kubeController.AddObserveFilter(server.xNetDnsProxyUpstreamsObserveFilter)
	return server
}

func (s *Server) Start() error {
	s.waitXnetReady()

	retries := 0
	for {
		retries++
		if retries > 12 {
			log.Fatal().Msg(`timeout waiting for xnet to be ready`)
		}
		if err := s.loadNatEntries(); err != nil {
			if retries > 8 {
				log.Error().Err(err).Msg(`waiting for xnet to be ready ...`)
			} else if retries > 4 {
				log.Warn().Err(err).Msg(`waiting for xnet to be ready ...`)
			} else {
				log.Debug().Err(err).Msg(`waiting for xnet to be ready ...`)
			}
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	s.ready = true
	return nil
}

func (s *Server) waitXnetReady() {
	unixSock := path.Join(volume.SysRun.MountPath, `.xnet.sock`)
	for {
		if util.Exists(unixSock) {
			httpClient := http.Client{
				Transport: &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", unixSock)
					},
				},
			}
			if r, err := httpClient.Get("http://xcni/version"); err == nil {
				if r.StatusCode == http.StatusOK {
					break
				}
			}
		}
		time.Sleep(time.Second * 2)
	}
}
