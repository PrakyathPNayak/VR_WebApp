package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "encoding/base64"
    "fmt"
)

type AESCipher struct {
    gcm cipher.AEAD
}

func NewAESCipher(key []byte) (*AESCipher, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create AES cipher: %w", err)
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }
    
    return &AESCipher{gcm: gcm}, nil
}

func (a *AESCipher) Decrypt(encryptedData string) ([]byte, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
    if err != nil {
        return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
    }
    
    nonceSize := a.gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }
    
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }
    
    return plaintext, nil
}

func (a *AESCipher) DecryptBinary(data []byte) ([]byte, error) {
    nonceSize := a.gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("binary data too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt binary data: %w", err)
    }
    
    return plaintext, nil
}