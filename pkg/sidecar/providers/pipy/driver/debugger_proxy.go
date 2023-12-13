// Package driver implements debugger's methods.
package driver

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/registry"
)

const (
	uuidQueryKey        = "uuid"
	proxyConfigQueryKey = "cfg"
)

func (sd PipySidecarDriver) getProxies(proxyRegistry *registry.ProxyRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		proxyConfigDump := r.URL.Query()[proxyConfigQueryKey]
		uuid := r.URL.Query()[uuidQueryKey]

		switch {
		case len(uuid) == 0:
			sd.printProxies(proxyRegistry, w)
		case len(proxyConfigDump) > 0:
			sd.getConfigDump(proxyRegistry, uuid[0], w)
		default:
			sd.getProxy(proxyRegistry, uuid[0], w)
		}
	})
}

func (sd PipySidecarDriver) printProxies(proxyRegistry *registry.ProxyRegistry, w http.ResponseWriter) {
	// This function is needed to convert the list of connected proxies to
	// the type (map) required by the printProxies function.
	proxyMap := proxyRegistry.ListConnectedProxies()
	proxies := make([]models.Proxy, 0, len(proxyMap))
	for _, proxy := range proxyMap {
		proxies = append(proxies, proxy)
	}

	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].GetIdentity().String() < proxies[j].GetIdentity().String()
	})

	_, _ = fmt.Fprintf(w, "<h1>Connected Proxies (%d):</h1>", len(proxies))
	_, _ = fmt.Fprint(w, `<table>`)
	_, _ = fmt.Fprint(w, "<tr><td>#</td><td>Pipy's Service Identity</td><td>Pipy's UUID</td><td>Connected At</td><td>How long ago</td><td>tools</td></tr>")
	for idx, proxy := range proxies {
		ts := proxy.GetConnectedAt()
		proxyURL := fmt.Sprintf("/debug/proxy?%s=%s", uuidQueryKey, proxy.GetUUID())
		configDumpURL := fmt.Sprintf("%s&%s=%t", proxyURL, proxyConfigQueryKey, true)
		_, _ = fmt.Fprintf(w, `<tr><td>%d:</td><td>%s</td><td>%s</td><td>%+v</td><td>(%+v ago)</td><td><a href="%s">certs</a></td><td><a href="%s">cfg</a></td></tr>`,
			idx+1, proxy.GetIdentity(), proxy.GetUUID(), ts, time.Since(ts), proxyURL, configDumpURL)
	}
	_, _ = fmt.Fprint(w, `</table>`)
}

func (sd PipySidecarDriver) getConfigDump(proxyRegistry *registry.ProxyRegistry, uuid string, w http.ResponseWriter) {
	proxy := proxyRegistry.GetConnectedProxy(uuid)
	if proxy != nil {
		msg := fmt.Sprintf("Proxy for UUID %s not found, may have been disconnected", uuid)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	if !proxy.VM {
		pod, err := sd.ctx.MeshCatalog.GetKubeController().GetPodForProxy(proxy)
		if err != nil {
			msg := fmt.Sprintf("Error getting Pod from proxy %s", proxy.GetName())
			log.Error().Err(err).Msg(msg)
			http.Error(w, msg, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		pipyConfig := sd.getSidecarConfig(pod, "config_dump")
		_, _ = fmt.Fprintf(w, "%s", pipyConfig)
	}
}

func (sd PipySidecarDriver) getProxy(proxyRegistry *registry.ProxyRegistry, uuid string, w http.ResponseWriter) {
	proxy := proxyRegistry.GetConnectedProxy(uuid)
	if proxy == nil {
		msg := fmt.Sprintf("Proxy for UUID %s not found, may have been disconnected", uuid)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	if !proxy.VM {
		pod, err := sd.ctx.MeshCatalog.GetKubeController().GetPodForProxy(proxy)
		if err != nil {
			msg := fmt.Sprintf("Error getting Pod from proxy %s", proxy.GetName())
			log.Error().Err(err).Msg(msg)
			http.Error(w, msg, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		pipyConfig := sd.getSidecarConfig(pod, "certs")
		_, _ = fmt.Fprintf(w, "%s", pipyConfig)
	}
}
