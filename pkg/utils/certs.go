/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package utils

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"k8s.io/klog/v2"
)

func ConvertPEMCertToX509(pemCrt []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemCrt)
	if block == nil {
		klog.Error("No valid certificate in PEM")
		return nil, fmt.Errorf("no valid certificate in PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		klog.Errorf("failed to convert PEM certificate to x509, %s ", err.Error())
		return nil, err
	}
	return x509Cert, nil
}

func ConvertPEMPrivateKeyToX509(pemKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		klog.Error("No valid private key in PEM")
		return nil, fmt.Errorf("no valid private key in PEM")
	}

	x509Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		klog.Errorf("failed to convert PEM private key to x509, %s ", err.Error())
		return nil, err
	}

	return x509Key.(*rsa.PrivateKey), nil
}

func CertToPEM(caBytes []byte) ([]byte, error) {
	caPEM := new(bytes.Buffer)
	if err := pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return nil, fmt.Errorf("encode cert: %s", err.Error())
	}

	return caPEM.Bytes(), nil
}

func RSAKeyToPEM(privateKey *rsa.PrivateKey) ([]byte, error) {
	privateBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %s", err.Error())
	}

	keyPEM := new(bytes.Buffer)
	if err := pem.Encode(keyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateBytes,
	}); err != nil {
		return nil, fmt.Errorf("encode key: %s", err.Error())
	}

	return keyPEM.Bytes(), nil
}

func CsrToPEM(csrBytes []byte) ([]byte, error) {
	csrPEM := new(bytes.Buffer)
	if err := pem.Encode(csrPEM, &(pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})); err != nil {
		return nil, fmt.Errorf("encode CSR: %s", err.Error())
	}

	return csrPEM.Bytes(), nil
}
