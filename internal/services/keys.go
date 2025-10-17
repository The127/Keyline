package services

import (
	"Keyline/internal/caching"
	"Keyline/internal/config"
	"Keyline/utils"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:generate mockgen -destination=./mocks/key_store.go -package=mocks Keyline/internal/services KeyStore
type KeyStore interface {
	Get(virtualServerName string, algorithm config.SigningAlgorithm, kid string) (*KeyPair, error)
	GetAll(virtualServerName string) ([]KeyPair, error)
	GetAllForAlgorithm(virtualServerName string, algorithm config.SigningAlgorithm) ([]KeyPair, error)
	Add(virtualServerName string, keyPair KeyPair) error
	Remove(virtualServerName string, algorithm config.SigningAlgorithm, kid string) error
}

type memoryKeyStore struct {
	keyPairs map[string]KeyPair
}

func NewMemoryKeyStore() KeyStore {
	return &memoryKeyStore{
		keyPairs: make(map[string]KeyPair),
	}
}

func (m *memoryKeyStore) Get(virtualServerName string, algorithm config.SigningAlgorithm, kid string) (*KeyPair, error) {
	key := fmt.Sprintf("%s:%s:%s", virtualServerName, algorithm, kid)
	if keyPair, ok := m.keyPairs[key]; ok {
		return &keyPair, nil
	}
	return nil, nil
}

func (m *memoryKeyStore) GetAll(virtualServerName string) ([]KeyPair, error) {
	result := make([]KeyPair, 0)
	for key, keyPair := range m.keyPairs {
		if strings.HasPrefix(key, virtualServerName+":") {
			result = append(result, keyPair)
		}
	}
	return result, nil
}

func (m *memoryKeyStore) GetAllForAlgorithm(virtualServerName string, algorithm config.SigningAlgorithm) ([]KeyPair, error) {
	result := make([]KeyPair, 0)
	for key, keyPair := range m.keyPairs {
		if strings.HasPrefix(key, virtualServerName+":"+string(algorithm)+":") {
			result = append(result, keyPair)
		}
	}
	return result, nil
}

func (m *memoryKeyStore) Add(virtualServerName string, keyPair KeyPair) error {
	key := fmt.Sprintf("%s:%s:%s", virtualServerName, keyPair.algorithm, keyPair.ComputeKid())
	m.keyPairs[key] = keyPair
	return nil
}

func (m *memoryKeyStore) Remove(virtualServerName string, algorithm config.SigningAlgorithm, kid string) error {
	key := fmt.Sprintf("%s:%s:%s", virtualServerName, algorithm, kid)
	delete(m.keyPairs, key)
	return nil
}

type directoryKeyStore struct {
}

func NewDirectoryKeyStore() KeyStore {
	return &directoryKeyStore{}
}

func (d *directoryKeyStore) Add(virtualServerName string, keyPair KeyPair) error {
	algPath := d.getPath(virtualServerName, keyPair.algorithm)
	info, err := os.Stat(algPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = os.MkdirAll(algPath, 0700)
		if err != nil {
			return err
		}
	} else if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", algPath)
	}

	keyPath := filepath.Join(algPath, keyPair.ComputeKid())
	err = os.WriteFile(keyPath, keyPair.PrivateKeyBytes(), 0600)
	if err != nil {
		return fmt.Errorf("writing key: %w", err)
	}

	return nil
}

func (d *directoryKeyStore) Remove(virtualServerName string, algorithm config.SigningAlgorithm, kid string) error {
	algPath := d.getPath(virtualServerName, algorithm)
	keyPath := filepath.Join(algPath, kid)

	info, err := os.Stat(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("path %s is a directory", keyPath)
	}

	err = os.Remove(keyPath)
	if err != nil {
		return fmt.Errorf("removing key: %w", err)
	}

	return nil
}

func (d *directoryKeyStore) GetAllForAlgorithm(virtualServerName string, algorithm config.SigningAlgorithm) ([]KeyPair, error) {
	algPath := d.getPath(virtualServerName, algorithm)

	files, err := os.ReadDir(algPath)
	if err != nil {
		return nil, err
	}

	var keyPairs []KeyPair //nolint:prealloc
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		privateKeyBytes, err := os.ReadFile(filepath.Join(algPath, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading key: %w", err)
		}

		privateKey, publicKey := utils.ImportPrivateKey(privateKeyBytes, algorithm)
		keyPairs = append(keyPairs, KeyPair{
			algorithm:  algorithm,
			publicKey:  publicKey,
			privateKey: privateKey,
		})
	}

	return keyPairs, nil
}

func (d *directoryKeyStore) GetAll(virtualServerName string) ([]KeyPair, error) {
	var keyPairs []KeyPair

	for _, alg := range config.SupportedSigningAlgorithms {
		algKeyPairs, err := d.GetAllForAlgorithm(virtualServerName, alg)
		if err != nil {
			return nil, err
		}

		keyPairs = append(keyPairs, algKeyPairs...)
	}

	return keyPairs, nil
}

func (d *directoryKeyStore) Get(virtualServerName string, algorithm config.SigningAlgorithm, kid string) (*KeyPair, error) {
	algPath := d.getPath(virtualServerName, algorithm)
	keyPath := filepath.Join(algPath, kid)

	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading key: %w", err)
	}

	privateKey, publicKey := utils.ImportPrivateKey(privateKeyBytes, algorithm)
	return &KeyPair{
		algorithm:  algorithm,
		publicKey:  publicKey,
		privateKey: privateKey,
	}, nil
}

func (d *directoryKeyStore) getPath(virtualServerName string, algorithm config.SigningAlgorithm) string {
	return filepath.Join(config.C.KeyStore.Directory.Path, virtualServerName, string(algorithm))
}

type KeyCache caching.Cache[string, KeyPair]

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

func (k *KeyPair) ComputeKid() string {
	hash := sha256.Sum256(k.PublicKeyBytes())
	return base64.RawURLEncoding.EncodeToString(hash[:])
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

//go:generate mockgen -destination=./mocks/key_service.go -package=mocks Keyline/internal/services KeyService
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

	err := s.store.Add(virtualServerName, keyPair)
	if err != nil {
		return KeyPair{}, fmt.Errorf("storing key pair: %w", err)
	}

	return keyPair, nil
}

func (s *keyServiceImpl) GetKey(virtualServerName string, algorithm config.SigningAlgorithm) KeyPair {
	keyPair, ok := s.cache.TryGet(virtualServerName)
	if !ok {
		keyPairs, err := s.store.GetAllForAlgorithm(virtualServerName, algorithm)
		if err != nil {
			panic(fmt.Errorf("getting key pairs: %w", err))
		}

		if len(keyPairs) == 0 {
			panic(fmt.Errorf("no key pairs found for virtual server %s", virtualServerName))
		}

		keyPair = keyPairs[0]
		s.cache.Put(virtualServerName, keyPair)
	}

	return keyPair
}
