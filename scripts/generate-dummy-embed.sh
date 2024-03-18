#!/bin/bash

# This script is used to generate dummy embedded files for CI purposes.
if [ ! -f "cmd/cli/chart.tgz" ]; then
touch cmd/cli/chart.tgz
fi

if [ ! -f "pkg/controllers/namespacedingress/v1alpha1/chart.tgz" ]; then
touch pkg/controllers/namespacedingress/v1alpha1/chart.tgz
fi

if [ ! -f "pkg/controllers/gateway/v1beta1/chart.tgz" ]; then
touch pkg/controllers/gateway/v1beta1/chart.tgz
fi

if [ ! -f "pkg/controllers/connector/v1alpha1/chart.tgz" ]; then
touch pkg/controllers/connector/v1alpha1/chart.tgz
fi