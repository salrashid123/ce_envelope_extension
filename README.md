
## Cloud Events envelope encryption extension
     (for Pubsub)


Sample [Cloud Events](https://cloudevents.io/) extension that describes encrypted payloads using TINK Envelope Encryption.

Cloud [Events Extensions](https://github.com/cloudevents/spec/blob/v1.0/documented-extensions.md) basically just describes metadata to include into headers of any given message.

This particular extension first encrypts the `Data` area of a PubSub message using a KMS backed TINK Symmetric key.  Once encrypted, it places the key reference value and the encrypted Data Encryption Key (DEK) as the metadata value for transmission.

The subscriber of the event will decode the extension and then use the encrypted key to finally decode the message.

Since the extension framework is just metadata and not an 'interceptor' for pre/post processing messages, the decryption has to be done in code manually...

This sample sets up a pubsub topic with a subscriber as well as a symmetric GCP KMS key to use for encryption.

On startup, the pubsub publisher will generate 10 messages but only rotate the derived AES DEK key three times.  This is done to avoid repeated calls to kms to rewrap the DEK (meaning, you are using the DEK a couple of times before its rotated out). The DEK and TINK ke gets embedded into the cloud event payload

A sample event log is shown below:

```log
$ go run main.go --mode publish --projectID $PROJECT_ID -topicID crypt-topic -v 20 -alsologtostderr
I1103 09:17:33.394934  847093 main.go:162] Generating New Key
I1103 09:17:33.741124  847093 main.go:177]      DEK sha256 [579c0b87ff47601b61788f78815286ddd12d6ba7d6d65d9fcd45ac625cc71c9b]
I1103 09:17:33.742634  847093 main.go:192]      Encrypting data: [50eb0bbe-1ddf-11eb-b86d-e86a641d5560]
I1103 09:17:34.028597  847093 main.go:205] Validation: valid
Context Attributes,
  specversion: 1.0
  type: com.google.cloud.pubsub.topic.publish
  source: github.com/salrashid123/tink_samples
  id: 1f748398-7622-4588-9246-8505b7f9f134
  datacontenttype: application/json
Extensions,
  envelopeencryption: {"key_uri":"gcp-kms://projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1","dek":"{\"encryptedKeyset\":\"AAAAcwokAFY56F3+gzvVlIQ1iS2v0G0CMA3ezmyXA4sncHVD9UynCXdPEksAcQ7QQsQf8Z9tWC7V+mBM7ELpKuzCrbaupdbogVOG25dhosiSSg4xNVMuxTGLEakMzZhe33V3ChuVYcVnH1c3awo7VjbhrKjE0zNnoY1NLLksmM1iexty0gAsWtM4M0vlkMx80/gXXvMdireYHfsnByxjM23OURn8PJv8y2jPOkt9M648WLcJa6Ad13I+VSrkcan2H9wMrsUBK4OYBhMBsHwY+m21GVyhZffViK/+EpTTj/v2UPr6xdcz62jkaM04ht5kT6yum4J9OE2ogSw9g9wmgd/yVBpMIO7LQqYI0QTAFR9K7B3EuNGbuud2OV7GGxVHLTbf/DgxW8cBjrVNsi2NBoVc38zuUSbKAANU1tqz5DPbpMKxeV6HvqP9GW+6wUxP2IwF0sdgm8j8hlrfQCHKCQD84KUQiC3d/aHvXrLzmdKOfib8B2pNCQ7fBr/dmIod1AS2v+HhUQ==\",\"keysetInfo\":{\"primaryKeyId\":3630674293,\"keyInfo\":[{\"typeUrl\":\"type.googleapis.com/google.crypto.tink.KmsEnvelopeAeadKey\",\"status\":\"ENABLED\",\"keyId\":3630674293,\"outputPrefixType\":\"TINK\"}]}}","type":0}
Data,
  {
    "data": "AdhnsXUAAABzCiQAVjnoXVSPiDx6zMzWVv7K/ZY6/x1sKbYF30F6AujKsoWGCFQSSwBxDtBCJCQ6sTvEC22mTd3whVNRiMyY5mzx7XpZ+Rf9O5k/5iYwvR9kiR+/UfVE62ySnW9O11/x0L3uMaYomFoMm3DgeaeJNRDfG/yGQc/KT3k5Sdt+cefULibQm7ljCyD5czvvXVktrPht1Bvm12Rm4QxreBQZqLPpJUHwJoJ6c+qplpGw75gMJRY="
  }
```

Notice the `Extensions -> envelopencryption` section.  That is the metadata that gets sent with each data. The `encryptedKeyset` is the KMS wrapped key that uses the Key Encryption Key (KEK) defined by the `key_uri`.  The inner, encrypted KEK is used to encrypt the `Data` section.

On the subscriber side, the  KEK gets cached locally to avoid repeated lookups.  If the subscriber does not find the KEK in cache,it will use TINK to do envelope decryption on the Data and then save the KEK.  If the keys are rotated by the publisher, no problem,it will detect that the KEK is not in cache and continue.


>> Note about KMS operations: the fact we're using TINK Encrypted KMS backed keys, each operation invokes the KMS api.  If you would rather just use KMS to wrap a secret, pass the reference to KMS and wrap the key manually and rotate as described here: - [Message Payload Encryption in Google Cloud Pub/Sub (Part 4: Envelope Encryption with Google Key Management System and PubSub)](https://github.com/salrashid123/gcp_pubsub_message_encryption/tree/master/4_kms_dek)



for more info, see

- [Tink Samples](https://github.com/salrashid123/tink_samples)
- [Message Payload Encryption in Google Cloud Pub/Sub (Part 4: Envelope Encryption with Google Key Management System and PubSub)](https://github.com/salrashid123/gcp_pubsub_message_encryption/tree/master/4_kms_dek)


### Setup

First setup KMS, PubSub

```bash
# in each window
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`


gcloud pubsub topics create crypt-topic

gcloud pubsub subscriptions create crypt-subscribe --topic=crypt-topic

gcloud kms keyrings create pubsub-kr --location us-central1

gcloud kms keys  create key1 --keyring pubsub-kr --location us-central1 --purpose encryption 
```

Then just run the client and server

The following will show the key rotation frequency and the hash value of the DEK used. Thats just shown so that you know which key is being used for encryption.


- Publisher

```log
$ go run main.go --mode publish --projectID $PROJECT_ID -topicID crypt-topic -v 10 -alsologtostderr
          I1103 09:14:30.195459  846494 main.go:162] Generating New Key
          I1103 09:14:30.525427  846494 main.go:177]      DEK sha256 [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:30.526103  846494 main.go:192]      Encrypting data: [e3b66cd8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:32.694532  846494 main.go:172] Using Existing Key
          I1103 09:14:32.694642  846494 main.go:177]      DEK sha256 [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:32.694768  846494 main.go:192]      Encrypting data: [e50156e0-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:34.065773  846494 main.go:172] Using Existing Key
          I1103 09:14:34.065826  846494 main.go:177]      DEK sha256 [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:34.065884  846494 main.go:192]      Encrypting data: [e5d28e71-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:35.420385  846494 main.go:162] Generating New Key
          I1103 09:14:35.839211  846494 main.go:177]      DEK sha256 [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:35.839328  846494 main.go:192]      Encrypting data: [e6e12938-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:37.298530  846494 main.go:172] Using Existing Key
          I1103 09:14:37.298606  846494 main.go:177]      DEK sha256 [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:37.298745  846494 main.go:192]      Encrypting data: [e7bfd9b8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:38.493969  846494 main.go:172] Using Existing Key
          I1103 09:14:38.494000  846494 main.go:177]      DEK sha256 [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:38.494055  846494 main.go:192]      Encrypting data: [e8763de3-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:39.728670  846494 main.go:172] Using Existing Key
          I1103 09:14:39.728802  846494 main.go:177]      DEK sha256 [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:39.728936  846494 main.go:192]      Encrypting data: [e932ab10-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:41.169441  846494 main.go:162] Generating New Key
          I1103 09:14:41.661590  846494 main.go:177]      DEK sha256 [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:41.661707  846494 main.go:192]      Encrypting data: [ea5995e8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:43.138270  846494 main.go:172] Using Existing Key
          I1103 09:14:43.138375  846494 main.go:177]      DEK sha256 [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:43.138486  846494 main.go:192]      Encrypting data: [eb3aeca4-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:44.368086  846494 main.go:172] Using Existing Key
          I1103 09:14:44.368191  846494 main.go:177]      DEK sha256 [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:44.368316  846494 main.go:192]      Encrypting data: [ebf694d2-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:45.649285  846494 main.go:172] Using Existing Key
          I1103 09:14:45.649401  846494 main.go:177]      DEK sha256 [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:45.649523  846494 main.go:192]      Encrypting data: [ecba13fc-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:47.022698  846494 main.go:162] Generating New Key
          I1103 09:14:47.317502  846494 main.go:177]      DEK sha256 [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:47.317638  846494 main.go:192]      Encrypting data: [edb89cb2-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:48.762523  846494 main.go:172] Using Existing Key
          I1103 09:14:48.762613  846494 main.go:177]      DEK sha256 [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:48.762727  846494 main.go:192]      Encrypting data: [ee951d8b-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:50.133945  846494 main.go:172] Using Existing Key
          I1103 09:14:50.134023  846494 main.go:177]      DEK sha256 [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:50.134134  846494 main.go:192]      Encrypting data: [ef666023-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:51.687705  846494 main.go:172] Using Existing Key
          I1103 09:14:51.687811  846494 main.go:177]      DEK sha256 [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:51.687936  846494 main.go:192]      Encrypting data: [f0537772-1dde-11eb-9ecc-e86a641d5560]
