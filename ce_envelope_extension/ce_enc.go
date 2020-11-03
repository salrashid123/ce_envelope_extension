package extensions

import (
	"bytes"
	"fmt"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/core/registry"
	"github.com/google/tink/go/integration/gcpkms"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

type EncType int

const (
	TINK EncType = iota
	TPM
	SHARED
)

type EncryptionExtension struct {
	KeyUri string  `json:"key_uri"`
	DEK    string  `json:"dek"`
	Type   EncType `json:"type"`
	a      tink.AEAD
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
		masterKey := aead.NewKMSEnvelopeAEAD2(aead.AES256GCMKeyTemplate(), backend)
		memKeyset := &keyset.MemReaderWriter{}

		dek := aead.AES256GCMKeyTemplate()
		if conf.DEK == "" {

			kh1, err = keyset.NewHandle(aead.KMSEnvelopeAEADKeyTemplate(conf.KeyUri, dek))
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not create TINK keyHandle %v", err)
			}

			if err := kh1.Write(memKeyset, masterKey); err != nil {
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
			kse2, err := r.ReadEncrypted()
			if err != nil {
				return &EncryptionExtension{}, fmt.Errorf("Could not Unmarshal %v", err)
			}

			memKeyset.EncryptedKeyset = kse2
			kh1, err = keyset.Read(memKeyset, masterKey)
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
	return conf, nil
}

func (d *EncryptionExtension) GetType() *EncryptionExtension {
	return &EncryptionExtension{
		KeyUri: d.KeyUri,
		DEK:    d.DEK,
		Type:   d.Type,
	}
}

func (d *EncryptionExtension) Encrypt(raw []byte) (encrypted []byte, err error) {
	var ct []byte
	if d.Type == TINK {
		ct, err = d.a.Encrypt(raw, []byte(d.KeyUri))
		if err != nil {
			return []byte(""), fmt.Errorf("Could not encrypt data %v", err)
		}
	}
	return ct, nil
}

func (d *EncryptionExtension) Decrypt(raw []byte) (decrypted []byte, err error) {
	var ct []byte
	if d.Type == TINK {
		ct, err = d.a.Decrypt(raw, []byte(d.KeyUri))
		if err != nil {
			return []byte(""), fmt.Errorf("Could not decrypt data %v", err)
		}
	}
	return ct, nil
}
