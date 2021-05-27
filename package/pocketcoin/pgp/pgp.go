package pgp

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/sha256"
    "crypto"
    "encoding/pem"
    "errors"
    
)


func check(err error) {
    if err != nil {
        panic(err)
    }
}


func GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
    privkey, _ := rsa.GenerateKey(rand.Reader, 4096)
    return privkey, &privkey.PublicKey
}


func ExportPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
    privkey_bytes := x509.MarshalPKCS1PrivateKey(privkey)
    privkey_pem := pem.EncodeToMemory(
            &pem.Block{
                    Type:  "RSA PRIVATE KEY",
                    Bytes: privkey_bytes,
            },
    )
    return string(privkey_pem)
}


func ExportPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, error) {
    pubkey_bytes, err := x509.MarshalPKIXPublicKey(pubkey)
    if err != nil {
            return "", err
    }
    pubkey_pem := pem.EncodeToMemory(
            &pem.Block{
                    Type:  "RSA PUBLIC KEY",
                    Bytes: pubkey_bytes,
            },
    )

    return string(pubkey_pem), nil
}


func ParsePrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
    block, _ := pem.Decode([]byte(privPEM))
    if block == nil {
            return nil, errors.New("failed to parse PEM block containing the key")
    }

    priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
            return nil, err
    }

    return priv, nil
}


func ParsePublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
    block, _ := pem.Decode([]byte(pubPEM))
    if block == nil {
            return nil, errors.New("failed to parse PEM block containing the key")
    }

    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
            return nil, err
    }

    switch pub := pub.(type) {
    case *rsa.PublicKey:
            return pub, nil
    default:
            break // fall through
    }
    return nil, errors.New("Key type is not RSA")
}


func SignMessage(message string, privateKey *rsa.PrivateKey) []uint8 {
    hashedMessage := sha256.Sum256([]byte(message))
    signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashedMessage[:])
    check(err)
    return signature
}


func ValidSignature(message string, signature []uint8, publicKey *rsa.PublicKey) bool {
    hashedMessage := sha256.Sum256([]byte(message))
    err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashedMessage[:], signature)
    if err != nil {
        return false
    }
    return true
}