```

- Subscriber

```log
$ go run main.go --mode subscribe --projectID $PROJECT_ID -topicID crypt-topic --subID crypt-subscribe -v 10 -alsologtostderr
          I1103 09:14:32.724995  846405 main.go:85] Initialize new key
          I1103 09:14:33.041328  846405 main.go:95]      DEK sha256 value [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:33.144991  846405 main.go:116] Decrypted Pubsub Message data [e3b66cd8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:34.123495  846405 main.go:82] Using Existing key
          I1103 09:14:34.123576  846405 main.go:95]      DEK sha256 value [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:34.277804  846405 main.go:116] Decrypted Pubsub Message data [e50156e0-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:34.468162  846405 main.go:82] Using Existing key
          I1103 09:14:34.468233  846405 main.go:95]      DEK sha256 value [e0e8dbe99eed292422283e2fec7566e07af81de5a11c2866d94d5fc99429687b]
          I1103 09:14:34.549903  846405 main.go:116] Decrypted Pubsub Message data [e5d28e71-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:37.292885  846405 main.go:85] Initialize new key
          I1103 09:14:37.449531  846405 main.go:95]      DEK sha256 value [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:37.546416  846405 main.go:82] Using Existing key
          I1103 09:14:37.546550  846405 main.go:95]      DEK sha256 value [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:37.555101  846405 main.go:116] Decrypted Pubsub Message data [e6e12938-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:37.638204  846405 main.go:116] Decrypted Pubsub Message data [e7bfd9b8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:38.795286  846405 main.go:82] Using Existing key
          I1103 09:14:38.795360  846405 main.go:95]      DEK sha256 value [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:38.880574  846405 main.go:116] Decrypted Pubsub Message data [e8763de3-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:40.255764  846405 main.go:82] Using Existing key
          I1103 09:14:40.255839  846405 main.go:95]      DEK sha256 value [c80c74de4e7dd6ac1a97644e08e2f677efcef7cea188f0e96cf173f635890c10]
          I1103 09:14:40.517288  846405 main.go:116] Decrypted Pubsub Message data [e932ab10-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:42.194178  846405 main.go:85] Initialize new key
          I1103 09:14:42.750679  846405 main.go:95]      DEK sha256 value [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:42.818931  846405 main.go:116] Decrypted Pubsub Message data [ea5995e8-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:43.408259  846405 main.go:82] Using Existing key
          I1103 09:14:43.408333  846405 main.go:95]      DEK sha256 value [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:43.531738  846405 main.go:116] Decrypted Pubsub Message data [eb3aeca4-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:44.701921  846405 main.go:82] Using Existing key
          I1103 09:14:44.701995  846405 main.go:95]      DEK sha256 value [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:44.941489  846405 main.go:116] Decrypted Pubsub Message data [ebf694d2-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:46.062342  846405 main.go:82] Using Existing key
          I1103 09:14:46.062415  846405 main.go:95]      DEK sha256 value [3e983dbe6291893c7844eec37ef0bc80e12b8e8dd3dbafd64475465950f515c6]
          I1103 09:14:46.138624  846405 main.go:116] Decrypted Pubsub Message data [ecba13fc-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:47.811789  846405 main.go:85] Initialize new key
          I1103 09:14:48.343641  846405 main.go:95]      DEK sha256 value [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:48.614037  846405 main.go:116] Decrypted Pubsub Message data [edb89cb2-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:49.172482  846405 main.go:82] Using Existing key
          I1103 09:14:49.172576  846405 main.go:95]      DEK sha256 value [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:49.306872  846405 main.go:116] Decrypted Pubsub Message data [ee951d8b-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:50.778310  846405 main.go:82] Using Existing key
          I1103 09:14:50.778387  846405 main.go:95]      DEK sha256 value [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:50.836617  846405 main.go:116] Decrypted Pubsub Message data [ef666023-1dde-11eb-9ecc-e86a641d5560]
          I1103 09:14:52.169845  846405 main.go:82] Using Existing key
          I1103 09:14:52.169926  846405 main.go:95]      DEK sha256 value [9abe0a005d1ff68e15123f951866b765e49d28e5c17b4ca1899f15725243ef04]
          I1103 09:14:52.251696  846405 main.go:116] Decrypted Pubsub Message data [f0537772-1dde-11eb-9ecc-e86a641d5560]
```