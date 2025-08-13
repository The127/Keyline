package services

import (
	"Keyline/utils"
	"crypto/ed25519"
	"errors"
	"fmt"
)

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
