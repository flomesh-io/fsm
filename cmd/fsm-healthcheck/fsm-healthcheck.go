// Package main implements the main entrypoint for fsm-healthcheck.
// fsm-healthcheck provides TCPSocket probe support for pods in the mesh.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/pflag"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/version"
)

var log = logger.New("fsm-healthcheck/main")

func main() {
	log.Info().Msgf("Starting fsm-healthcheck %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

	var verbosity string

	flags := pflag.NewFlagSet("fsm-healthcheck", pflag.ExitOnError)
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")

	err := flags.Parse(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("parsing flags")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	stop := signals.RegisterExitHandlers()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/fsm-healthcheck", healthcheckHandler)

	// Initialize fsm-healthcheck HTTP server
	server := &http.Server{
		Addr:              ":15904",
		Handler:           serverMux,
		ReadHeaderTimeout: time.Second * 10,
	}

	log.Info().Msgf("Starting FSM healthcheck HTTP server")
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start FSM healthcheck HTTP server")
		}
	}()

	<-stop

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down FSM healthcheck HTTP server")
	} else {
		log.Info().Msg("Done shutting down FSM healthcheck HTTP server")
	}
}

// healthcheckHandler handles HTTP requests and attempts to open a socket to a container
// on the TCP port specified in the request's header.
// If a connection is successfully established, the connection is closed and the response
// status code will be 200.
func healthcheckHandler(w http.ResponseWriter, req *http.Request) {
	port := req.Header.Get("Original-Tcp-Port")
	if port == "" {
		msg := "Header Original-Tcp-Port not found in request"
		log.Error().Msg(msg)
		setHealthcheckResponse(w, http.StatusBadRequest, msg)
		return
	}

	address := fmt.Sprintf("%s:%s", constants.LocalhostIPAddress, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		msg := fmt.Sprintf("Failed to establish connection to %s", address)
		log.Error().Err(err).Msg(msg)
		setHealthcheckResponse(w, http.StatusNotFound, msg)
		return
	}

	if err = conn.Close(); err != nil {
		log.Error().Err(err).Msgf("Failed to close connection to %s", address)
	}

	msg := fmt.Sprintf("Successfully established connection to %s", address)
	log.Debug().Msg(msg)
	setHealthcheckResponse(w, http.StatusOK, msg)
}

func setHealthcheckResponse(w http.ResponseWriter, responseCode int, msg string) {
	w.WriteHeader(responseCode)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
	}
}
