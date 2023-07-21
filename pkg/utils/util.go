package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"hash/adler32"
	"hash/fnv"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

func SimpleHash(obj interface{}) string {
	hash, err := hashstructure.Hash(obj, hashstructure.FormatV2, nil)

	if err != nil {
		klog.Errorf("Not able convert Data to hash, error: %s", err.Error())
		return ""
	}

	return fmt.Sprintf("%x", hash)
}

//func Hash(data []byte) string {
//	return fmt.Sprintf("%x", sha256.Sum256(data))
//}

func GetBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func HashFNV(s string) string {
	hasher := fnv.New32a()
	// Hash.Write never returns an error
	_, _ = hasher.Write([]byte(s))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// GenerateRandom generates random string.
func GenerateRandom(n int) string {
	b := make([]byte, 8)
	_, _ = io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:n]
}

func GetSecretDataHash(secret *corev1.Secret) uint32 {
	secretDataHasher := adler32.New()
	hashutil.DeepHashObject(secretDataHasher, secret.Data)
	return secretDataHasher.Sum32()
}
