package providers

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/certificate"
)

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) List() ([]*v1alpha3.MeshRootCertificate, error) {
	return []*v1alpha3.MeshRootCertificate{
		c.mrc,
	}, nil
}

// Watch is a basic Watch implementation for the MRC attached to the compat client
func (c *MRCCompatClient) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	ch := make(chan certificate.MRCEvent)
	go func() {
		ch <- certificate.MRCEvent{
			Type: certificate.MRCEventAdded,
			MRC:  c.mrc,
		}
		close(ch)
	}()

	return ch, nil
}
