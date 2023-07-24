package controller

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	conn "github.com/flomesh-io/fsm/pkg/mcs/connector"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/rs/zerolog/log"
	metautil "k8s.io/apimachinery/pkg/api/meta"
)

func (s *ControlPlaneServer) Run(stop <-chan struct{}) {
	mcsPubSub := s.msgBroker.GetMCSEventPubSub()
	svcExportCreatedCh := mcsPubSub.Sub(announcements.MultiClusterServiceExportCreated.String())
	defer s.msgBroker.Unsub(mcsPubSub, svcExportCreatedCh)

	for {
		select {
		case msg, ok := <-svcExportCreatedCh:
			mc := s.cfg
			// ONLY Control Plane takes care of the federation of service export/import
			if mc.IsManaged() && mc.GetMultiClusterControlPlaneUID() != "" && mc.GetClusterUID() != mc.GetMultiClusterControlPlaneUID() {
				log.Info().Msgf("Ignore processing ServiceExportCreated event due to cluster is managed and not a control plane ...")
				continue
			}

			if !ok {
				log.Warn().Msgf("Channel closed for ServiceExport")
				continue
			}
			log.Info().Msgf("Received event ServiceExportCreated %v", msg)

			e, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Received unexpected message %T on channel, expected Message", e)
				continue
			}

			svcExportEvt, ok := e.NewObj.(*mcsevent.ServiceExportEvent)
			if !ok {
				log.Error().Msgf("Received unexpected object %T, expected *event.ServiceExportEvent", svcExportEvt)
				continue
			}

			// check ServiceExport Status, Invalid and Conflict ServiceExport is ignored
			export := svcExportEvt.ServiceExport
			if metautil.IsStatusConditionFalse(export.Status.Conditions, string(mcsv1alpha1.ServiceExportValid)) {
				log.Warn().Msgf("ServiceExport %v is ignored due to Valid status is false", export)
				continue
			}
			if metautil.IsStatusConditionTrue(export.Status.Conditions, string(mcsv1alpha1.ServiceExportConflict)) {
				log.Warn().Msgf("ServiceExport %v is ignored due to Conflict status is true", export)
				continue
			}

			s.processServiceExportCreatedEvent(svcExportEvt)
		case <-stop:
			log.Warn().Msgf("Received stop signal.")
			return
		}
	}
}

func (s *ControlPlaneServer) processServiceExportCreatedEvent(svcExportEvt *mcsevent.ServiceExportEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	export := svcExportEvt.ServiceExport
	if s.isFirstTimeExport(svcExportEvt) {
		log.Info().Msgf("[%s] ServiceExport %s/%s is exported first in the cluster set, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
		s.acceptServiceExport(svcExportEvt)
	} else {
		valid, err := s.isValidServiceExport(svcExportEvt)
		if valid {
			log.Info().Msgf("[%s] ServiceExport %s/%s is valid, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
			s.acceptServiceExport(svcExportEvt)
		} else {
			log.Info().Msgf("[%s] ServiceExport %s/%s is invalid, will be rejected", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
			s.rejectServiceExport(svcExportEvt, err)
		}
	}
}

func (s *ControlPlaneServer) isFirstTimeExport(event *mcsevent.ServiceExportEvent) bool {
	export := event.ServiceExport
	for _, bg := range s.backgrounds {
		if bg.Connector.ServiceImportExists(export) {
			log.Warn().Msgf("[%s] ServiceExport %s/%s exists in Cluster %s", event.Geo.Key(), export.Namespace, export.Name, bg.Context.ClusterKey)
			return false
		}
	}

	return true
}

func (s *ControlPlaneServer) isValidServiceExport(svcExportEvt *mcsevent.ServiceExportEvent) (bool, error) {
	export := svcExportEvt.ServiceExport
	for _, bg := range s.backgrounds {
		connectorContext := bg.Context
		if connectorContext.ClusterKey == svcExportEvt.ClusterKey() {
			// no need to test against itself
			continue
		}

		if err := bg.Connector.ValidateServiceExport(svcExportEvt.ServiceExport, svcExportEvt.Service); err != nil {
			log.Warn().Msgf("[%s] ServiceExport %s/%s has conflict in Cluster %s", svcExportEvt.Geo.Key(), export.Namespace, export.Name, connectorContext.ClusterKey)
			return false, err
		}
	}

	return true, nil
}

func (s *ControlPlaneServer) acceptServiceExport(svcExportEvt *mcsevent.ServiceExportEvent) {
	s.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
		Kind:   announcements.MultiClusterServiceExportAccepted,
		OldObj: nil,
		NewObj: svcExportEvt,
	})
}

func (s *ControlPlaneServer) rejectServiceExport(svcExportEvt *mcsevent.ServiceExportEvent, err error) {
	svcExportEvt.Error = err.Error()

	s.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
		Kind:   announcements.MultiClusterServiceExportRejected,
		OldObj: nil,
		NewObj: svcExportEvt,
	})
}

func (s *ControlPlaneServer) GetBackground(key string) (*conn.Background, bool) {
	bg, exists := s.backgrounds[key]
	return bg, exists
}

func (s *ControlPlaneServer) AddBackground(key string, background *conn.Background) {
	s.backgrounds[key] = background
}

func (s *ControlPlaneServer) DestroyBackground(key string) {
	bg, exists := s.backgrounds[key]
	if !exists {
		return
	}

	close(bg.Context.StopCh)
	delete(s.backgrounds, key)
}
