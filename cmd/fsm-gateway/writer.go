package main

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewayStatusWriter struct {
	addresses     chan []gwv1.GatewayStatusAddress
	statusUpdater status.Updater
}

func (w *gatewayStatusWriter) NeedLeaderElection() bool {
	return true
}

func (w *gatewayStatusWriter) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case addresses := <-w.addresses:
			log.Info().Msgf("[GW] Received new addresses: %v", addresses)

			w.statusUpdater.Send(status.Update{
				NamespacedName: types.NamespacedName{
					Name:      gatewayName,
					Namespace: gatewayNamespace,
				},
				Resource: &gwv1.Gateway{},
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					gw, ok := obj.(*gwv1.Gateway)
					if !ok {
						log.Error().Msgf("Unexpected object type: %T", obj)
						panic("not a gateway resource")
					}

					gwCopy := gw.DeepCopy()
					gwCopy.Status.Addresses = addresses

					return gwCopy
				}),
			})
		}
	}
}
