package cryptox

import (
	"bytes"
	"crypto/sha512"
	"testing"
)

// Test data
var (
	testPlaintext  = []byte("The quick brown fox jumps over the lazy dog. Testing crypto library with some longer content to ensure everything works correctly.")
	testPassword   = []byte("test-password-very-secure-2025!")
	testKey32      = []byte("12345678901234567890123456789012") // 32 bytes
	shortKey       = []byte("too-short")
	emptyPlaintext = []byte{}
)

// ============================================================================
// AES-256-GCM Tests
// ============================================================================

func TestAESGCM_NewWithInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"nil key", nil},
		{"empty key", []byte{}},
		{"short key", shortKey},
		{"long key", append(testKey32, byte('x'))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAESGCM(tt.key)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestAESGCM_EncryptDecrypt(t *testing.T) {
	cipher, err := NewAESGCM(testKey32)
	if err != nil {
		t.Fatalf("NewAESGCM failed: %v", err)
	}

	// Test normal encryption/decryption
	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, testPlaintext) {
		t.Error("encrypted data should not equal plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("decrypted data does not match original plaintext")
	}

	// Test empty plaintext
	encrypted, err = cipher.Encrypt(emptyPlaintext)
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err = cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}

	if !bytes.Equal(decrypted, emptyPlaintext) {
		t.Error("decrypted empty data does not match")
	}
}

func TestAESGCM_NonceRandomness(t *testing.T) {
	cipher, err := NewAESGCM(testKey32)
	if err != nil {
		t.Fatalf("NewAESGCM failed: %v", err)
	}

	encrypted1, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	encrypted2, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("two encryptions of same plaintext should produce different ciphertexts (different nonces)")
	}

	// Both should decrypt to same plaintext
	decrypted1, _ := cipher.Decrypt(encrypted1)
	decrypted2, _ := cipher.Decrypt(encrypted2)

	if !bytes.Equal(decrypted1, decrypted2) || !bytes.Equal(decrypted1, testPlaintext) {
		t.Error("both ciphertexts should decrypt to same plaintext")
	}
}

