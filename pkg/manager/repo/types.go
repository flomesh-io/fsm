package repo

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// Rebuilder is the interface for the rebuilder
type Rebuilder interface {
	manager.Runnable
	manager.LeaderElectionRunnable
}

type rebuilder struct {
	repoClient *repo.PipyRepoClient
	client     client.Client
	mc         configurator.Configurator
	scheduler  *gocron.Scheduler
}

func NewRebuilder(repoClient *repo.PipyRepoClient, client client.Client, mc configurator.Configurator) Rebuilder {
	s := gocron.NewScheduler(time.Local)
	s.SingletonModeAll()
	s.RegisterEventListeners(
		gocron.AfterJobRuns(func(jobName string) {
			log.Debug().Msgf(">>>>>> After ChecksAndRebuildRepo: %s\n", jobName)
		}),
		gocron.BeforeJobRuns(func(jobName string) {
			log.Debug().Msgf(">>>>>> Before ChecksAndRebuildRepo: %s\n", jobName)
		}),
		gocron.WhenJobReturnsError(func(jobName string, err error) {
			log.Error().Msgf(">>>>>> ChecksAndRebuildRepo Returns Error: %s, %v\n", jobName, err)
		}),
	)

	r := &rebuilder{
		repoClient: repoClient,
		client:     client,
		mc:         mc,
		scheduler:  s,
	}

	return r
}

func (r *rebuilder) Start(_ context.Context) error {
	if _, err := r.scheduler.Every(60).Seconds().
		Name("rebuild-repo").
		Do(r.rebuildRepoJob); err != nil {
		log.Error().Msgf("Error happened while rebuilding repo: %s", err)
	}

	r.scheduler.StartAsync()

	return nil
}

func (r *rebuilder) NeedLeaderElection() bool {
	return false
}
