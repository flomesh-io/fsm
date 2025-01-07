package catalog

import (
	"strconv"
	"strings"
	"time"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/service"
)

func (mc *MeshCatalog) GetTrafficWarmupPolicy(svc service.MeshService) *configv1alpha3.TrafficWarmupSpec {
	if !mc.configurator.GetFeatureFlags().EnableTrafficWarmupPolicy {
		return nil
	}

	if svcWarmupPolicy := mc.policyController.GetTrafficWarmupPolicy(svc); svcWarmupPolicy != nil && svcWarmupPolicy.Enable {
		return svcWarmupPolicy
	}

	if ns := mc.kubeController.GetNamespace(svc.Namespace); ns != nil {
		if enableAnno, exists := ns.Annotations[constants.TrafficWarmupEnableAnnotation]; exists {
			enabled := false
			switch strings.ToLower(enableAnno) {
			case "enabled", "yes", "true":
				enabled = true
			case "disabled", "no", "false":
				enabled = false
			default:
				log.Error().Msgf("invalid traffic warmup enable annotation, namespace:%s value:%s", svc.Namespace, enableAnno)
			}

			if enabled {
				nsWarmupPolicy := new(configv1alpha3.TrafficWarmupSpec)
				if durationAnno, exists := ns.Annotations[constants.TrafficWarmupDurationAnnotation]; exists {
					if duration, err := time.ParseDuration(durationAnno); err == nil {
						nsWarmupPolicy.Duration.Duration = duration
					} else {
						log.Error().Err(err).Msgf("invalid traffic warmup duration annotation, namespace:%s value:%s", svc.Namespace, durationAnno)
						return nil
					}
				} else {
					nsWarmupPolicy.Duration.Duration = time.Second * 90
				}
				if minWeightAnno, exists := ns.Annotations[constants.TrafficWarmupMinWeightAnnotation]; exists {
					if minWeight, err := strconv.ParseUint(minWeightAnno, 10, 32); err == nil {
						nsWarmupPolicy.MinWeight = &minWeight
					} else {
						log.Error().Err(err).Msgf("invalid traffic warmup minweight annotation, namespace:%s value:%s", svc.Namespace, minWeightAnno)
						return nil
					}
				}
				if maxWeightAnno, exists := ns.Annotations[constants.TrafficWarmupMaxWeightAnnotation]; exists {
					if maxWeight, err := strconv.ParseUint(maxWeightAnno, 10, 32); err == nil {
						nsWarmupPolicy.MaxWeight = &maxWeight
					} else {
						log.Error().Err(err).Msgf("invalid traffic warmup maxweight annotation, namespace:%s value:%s", svc.Namespace, maxWeightAnno)
						return nil
					}
				}
				if aggressionAnno, exists := ns.Annotations[constants.TrafficWarmupAggressionAnnotation]; exists {
					if aggression, err := strconv.ParseFloat(aggressionAnno, 64); err == nil {
						nsWarmupPolicy.Aggression = &aggression
					} else {
						log.Error().Err(err).Msgf("invalid traffic warmup aggression annotation, namespace:%s value:%s", svc.Namespace, aggressionAnno)
						return nil
					}
				}
				return nsWarmupPolicy
			}
		}
	}

	if globalWarmupPolicy := mc.configurator.GetMeshConfig().Spec.Warmup; globalWarmupPolicy.Enable {
		return &globalWarmupPolicy
	}

	return nil
}
