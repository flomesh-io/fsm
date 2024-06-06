package driver

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	v1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/debugger"
)

func (sd PipySidecarDriver) getSidecarConfig(pod *v1.Pod, url string) string {
	log.Debug().Msgf("Getting Pipy config on Pod with UID=%s", pod.ObjectMeta.UID)

	minPort := 16000
	maxPort := 18000

	portFwdRequest := debugger.PortForward{
		Pod:       pod,
		LocalPort: rand.Intn(maxPort-minPort) + minPort, // #nosec G404
		PodPort:   15000,
		Stop:      make(chan struct{}),
		Ready:     make(chan struct{}),
	}

	go debugger.ForwardPort(sd.ctx.KubeConfig, portFwdRequest)

	<-portFwdRequest.Ready

	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/%s", "localhost", portFwdRequest.LocalPort, url))
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	defer func() {
		portFwdRequest.Stop <- struct{}{}
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Error getting Pipy config on Pod with UID=%s; HTTP Error %d", pod.ObjectMeta.UID, resp.StatusCode)
		portFwdRequest.Stop <- struct{}{}
		return fmt.Sprintf("Error: %s", err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	return string(bodyBytes)
}
