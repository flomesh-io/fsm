#!/bin/bash

./scripts/port-forward-bookbuyer-ui.sh &
./scripts/port-forward-bookstore-ui.sh app=bookstore &
./scripts/port-forward-bookstore-ui-v2.sh &
./scripts/port-forward-bookstore-ui-v1.sh &
./scripts/port-forward-bookthief-ui.sh &
./scripts/port-forward-fsm-debug.sh &
./scripts/port-forward-grafana.sh &
./scripts/port-forward-jaeger.sh &
./scripts/port-forward-prometheus.sh &

wait

