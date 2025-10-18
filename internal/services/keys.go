package services

import (
	"Keyline/internal/caching"
	"Keyline/internal/config"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type KeyAlgorithmStrategy interface {
	Generate() (any, any, error)
	Import(serializedPrivateKey string) (any, any, error)
	Export(privateKey any) (string, error)
}

func GetKeyStrategy(algorithm config.SigningAlgorithm) KeyAlgorithmStrategy {
	switch algorithm {
	case config.SigningAlgorithmRS256:
		return &RSAKeyStrategy{}

	case config.SigningAlgorithmEdDSA:
		return &EdDSAKeyStrategy{}

	default:
		panic(fmt.Sprintf("not implemented for algorithm: %s", algorithm))
	}
}

type RSAKeyStrategy struct{}

func (s *RSAKeyStrategy) Generate() (any, any, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("generating key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

func (s *RSAKeyStrategy) Import(serializedPrivateKey string) (any, any, error) {
	key, err := x509.ParsePKCS1PrivateKey([]byte(serializedPrivateKey))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key: %w", err)
	}

	return key, &key.PublicKey, nil
}

func (s *RSAKeyStrategy) Export(privateKey any) (string, error) {
	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("invalid private key type, expected *rsa.PrivateKey got %T", privateKey)
	}

	key := x509.MarshalPKCS1PrivateKey(rsaPrivateKey)
	return string(key), nil
}

type EdDSAKeyStrategy struct{}

func (s *EdDSAKeyStrategy) Generate() (any, any, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating key pair: %w", err)
	}

	return privateKey, publicKey, nil
}

func (s *EdDSAKeyStrategy) Export(privateKey any) (string, error) {
	ed25519PrivateKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return "", fmt.Errorf("invalid private key type, expected ed25519.PrivateKey got %T", privateKey)
	}

	return base64.RawURLEncoding.EncodeToString(ed25519PrivateKey), nil
}

func (s *EdDSAKeyStrategy) Import(serializedPrivateKey string) (any, any, error) {
	privateKeyBytes, err := base64.RawURLEncoding.DecodeString(serializedPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding private key: %w", err)
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey, nil
}

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

type keyPairJson struct {
	Algorithm  string    `json:"algorithm"`
	PrivateKey string    `json:"private_key"`
	CreatedAt  time.Time `json:"created_at"`
	RotatesAt  time.Time `json:"rotates_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (d *directoryKeyStore) Serialize(keyPair KeyPair) ([]byte, error) {
	strategy := GetKeyStrategy(keyPair.algorithm)
	serializedPrivateKey, err := strategy.Export(keyPair.privateKey)
	if err != nil {
		return nil, fmt.Errorf("exporting key pair: %w", err)
	}

	dto := keyPairJson{
		Algorithm:  string(keyPair.algorithm),
		PrivateKey: serializedPrivateKey,
		CreatedAt:  keyPair.createdAt,
		RotatesAt:  keyPair.rotatesAt,
		ExpiresAt:  keyPair.expiresAt,
	}

	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("marshaling key pair: %w", err)
	}

	return bytes, nil
}

func (d *directoryKeyStore) Deserialize(data []byte) (KeyPair, error) {
	dto := keyPairJson{}
	err := json.Unmarshal(data, &dto)
	if err != nil {
		return KeyPair{}, fmt.Errorf("unmarshaling key pair: %w", err)
	}

	strategy := GetKeyStrategy(config.SigningAlgorithm(dto.Algorithm))
	privateKey, publicKey, err := strategy.Import(dto.PrivateKey)
	if err != nil {
		return KeyPair{}, fmt.Errorf("importing key pair: %w", err)
	}

	return KeyPair{
		algorithm:  config.SigningAlgorithm(dto.Algorithm),
		publicKey:  publicKey,
		privateKey: privateKey,
		createdAt:  dto.CreatedAt,
		rotatesAt:  dto.RotatesAt,
		expiresAt:  dto.ExpiresAt,
	}, nil
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

	serializedKeyPair, err := d.Serialize(keyPair)
	if err != nil {
		return fmt.Errorf("serializing key pair: %w", err)
	}

	err = os.WriteFile(keyPath, serializedKeyPair, 0600)
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

		serializedKeyPair, err := os.ReadFile(filepath.Join(algPath, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading key: %w", err)
		}

		strategy := GetKeyStrategy(algorithm)
		privateKey, publicKey, err := strategy.Import(string(serializedKeyPair))
		if err != nil {
			return nil, fmt.Errorf("importing key pair: %w", err)
		}

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

	strategy := GetKeyStrategy(algorithm)
	privateKey, publicKey, err := strategy.Import(string(privateKeyBytes))
	if err != nil {
		return nil, fmt.Errorf("importing key pair: %w", err)
	}

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
	createdAt  time.Time
	rotatesAt  time.Time
	expiresAt  time.Time
}

func NewKeyPair(algorithm config.SigningAlgorithm, publicKey any, privateKey any) KeyPair {
	// TODO: use clock service and settings in virtual server
	now := time.Now()
	rotatesAt := now.Add(time.Hour * 24 * 20)
	expiresAt := now.Add(time.Hour * 24 * 30)

	return KeyPair{
		algorithm:  algorithm,
		publicKey:  publicKey,
		privateKey: privateKey,
		createdAt:  now,
		rotatesAt:  rotatesAt,
		expiresAt:  expiresAt,
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

func (k *KeyPair) CreatedAt() time.Time {
	return k.createdAt
}

func (k *KeyPair) RotatesAt() time.Time {
	return k.rotatesAt
}

func (k *KeyPair) ExpiresAt() time.Time {
	return k.expiresAt
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
	strategy := GetKeyStrategy(algorithm)
	privateKey, publicKey, err := strategy.Generate()
	if err != nil {
		return KeyPair{}, fmt.Errorf("generating key pair: %w", err)
	}

	keyPair := KeyPair{
		algorithm:  algorithm,
		publicKey:  publicKey,
		privateKey: privateKey,
	}

	err = s.store.Add(virtualServerName, keyPair)
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
