package utils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"strings"

	//  Import the crypto sha256 algorithm for the docker image parser to work
	_ "crypto/sha256"
	//  Import the crypto/sha512 algorithm for the docker image parser to work with 384 and 512 sha hashes
	_ "crypto/sha512"

	dockerref "github.com/docker/distribution/reference"
)

// ParseImageName parses a docker image string into three parts: repo, tag and digest.
// If both tag and digest are empty, a default image tag will be returned.
func ParseImageName(image string) (string, string, string, error) {
	named, err := dockerref.ParseNormalizedNamed(image)
	if err != nil {
		return "", "", "", fmt.Errorf("couldn't parse image name: %v", err)
	}

	repoToPull := named.Name()
	var tag, digest string

	tagged, ok := named.(dockerref.Tagged)
	if ok {
		tag = tagged.Tag()
	}

	digested, ok := named.(dockerref.Digested)
	if ok {
		digest = digested.Digest().String()
	}
	// If no tag was specified, use the default "latest".
	if len(tag) == 0 && len(digest) == 0 {
		tag = "latest"
	}

	return repoToPull, tag, digest, nil
}

func ImagePullPolicyByTag(image string) corev1.PullPolicy {
	_, tag, _, _ := ParseImageName(image)

	switch tag {
	case "latest", "dev", "edge":
		return corev1.PullAlways
	}

	if strings.HasSuffix(tag, "dev") || strings.HasSuffix(tag, "edge") {
		return corev1.PullAlways
	}

	return corev1.PullIfNotPresent
}
