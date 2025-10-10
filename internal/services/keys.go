package services

import (
	"Keyline/internal/config"
	"Keyline/utils"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrKeyPairNotFound = errors.New("KeyPair not found")

//go:generate mockgen -destination=./mocks/key_store.go -package=mocks Keyline/services KeyStore
type KeyStore interface {
	Store(virtualServerName string, keyPair KeyPair) error
	Load(virtualServerName string, algorithm config.SigningAlgorithm) (KeyPair, error)
}

type directoryKeyStore struct {
}

func NewDirectoryKeyStore() KeyStore {
	return &directoryKeyStore{}
}

func (d *directoryKeyStore) getPath(virtualServerName string) string {
	return filepath.Join(config.C.KeyStore.Directory.Path, virtualServerName)
}

func (d *directoryKeyStore) Store(virtualServerName string, keyPair KeyPair) error {
	path := d.getPath(virtualServerName)
	privateKeyBytes := keyPair.PrivateKeyBytes()

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.New("keystore directory does not exist: " + dir)
	}

	return os.WriteFile(path, privateKeyBytes, 0600)
}

func (d *directoryKeyStore) Load(virtualServerName string, algorithm config.SigningAlgorithm) (KeyPair, error) {
	path := d.getPath(virtualServerName)

	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return KeyPair{}, ErrKeyPairNotFound
		}
		return KeyPair{}, err
	}

	privateKey, publicKey := utils.ImportPrivateKey(privateKeyBytes, algorithm)

	return KeyPair{
		algorithm:  algorithm,
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

type KeyCache Cache[string, KeyPair]

type KeyPair struct {
	algorithm  config.SigningAlgorithm
	publicKey  any
	privateKey any
}

func NewKeyPair(algorithm config.SigningAlgorithm, publicKey any, privateKey any) KeyPair {
	return KeyPair{
		algorithm:  algorithm,
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (k *KeyPair) PublicKeyBytes() []byte {
	switch k.algorithm {
	case config.SigningAlgorithmEdDSA:
		return k.publicKey.(ed25519.PublicKey)

	case config.SigningAlgorithmRS256:
		var rsaPubKey = k.publicKey.(*rsa.PublicKey)
		rsaPubKeyBytes, err := x509.MarshalPKIXPublicKey(rsaPubKey)
		if err != nil {
			panic(fmt.Errorf("marshaling public key: %w", err))
		}
		return rsaPubKeyBytes
	default:
		panic(fmt.Sprintf("not implemented for algorithm: %s", k.algorithm))
	}
}

func (k *KeyPair) PrivateKey() any {
	return k.privateKey
}

func (k *KeyPair) Algorithm() config.SigningAlgorithm {
	return k.algorithm
}

func (k *KeyPair) PrivateKeyBytes() []byte {
	switch k.algorithm {
	case config.SigningAlgorithmEdDSA:
		return k.privateKey.(ed25519.PrivateKey)
	case config.SigningAlgorithmRS256:
		rsaKey := k.privateKey.(*rsa.PrivateKey)
		return x509.MarshalPKCS1PrivateKey(rsaKey)
	default:
		panic(fmt.Sprintf("not implemented for algorithm: %s", k.algorithm))
	}
}

func (k *KeyPair) PublicKey() any {
	return k.publicKey
}

//go:generate mockgen -destination=./mocks/key_service.go -package=mocks Keyline/services KeyService
type KeyService interface {
	Generate(virtualServerName string, algorithm config.SigningAlgorithm) (KeyPair, error)
	GetKey(virtualServerName string, algorithm config.SigningAlgorithm) KeyPair
}

type keyServiceImpl struct {
	cache KeyCache
	store KeyStore
}

func NewKeyService(cache KeyCache, store KeyStore) KeyService {
	return &keyServiceImpl{
		cache: cache,
		store: store,
	}
}

func (s *keyServiceImpl) Generate(virtualServerName string, algorithm config.SigningAlgorithm) (KeyPair, error) {
	privateKey, publicKey := utils.GenerateKeyPair(algorithm)

	keyPair := KeyPair{
		algorithm:  algorithm,
		publicKey:  publicKey,
		privateKey: privateKey,
	}

	err := s.store.Store(virtualServerName, keyPair)
	if err != nil {
		return KeyPair{}, fmt.Errorf("storing key pair: %w", err)
	}

	return keyPair, nil
}

func (s *keyServiceImpl) GetKey(virtualServerName string, algorithm config.SigningAlgorithm) KeyPair {
	keyPair, ok := s.cache.TryGet(virtualServerName)
	if !ok {
		var err error
		keyPair, err = s.store.Load(virtualServerName, algorithm)
		switch {
		case errors.Is(err, ErrKeyPairNotFound):
			// TODO: regenerate
			panic(fmt.Errorf("loading keypair for %s: %w", virtualServerName, err))

		case err != nil:
			panic(fmt.Errorf("loading keypair for %s: %w", virtualServerName, err))
		}

		s.cache.Put(virtualServerName, keyPair)
	}

	return keyPair
}
