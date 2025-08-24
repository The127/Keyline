package services

import (
	"Keyline/config"
	"Keyline/utils"
	"errors"
	"os"
	"path/filepath"
)

var ErrKeyPairNotFound = errors.New("KeyPair not found")

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
