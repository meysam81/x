package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
)

// Cipher defines the interface for encryption/decryption operations
type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// Common errors
var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrInvalidKey        = errors.New("invalid key")
	ErrInvalidNonce      = errors.New("invalid nonce")
)

// ============================================================================
// AES-256-GCM Implementation
// ============================================================================

// AESGCMCipher implements AES-256-GCM encryption
type AESGCMCipher struct {
	key       []byte
	nonceSize int
	tagSize   int
}

// AESGCMOption configures AES-GCM parameters
type AESGCMOption func(*AESGCMCipher)

// WithAESGCMNonceSize sets custom nonce size (default: 12 bytes)
func WithAESGCMNonceSize(size int) AESGCMOption {
	return func(c *AESGCMCipher) {
		if size >= 12 && size <= 16 {
			c.nonceSize = size
		}
	}
}

// WithAESGCMTagSize sets custom tag size (default: 16 bytes)
func WithAESGCMTagSize(size int) AESGCMOption {
	return func(c *AESGCMCipher) {
		if size >= 12 && size <= 16 {
			c.tagSize = size
		}
	}
}

// NewAESGCM creates a new AES-256-GCM cipher with secure defaults
func NewAESGCM(key []byte, opts ...AESGCMOption) (*AESGCMCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: key must be 32 bytes for AES-256", ErrInvalidKey)
	}

	c := &AESGCMCipher{
		key:       make([]byte, 32),
		nonceSize: 12, // NIST recommended
		tagSize:   16, // Maximum security
	}
	copy(c.key, key)

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns: nonce || ciphertext || tag
func (c *AESGCMCipher) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCMWithNonceSize(block, c.nonceSize)
	if err != nil {
		return nil, err
	}

	// Pre-allocate result slice with exact size needed
	result := make([]byte, c.nonceSize, c.nonceSize+len(plaintext)+c.tagSize)
	if _, err := io.ReadFull(rand.Reader, result); err != nil {
		return nil, err
	}

	// Use result slice as nonce, then append ciphertext directly
	ciphertext := aead.Seal(result, result[:c.nonceSize], plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with AES-256-GCM
func (c *AESGCMCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.nonceSize+c.tagSize {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCMWithNonceSize(block, c.nonceSize)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:c.nonceSize]
	// Avoid slice copy by using direct slice reference
	encryptedData := ciphertext[c.nonceSize:]

	plaintext, err := aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}

// ============================================================================
// ChaCha20-Poly1305 Implementation
// ============================================================================

// ChaCha20Cipher implements ChaCha20-Poly1305 AEAD encryption
type ChaCha20Cipher struct {
	key []byte
}

// ChaCha20Option configures ChaCha20-Poly1305 parameters
type ChaCha20Option func(*ChaCha20Cipher)

// NewChaCha20 creates a new ChaCha20-Poly1305 cipher
func NewChaCha20(key []byte, opts ...ChaCha20Option) (*ChaCha20Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: key must be 32 bytes for ChaCha20-Poly1305", ErrInvalidKey)
	}

	c := &ChaCha20Cipher{
		key: make([]byte, 32),
	}
	copy(c.key, key)

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Encrypt encrypts plaintext using ChaCha20-Poly1305
// Returns: nonce || ciphertext || tag
func (c *ChaCha20Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(c.key)
	if err != nil {
		return nil, err
	}

	// Pre-allocate with exact size needed
	result := make([]byte, chacha20poly1305.NonceSize, chacha20poly1305.NonceSize+len(plaintext)+16)
	if _, err := io.ReadFull(rand.Reader, result); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(result, result[:chacha20poly1305.NonceSize], plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with ChaCha20-Poly1305
func (c *ChaCha20Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(c.key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < chacha20poly1305.NonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce := ciphertext[:chacha20poly1305.NonceSize]
	// Direct slice reference to avoid copy
	encryptedData := ciphertext[chacha20poly1305.NonceSize:]

	plaintext, err := aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}

// ============================================================================
// Argon2id Password-Based Encryption
// ============================================================================

// Argon2idCipher implements password-based encryption using Argon2id + AES-256-GCM
type Argon2idCipher struct {
	password []byte
	saltSize int
	time     uint32
	memory   uint32
	threads  uint8
	keyLen   uint32
}

// Argon2idOption configures Argon2id parameters
type Argon2idOption func(*Argon2idCipher)

// WithArgon2idTime sets the time parameter (iterations)
func WithArgon2idTime(time uint32) Argon2idOption {
	return func(c *Argon2idCipher) {
		c.time = time
	}
}

// WithArgon2idMemory sets the memory parameter in KiB
func WithArgon2idMemory(memory uint32) Argon2idOption {
	return func(c *Argon2idCipher) {
		c.memory = memory
	}
}

// WithArgon2idThreads sets the parallelism parameter
func WithArgon2idThreads(threads uint8) Argon2idOption {
	return func(c *Argon2idCipher) {
		c.threads = threads
	}
}

// WithArgon2idSaltSize sets the salt size in bytes
func WithArgon2idSaltSize(size int) Argon2idOption {
	return func(c *Argon2idCipher) {
		if size >= 16 {
			c.saltSize = size
		}
	}
}

// NewArgon2id creates a new Argon2id-based cipher with secure 2025 defaults
func NewArgon2id(password []byte, opts ...Argon2idOption) (*Argon2idCipher, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("%w: password cannot be empty", ErrInvalidKey)
	}

	c := &Argon2idCipher{
		password: make([]byte, len(password)),
		saltSize: 16,        // 128-bit salt
		time:     3,         // 3 iterations
		memory:   64 * 1024, // 64 MiB
		threads:  4,         // 4 parallel threads
		keyLen:   32,        // 256-bit key
	}
	copy(c.password, password)

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Encrypt encrypts plaintext using Argon2id + AES-256-GCM
// Returns: salt || aes_gcm_ciphertext
func (c *Argon2idCipher) Encrypt(plaintext []byte) ([]byte, error) {
	salt := make([]byte, c.saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := argon2.IDKey(c.password, salt, c.time, c.memory, c.threads, c.keyLen)
	defer func() {
		// Clear key from memory after use
		for i := range key {
			key[i] = 0
		}
	}()

	aesCipher, err := NewAESGCM(key)
	if err != nil {
		return nil, err
	}

	encrypted, err := aesCipher.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	// Pre-allocate result with exact size
	result := make([]byte, 0, len(salt)+len(encrypted))
	result = append(result, salt...)
	result = append(result, encrypted...)

	return result, nil
}

// Decrypt decrypts ciphertext encrypted with Argon2id + AES-256-GCM
func (c *Argon2idCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.saltSize {
		return nil, ErrInvalidCiphertext
	}

	salt := ciphertext[:c.saltSize]
	encryptedData := ciphertext[c.saltSize:]

	key := argon2.IDKey(c.password, salt, c.time, c.memory, c.threads, c.keyLen)
	defer func() {
		// Clear key from memory after use
		for i := range key {
			key[i] = 0
		}
	}()

	aesCipher, err := NewAESGCM(key)
	if err != nil {
		return nil, err
	}

	return aesCipher.Decrypt(encryptedData)
}

// ============================================================================
// PBKDF2 Password-Based Encryption
// ============================================================================

// PBKDF2Cipher implements password-based encryption using PBKDF2 + AES-256-GCM
type PBKDF2Cipher struct {
	password   []byte
	saltSize   int
	iterations int
	keyLen     int
	hashFunc   func() hash.Hash
}

// PBKDF2Option configures PBKDF2 parameters
type PBKDF2Option func(*PBKDF2Cipher)

// WithPBKDF2Iterations sets the iteration count
func WithPBKDF2Iterations(iterations int) PBKDF2Option {
	return func(c *PBKDF2Cipher) {
		if iterations >= 100000 {
			c.iterations = iterations
		}
	}
}

// WithPBKDF2SaltSize sets the salt size in bytes
func WithPBKDF2SaltSize(size int) PBKDF2Option {
	return func(c *PBKDF2Cipher) {
		if size >= 16 {
			c.saltSize = size
		}
	}
}

// WithPBKDF2Hash sets the hash function (default: SHA-256)
func WithPBKDF2Hash(hashFunc func() hash.Hash) PBKDF2Option {
	return func(c *PBKDF2Cipher) {
		c.hashFunc = hashFunc
	}
}

// NewPBKDF2 creates a new PBKDF2-based cipher with secure 2025 defaults
func NewPBKDF2(password []byte, opts ...PBKDF2Option) (*PBKDF2Cipher, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("%w: password cannot be empty", ErrInvalidKey)
	}

	c := &PBKDF2Cipher{
		password:   make([]byte, len(password)),
		saltSize:   16,     // 128-bit salt
		iterations: 600000, // OWASP 2025 recommendation for PBKDF2-SHA256
		keyLen:     32,     // 256-bit key
		hashFunc:   sha256.New,
	}
	copy(c.password, password)

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Encrypt encrypts plaintext using PBKDF2 + AES-256-GCM
// Returns: salt || aes_gcm_ciphertext
func (c *PBKDF2Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	salt := make([]byte, c.saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key(c.password, salt, c.iterations, c.keyLen, c.hashFunc)
	defer func() {
		// Clear key from memory after use
		for i := range key {
			key[i] = 0
		}
	}()

	aesCipher, err := NewAESGCM(key)
	if err != nil {
		return nil, err
	}

	encrypted, err := aesCipher.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	// Pre-allocate result with exact size
	result := make([]byte, 0, len(salt)+len(encrypted))
	result = append(result, salt...)
	result = append(result, encrypted...)

	return result, nil
}

// Decrypt decrypts ciphertext encrypted with PBKDF2 + AES-256-GCM
func (c *PBKDF2Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.saltSize {
		return nil, ErrInvalidCiphertext
	}

	salt := ciphertext[:c.saltSize]
	encryptedData := ciphertext[c.saltSize:]

	key := pbkdf2.Key(c.password, salt, c.iterations, c.keyLen, c.hashFunc)
	defer func() {
		// Clear key from memory after use
		for i := range key {
			key[i] = 0
		}
	}()

	aesCipher, err := NewAESGCM(key)
	if err != nil {
		return nil, err
	}

	return aesCipher.Decrypt(encryptedData)
}

// ============================================================================
// Helper Functions
// ============================================================================

// GenerateKey generates a cryptographically secure random key
func GenerateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateKey256 generates a 256-bit key suitable for AES-256 or ChaCha20
func GenerateKey256() ([]byte, error) {
	return GenerateKey(32)
}
