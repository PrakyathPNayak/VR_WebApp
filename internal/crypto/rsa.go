package crypto

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/base64"
    "encoding/pem"
    "fmt"
)

var (
    privateKey   *rsa.PrivateKey
    publicKeyPEM string
)

func InitializeRSA() error {
    key, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return fmt.Errorf("failed to generate RSA key: %w", err)
    }
    privateKey = key
    
    pubASN1, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
    if err != nil {
        return fmt.Errorf("failed to marshal public key: %w", err)
    }
    
    pubPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: pubASN1,
    })
    publicKeyPEM = base64.StdEncoding.EncodeToString(pubPEM)
    return nil
}

func GetPublicKeyPEM() string {
    return publicKeyPEM
}

func DecryptAESKey(encryptedKey string) ([]byte, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
    if err != nil {
        return nil, fmt.Errorf("failed to decode encrypted AES key: %w", err)
    }

    decryptedKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
    if err != nil {
        return nil, fmt.Errorf("RSA decryption error: %w", err)
    }

    actualKey, err := base64.StdEncoding.DecodeString(string(decryptedKey))
    if err != nil {
        return nil, fmt.Errorf("failed to decode base64 AES key: %w", err)
    }

    if len(actualKey) != 32 { // AES-256
        return nil, fmt.Errorf("invalid AES key length: %d", len(actualKey))
    }

    return actualKey, nil
}