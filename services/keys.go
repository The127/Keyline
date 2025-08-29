package services

import (
	"Keyline/config"
	"Keyline/utils"
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrKeyPairNotFound = errors.New("KeyPair not found")

//go:generate mockgen -destination=./mocks/key_store.go -package=mocks Keyline/services KeyStore
type KeyStore interface {
	Store(virtualServerName string, keyPair KeyPair) error
	Load(virtualServerName string) (KeyPair, error)
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
	privateKeyBytes := utils.ExportPrivateKey(keyPair.privateKey)

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.New("keystore directory does not exist: " + dir)
	}

	return os.WriteFile(path, privateKeyBytes, 0600)
}

func (d *directoryKeyStore) Load(virtualServerName string) (KeyPair, error) {
	path := d.getPath(virtualServerName)

	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return KeyPair{}, ErrKeyPairNotFound
		}
		return KeyPair{}, err
	}

	privateKey, publicKey := utils.ImportPrivateKey(privateKeyBytes)

	return KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

type KeyCache Cache[string, KeyPair]

type KeyPair struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func (k *KeyPair) PublicKey() ed25519.PublicKey {
	return k.publicKey
}

func (k *KeyPair) PrivateKey() ed25519.PrivateKey {
	return k.privateKey
}

//go:generate mockgen -destination=./mocks/key_service.go -package=mocks Keyline/services KeyService
type KeyService interface {
	Generate(virtualServerName string) (KeyPair, error)
	GetKey(virtualServerName string) KeyPair
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

func (s *keyServiceImpl) Generate(virtualServerName string) (KeyPair, error) {
	privateKey, publicKey := utils.GenerateKeyPair()

	keyPair := KeyPair{
		publicKey:  publicKey,
		privateKey: privateKey,
	}

	err := s.store.Store(virtualServerName, keyPair)
	if err != nil {
		return KeyPair{}, fmt.Errorf("storing key pair: %w", err)
	}

	return keyPair, nil
}

func (s *keyServiceImpl) GetKey(virtualServerName string) KeyPair {
	keyPair, ok := s.cache.TryGet(virtualServerName)
	if !ok {
		var err error
		keyPair, err = s.store.Load(virtualServerName)
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