func TestAESGCM_TamperDetection(t *testing.T) {
	cipher, err := NewAESGCM(testKey32)
	if err != nil {
		t.Fatalf("NewAESGCM failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with ciphertext
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0x01 // Flip last bit

	_, err = cipher.Decrypt(tampered)
	if err == nil {
		t.Error("should detect tampered ciphertext")
	}

	// Test truncated ciphertext
	if len(encrypted) > 20 {
		truncated := encrypted[:20]
		_, err = cipher.Decrypt(truncated)
		if err == nil {
			t.Error("should detect truncated ciphertext")
		}
	}
}

func TestAESGCM_CustomOptions(t *testing.T) {
	cipher, err := NewAESGCM(testKey32,
		WithAESGCMNonceSize(16),
		WithAESGCMTagSize(12))
	if err != nil {
		t.Fatalf("NewAESGCM with options failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("custom options: decryption failed")
	}
}

// ============================================================================
// ChaCha20-Poly1305 Tests
// ============================================================================

func TestChaCha20_EncryptDecrypt(t *testing.T) {
	cipher, err := NewChaCha20(testKey32)
	if err != nil {
		t.Fatalf("NewChaCha20 failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, testPlaintext) {
		t.Error("encrypted data should not equal plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("decrypted data does not match original plaintext")
	}
}

func TestChaCha20_InvalidKey(t *testing.T) {
	_, err := NewChaCha20(shortKey)
	if err == nil {
		t.Error("should reject short key")
	}
}

func TestChaCha20_TamperDetection(t *testing.T) {
	cipher, err := NewChaCha20(testKey32)
	if err != nil {
		t.Fatalf("NewChaCha20 failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with ciphertext
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0x01

	_, err = cipher.Decrypt(tampered)
	if err == nil {
		t.Error("should detect tampered ciphertext")
	}
}

// ============================================================================
// Argon2id Tests
// ============================================================================

func TestArgon2id_EncryptDecrypt(t *testing.T) {
	cipher, err := NewArgon2id(testPassword)
	if err != nil {
		t.Fatalf("NewArgon2id failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, testPlaintext) {
		t.Error("encrypted data should not equal plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("decrypted data does not match original plaintext")
	}
}

func TestArgon2id_EmptyPassword(t *testing.T) {
	_, err := NewArgon2id([]byte{})
	if err == nil {
		t.Error("should reject empty password")
	}
}

func TestArgon2id_DifferentSalts(t *testing.T) {
	cipher, err := NewArgon2id(testPassword)
	if err != nil {
		t.Fatalf("NewArgon2id failed: %v", err)
	}

	encrypted1, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	encrypted2, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("two encryptions should produce different ciphertexts (different salts)")
	}

	// Both should decrypt successfully
	decrypted1, _ := cipher.Decrypt(encrypted1)
	decrypted2, _ := cipher.Decrypt(encrypted2)

	if !bytes.Equal(decrypted1, testPlaintext) || !bytes.Equal(decrypted2, testPlaintext) {
		t.Error("both should decrypt to original plaintext")
	}
}

func TestArgon2id_CustomOptions(t *testing.T) {
	cipher, err := NewArgon2id(testPassword,
		WithArgon2idTime(5),
		WithArgon2idMemory(128*1024),
		WithArgon2idThreads(8),
		WithArgon2idSaltSize(32))
	if err != nil {
		t.Fatalf("NewArgon2id with options failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("custom options: decryption failed")
	}
}

func TestArgon2id_WrongPassword(t *testing.T) {
	cipher1, err := NewArgon2id(testPassword)
	if err != nil {
		t.Fatalf("NewArgon2id failed: %v", err)
	}

	encrypted, err := cipher1.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with wrong password
	wrongPassword := append(testPassword, byte('x'))
	cipher2, err := NewArgon2id(wrongPassword)
	if err != nil {
		t.Fatalf("NewArgon2id with wrong password failed: %v", err)
	}

	_, err = cipher2.Decrypt(encrypted)
	if err == nil {
		t.Error("should fail to decrypt with wrong password")
	}
}

// ============================================================================
// PBKDF2 Tests
// ============================================================================

func TestPBKDF2_EncryptDecrypt(t *testing.T) {
	cipher, err := NewPBKDF2(testPassword)
	if err != nil {
		t.Fatalf("NewPBKDF2 failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, testPlaintext) {
		t.Error("encrypted data should not equal plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("decrypted data does not match original plaintext")
	}
}

func TestPBKDF2_EmptyPassword(t *testing.T) {
	_, err := NewPBKDF2([]byte{})
	if err == nil {
		t.Error("should reject empty password")
	}
}

func TestPBKDF2_CustomOptions(t *testing.T) {
	cipher, err := NewPBKDF2(testPassword,
		WithPBKDF2Iterations(1000000),
		WithPBKDF2SaltSize(32),
		WithPBKDF2Hash(sha512.New))
	if err != nil {
		t.Fatalf("NewPBKDF2 with options failed: %v", err)
	}

	encrypted, err := cipher.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, testPlaintext) {
		t.Error("custom options: decryption failed")
	}
}

func TestPBKDF2_WrongPassword(t *testing.T) {
	cipher1, err := NewPBKDF2(testPassword)
	if err != nil {
		t.Fatalf("NewPBKDF2 failed: %v", err)
	}

	encrypted, err := cipher1.Encrypt(testPlaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with wrong password
	wrongPassword := []byte("wrong-password")
	cipher2, err := NewPBKDF2(wrongPassword)
	if err != nil {
		t.Fatalf("NewPBKDF2 with wrong password failed: %v", err)
	}

	_, err = cipher2.Decrypt(encrypted)
	if err == nil {
		t.Error("should fail to decrypt with wrong password")
	}
}

// ============================================================================
// Helper Functions Tests
// ============================================================================

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(key1))
	}

	key2, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("two generated keys should not be equal")
	}
}

func TestGenerateKey256(t *testing.T) {
	key, err := GenerateKey256()
	if err != nil {
		t.Fatalf("GenerateKey256 failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(key))
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkAESGCM_Encrypt_1KB(b *testing.B) {
	cipher, _ := NewAESGCM(testKey32)
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkAESGCM_Decrypt_1KB(b *testing.B) {
	cipher, _ := NewAESGCM(testKey32)
	data := make([]byte, 1024)
	encrypted, _ := cipher.Encrypt(data)

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Decrypt(encrypted)
	}
}

func BenchmarkAESGCM_Encrypt_1MB(b *testing.B) {
	cipher, _ := NewAESGCM(testKey32)
	data := make([]byte, 1024*1024)

	b.ResetTimer()
	b.SetBytes(1024 * 1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkChaCha20_Encrypt_1KB(b *testing.B) {
	cipher, _ := NewChaCha20(testKey32)
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkChaCha20_Decrypt_1KB(b *testing.B) {
	cipher, _ := NewChaCha20(testKey32)
	data := make([]byte, 1024)
	encrypted, _ := cipher.Encrypt(data)

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Decrypt(encrypted)
	}
}

func BenchmarkChaCha20_Encrypt_1MB(b *testing.B) {
	cipher, _ := NewChaCha20(testKey32)
	data := make([]byte, 1024*1024)

	b.ResetTimer()
	b.SetBytes(1024 * 1024)
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkArgon2id_Encrypt_Default(b *testing.B) {
	cipher, _ := NewArgon2id(testPassword)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkArgon2id_Decrypt_Default(b *testing.B) {
	cipher, _ := NewArgon2id(testPassword)
	data := make([]byte, 1024)
	encrypted, _ := cipher.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Decrypt(encrypted)
	}
}

func BenchmarkArgon2id_Encrypt_HighSecurity(b *testing.B) {
	cipher, _ := NewArgon2id(testPassword,
		WithArgon2idTime(5),
		WithArgon2idMemory(128*1024))
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkPBKDF2_Encrypt_Default(b *testing.B) {
	cipher, _ := NewPBKDF2(testPassword)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkPBKDF2_Decrypt_Default(b *testing.B) {
	cipher, _ := NewPBKDF2(testPassword)
	data := make([]byte, 1024)
	encrypted, _ := cipher.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Decrypt(encrypted)
	}
}

func BenchmarkPBKDF2_Encrypt_HighIterations(b *testing.B) {
	cipher, _ := NewPBKDF2(testPassword,
		WithPBKDF2Iterations(1000000))
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.Encrypt(data)
	}
}

func BenchmarkGenerateKey256(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateKey256()
	}
}

// ============================================================================
// Parallel Benchmarks
// ============================================================================

func BenchmarkAESGCM_Parallel(b *testing.B) {
	cipher, _ := NewAESGCM(testKey32)
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			encrypted, _ := cipher.Encrypt(data)
			_, _ = cipher.Decrypt(encrypted)
		}
	})
}

func BenchmarkChaCha20_Parallel(b *testing.B) {
	cipher, _ := NewChaCha20(testKey32)
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			encrypted, _ := cipher.Encrypt(data)
			_, _ = cipher.Decrypt(encrypted)
		}
	})
}
