package utils

import (
	"fmt"
	"hash/adler32"

	"github.com/mitchellh/hashstructure/v2"
	corev1 "k8s.io/api/core/v1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

// SimpleHash returns a hash string of the given object.
func SimpleHash(obj interface{}) string {
	hash, err := hashstructure.Hash(obj, hashstructure.FormatV2, nil)

	if err != nil {
		log.Error().Msgf("Not able convert Data to hash, error: %s", err.Error())
		return ""
	}

	return fmt.Sprintf("%x", hash)
}

// GetSecretDataHash returns a hash of the given secret data.
func GetSecretDataHash(secret *corev1.Secret) uint32 {
	secretDataHasher := adler32.New()
	hashutil.DeepHashObject(secretDataHasher, secret.Data)
	return secretDataHasher.Sum32()
}
