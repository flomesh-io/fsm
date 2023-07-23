package version

import (
	"github.com/blang/semver"
	"k8s.io/client-go/kubernetes"
	"strings"
)

var (
	ServerVersion = semver.Version{Major: 0, Minor: 0, Patch: 0}
)

var (
	MinK8sVersion              = semver.Version{Major: 1, Minor: 19, Patch: 0}
	MinEndpointSliceVersion    = semver.Version{Major: 1, Minor: 21, Patch: 0}
	MinK8sVersionForGatewayAPI = MinEndpointSliceVersion
)

func getServerVersion(kubeClient kubernetes.Interface) (semver.Version, error) {
	serverVersion, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		log.Error().Msgf("unable to get Server Version: %s", err)
		return semver.Version{Major: 0, Minor: 0, Patch: 0}, err
	}

	gitVersion := serverVersion.GitVersion
	if len(gitVersion) > 1 && strings.HasPrefix(gitVersion, "v") {
		gitVersion = gitVersion[1:]
	}

	return semver.MustParse(gitVersion), nil
}

func detectServerVersion(kubeClient kubernetes.Interface) {
	if ServerVersion.EQ(semver.Version{Major: 0, Minor: 0, Patch: 0}) {
		ver, err := getServerVersion(kubeClient)
		if err != nil {
			log.Error().Msgf("unable to get server version: %s", err)
			panic(err)
		}

		ServerVersion = ver
	}
}

func IsSupportedK8sVersion(kubeClient kubernetes.Interface) bool {
	detectServerVersion(kubeClient)
	return ServerVersion.GTE(MinK8sVersion)
}

func IsEndpointSliceEnabled(kubeClient kubernetes.Interface) bool {
	detectServerVersion(kubeClient)
	return ServerVersion.GTE(MinEndpointSliceVersion)
}

func IsSupportedK8sVersionForGatewayAPI(kubeClient kubernetes.Interface) bool {
	return IsEndpointSliceEnabled(kubeClient)
}
