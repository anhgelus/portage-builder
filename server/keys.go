package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand/v2"
	"os"
)

func GenerateServerKeys(name string, rootPath Key, serverPath Key) error {
	root := x509.Certificate{
		Subject: pkix.Name{
			CommonName: name,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	rootKey, err := generateCert(rootPath, &root, &root, nil)
	if err != nil {
		return err
	}
	server := x509.Certificate{
		Subject: pkix.Name{
			CommonName: name + " server",
		},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	_, err = generateCert(serverPath, &server, &root, rootKey)
	return err
}

func GenerateUserKey(user Key, name string, rootPath Key) error {
	rootPEMCert, keyPEM, err := rootPath.Read()
	if err != nil {
		return err
	}
	rootBlock, _ := pem.Decode(rootPEMCert)
	if err != nil {
		return err
	}
	rootCert, err := x509.ParseCertificate(rootBlock.Bytes)
	if err != nil {
		return err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if err != nil {
		return err
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}
	certif := x509.Certificate{
		Subject: pkix.Name{
			CommonName: name,
		},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = generateCert(user, &certif, rootCert, key)
	return err
}

func pemWrite(path, kind string, b []byte) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  kind,
		Bytes: b,
	}), 0o600)
}

func generateCert(path Key, template, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (*ecdsa.PrivateKey, error) {
	x := big.NewInt(rand.Int64())
	y := big.NewInt(rand.Int64())
	template.SerialNumber = x.Mul(x, y)
	key, err := ecdsa.GenerateKey(elliptic.P256(), nil)
	if err != nil {
		return nil, err
	}
	if parentKey == nil {
		parentKey = key
	}
	cert, err := x509.CreateCertificate(nil, template, parent, &key.PublicKey, parentKey)
	if err != nil {
		return nil, err
	}
	err = pemWrite(path.PemFile, "CERTIFICATE", cert)
	if err != nil {
		return nil, err
	}
	rawKey, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return key, pemWrite(path.PrivateKeyFile, "EC PRIVATE KEY", rawKey)
}
