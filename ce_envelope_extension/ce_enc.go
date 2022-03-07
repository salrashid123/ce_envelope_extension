package extensions

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/core/registry"
	"github.com/google/tink/go/integration/gcpkms"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"

	// GCP
	cloudkms "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

type EncType int

const (
	TINK EncType = iota
	TPM
	SHARED
	KMS
)

type EncryptionExtension struct {
	KeyUri string  `json:"key_uri,omitempty"`
	DEK    string  `json:"dek"`
	Type   EncType `json:"type"`
	a      tink.AEAD
	b      cipher.AEAD
}

const (
	EncryptionExtensionName = "envelopeencryption"
)

var ()

func NewEncryptionExtension(conf *EncryptionExtension) (*EncryptionExtension, error) {

	if conf.Type == TINK {

		var kh1 *keyset.Handle
		var err error

		gcpClient, err := gcpkms.NewClient("gcp-kms://")
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could not create TINK KMS Client %v", err)
		}

		registry.RegisterKMSClient(gcpClient)
		backend, err := gcpClient.GetAEAD(conf.KeyUri)
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
		}

		memKeyset := &keyset.MemReaderWriter{}

		if conf.DEK == "" {

			kh1, err = keyset.NewHandle(aead.AES256GCMKeyTemplate())
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not create TINK keyHandle %v", err)
			}

			if err := kh1.Write(memKeyset, backend); err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not serialize KeyHandle  %v", err)
			}

			buf := new(bytes.Buffer)
			w := keyset.NewJSONWriter(buf)
			if err := w.WriteEncrypted(memKeyset.EncryptedKeyset); err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not write encrypted keyhandle %v", err)
			}

			conf.DEK = string(buf.Bytes())

		} else {

			buf := new(bytes.Buffer)

			_, err := buf.WriteString(conf.DEK)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not Unmarshal %v", err)
			}

			r := keyset.NewJSONReader(buf)

			kh1, err = keyset.Read(r, backend)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not create TINK KMS Client %v", err)
			}
		}

		a, err := aead.New(kh1)
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could not create TINK AEAD  %v", err)
		}
		conf.a = a
	}
	if conf.Type == KMS {

		ctx := context.Background()
		kmsClient, err := cloudkms.NewKeyManagementClient(ctx)
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could create KMS Client %v", err)
		}

		if conf.DEK == "" {
			key := make([]byte, 32)
			if _, err := io.ReadFull(rand.Reader, key); err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
			}
			block, err := aes.NewCipher(key)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not create new Cipher %v", err)
			}

			conf.b, err = cipher.NewGCM(block)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
			}

			req := &kmspb.EncryptRequest{
				Name:      conf.KeyUri,
				Plaintext: key,
			}

			result, err := kmsClient.Encrypt(ctx, req)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
			}
			conf.DEK = base64.StdEncoding.EncodeToString(result.Ciphertext)

		} else {

			dd, err := base64.StdEncoding.DecodeString(conf.DEK)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not decode DEK %v", err)
			}

			req := &kmspb.DecryptRequest{
				Name:       conf.KeyUri,
				Ciphertext: dd,
			}

			result, err := kmsClient.Decrypt(ctx, req)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
			}

			block, err := aes.NewCipher(result.Plaintext)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not create new Cipher %v", err)
			}

			conf.b, err = cipher.NewGCM(block)
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not acquire KMS AEAD %v", err)
			}

		}

	}

	if conf.Type == SHARED {
		if conf.DEK == "" {
			return &EncryptionExtension{}, fmt.Errorf("DEK Cannot be null")
		}
		block, err := aes.NewCipher([]byte(conf.DEK))
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could not acquire create shared AES Cipher %v", err)
		}

		conf.b, err = cipher.NewGCM(block)
		if err != nil {
			return &EncryptionExtension{}, fmt.Errorf("Could not acquire create shared AES Cipher %v", err)
		}

		// note, this is just a lazy hack
		//  we're resetting the DEK value here with the hash
		//  after we've already initialized it (i.,e conf.b already holds the cipher.AEAD with the raw key)
		//  the only reason why we do this is to allow serializing EncryptionExtension and sending
		//  the sha256 form of the key
		h := sha256.New()
		h.Write([]byte(conf.DEK))
		conf.DEK = fmt.Sprintf("%x", h.Sum(nil))
	}

	return conf, nil
}

func (d *EncryptionExtension) GetType() *EncryptionExtension {
	return &EncryptionExtension{
		KeyUri: d.KeyUri,
		DEK:    d.DEK,
		Type:   d.Type,
	}
}

func (d *EncryptionExtension) Encrypt(raw []byte, aad []byte) (encrypted []byte, err error) {
	var ct []byte
	if d.Type == TINK {
		ct, err = d.a.Encrypt(raw, []byte(d.KeyUri))
		if err != nil {
			return []byte(""), fmt.Errorf("Could not encrypt data %v", err)
		}
	}
	if d.Type == KMS || d.Type == SHARED {
		nonce := make([]byte, d.b.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return []byte(""), fmt.Errorf("Could not encrypt data %v", err)
		}
		ct = d.b.Seal(nonce, nonce, raw, aad)

	}
	return ct, nil
}

func (d *EncryptionExtension) Decrypt(raw []byte, aad []byte) (decrypted []byte, err error) {
	var ct []byte
	if d.Type == TINK {
		ct, err = d.a.Decrypt(raw, []byte(d.KeyUri))
		if err != nil {
			return []byte(""), fmt.Errorf("Could not decrypt data %v", err)
		}
	}
	if d.Type == KMS || d.Type == SHARED {
		nonceSize := d.b.NonceSize()
		nonce, raw := raw[:nonceSize], raw[nonceSize:]
		ct, err = d.b.Open(nil, nonce, raw, aad)
		if err != nil {
			return []byte(""), fmt.Errorf("Could not decrypt data %v", err)
		}

	}
	return ct, nil
}
