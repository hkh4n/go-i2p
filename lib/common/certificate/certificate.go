// Package certificate implements the certificate common-structure of I2P.

package certificate

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"

	// log "github.com/sirupsen/logrus"
	"github.com/go-i2p/go-i2p/lib/util/logger"

	. "github.com/go-i2p/go-i2p/lib/common/data"
)

var log = logger.GetGoI2PLogger()

// Certificate Types
const (
	CERT_NULL = iota
	CERT_HASHCASH
	CERT_HIDDEN
	CERT_SIGNED
	CERT_MULTIPLE
	CERT_KEY
)

// CERT_MIN_SIZE is the minimum size of a valid Certificate in []byte
// 1 byte for type
// 2 bytes for payload length
const CERT_MIN_SIZE = 3

/*
[I2P Certificate]
Accurate for version 0.9.49

Description
A certifificate is a container for various receipts of proof of works used throughout the I2P network.

Contents
1 byte Integer specifying certificate type, followed by a 2 byte Integer specifying the size of the certificate playload, then that many bytes.

+----+----+----+----+----+-//
|type| length  | payload
+----+----+----+----+----+-//

type :: Integer
        length -> 1 byte

        case 0 -> NULL
        case 1 -> HASHCASH
        case 2 -> HIDDEN
        case 3 -> SIGNED
        case 4 -> MULTIPLE
        case 5 -> KEY

length :: Integer
          length -> 2 bytes

payload :: data
           length -> $length bytes
*/

// Certificate is the representation of an I2P Certificate.
//
// https://geti2p.net/spec/common-structures#certificate
type Certificate struct {
	kind    Integer
	len     Integer
	payload []byte
}

// RawBytes returns the entire certificate in []byte form, includes excess payload data.
func (c *Certificate) RawBytes() []byte {
	bytes := c.kind.Bytes()
	bytes = append(bytes, c.len.Bytes()...)
	bytes = append(bytes, c.payload...)
	log.WithFields(logrus.Fields{
		"raw_bytes_length": len(bytes),
	}).Debug("Generated raw bytes for certificate")
	return bytes
}

// ExcessBytes returns the excess bytes in a certificate found after the specified payload length.
func (c *Certificate) ExcessBytes() []byte {
	if len(c.payload) >= c.len.Int() {
		excess := c.payload[c.len.Int():]
		log.WithFields(logrus.Fields{
			"excess_bytes_length": len(excess),
		}).Debug("Found excess bytes in certificate")
		return excess
	}
	log.Debug("No excess bytes found in certificate")
	return nil
}

// Bytes returns the entire certificate in []byte form, trims payload to specified length.
func (c *Certificate) Bytes() []byte {
	bytes := c.kind.Bytes()
	bytes = append(bytes, c.len.Bytes()...)
	bytes = append(bytes, c.Data()...)
	log.WithFields(logrus.Fields{
		"bytes_length": len(bytes),
	}).Debug("Generated bytes for certificate")
	return bytes
}

func (c *Certificate) length() (cert_len int) {
	cert_len = len(c.Bytes())
	return
}

// Type returns the Certificate type specified in the first byte of the Certificate,
func (c *Certificate) Type() (cert_type int) {
	cert_type = c.kind.Int()
	log.WithFields(logrus.Fields{
		"cert_type": cert_type,
	}).Debug("Retrieved certificate type")
	return
}

// Length returns the payload length of a Certificate.
func (c *Certificate) Length() (length int) {
	length = c.len.Int()
	log.WithFields(logrus.Fields{
		"length": length,
	}).Debug("Retrieved certificate length")
	return
}

// Data returns the payload of a Certificate, payload is trimmed to the specified length.
func (c *Certificate) Data() (data []byte) {
	lastElement := c.Length()
	if lastElement > len(c.payload) {
		data = c.payload
		log.Warn("Certificate payload shorter than specified length")
	} else {
		data = c.payload[0:lastElement]
	}
	log.WithFields(logrus.Fields{
		"data_length": len(data),
	}).Debug("Retrieved certificate data")
	return
}

// readCertificate creates a new Certficiate from []byte
// returns err if the certificate is too short or if the payload doesn't match specified length.
func readCertificate(data []byte) (certificate Certificate, err error) {
	certificate = Certificate{}
	switch len(data) {
	case 0:
		certificate.kind = Integer([]byte{0})
		certificate.len = Integer([]byte{0})
		log.WithFields(logrus.Fields{
			"at":                       "(Certificate) NewCertificate",
			"certificate_bytes_length": len(data),
			"reason":                   "too short (len < CERT_MIN_SIZE)" + fmt.Sprintf("%d", certificate.kind.Int()),
		}).Error("invalid certificate, empty")
		err = fmt.Errorf("error parsing certificate: certificate is empty")
		return
	case 1, 2:
		certificate.kind = Integer(data[0 : len(data)-1])
		certificate.len = Integer([]byte{0})
		log.WithFields(logrus.Fields{
			"at":                       "(Certificate) NewCertificate",
			"certificate_bytes_length": len(data),
			"reason":                   "too short (len < CERT_MIN_SIZE)" + fmt.Sprintf("%d", certificate.kind.Int()),
		}).Error("invalid certificate, too short")
		err = fmt.Errorf("error parsing certificate: certificate is too short")
		return
	default:
		certificate.kind = Integer(data[0:1])
		certificate.len = Integer(data[1:3])
		payloadLength := len(data) - CERT_MIN_SIZE
		certificate.payload = data[CERT_MIN_SIZE:]
		if certificate.len.Int() > len(data)-CERT_MIN_SIZE {
			err = fmt.Errorf("certificate parsing warning: certificate data is shorter than specified by length")
			log.WithFields(logrus.Fields{
				"at":                         "(Certificate) NewCertificate",
				"certificate_bytes_length":   certificate.len.Int(),
				"certificate_payload_length": payloadLength,
				"data_bytes:":                string(data),
				"kind_bytes":                 data[0:1],
				"len_bytes":                  data[1:3],
				"reason":                     err.Error(),
			}).Error("invalid certificate, shorter than specified by length")
			return
		}
		log.WithFields(logrus.Fields{
			"type":   certificate.kind.Int(),
			"length": certificate.len.Int(),
		}).Debug("Successfully created new certificate")
		return
	}
}

// ReadCertificate creates a Certificate from []byte and returns any ExcessBytes at the end of the input.
// returns err if the certificate could not be read.
func ReadCertificate(data []byte) (certificate Certificate, remainder []byte, err error) {
	certificate, err = readCertificate(data)
	if err != nil && err.Error() == "certificate parsing warning: certificate data is longer than specified by length" {
		log.Warn("Certificate data longer than specified length")
		err = nil
	}
	remainder = certificate.ExcessBytes()
	log.WithFields(logrus.Fields{
		"remainder_length": len(remainder),
	}).Debug("Read certificate and extracted remainder")
	return
}

// NewCertificate creates a new Certificate with default NULL type
func NewCertificate() *Certificate {
	return &Certificate{
		kind:    Integer([]byte{CERT_NULL}),
		len:     Integer([]byte{0}),
		payload: make([]byte, 0),
	}
}

// NewCertificateWithType creates a new Certificate with specified type and payload
func NewCertificateWithType(certType uint8, payload []byte) (*Certificate, error) {
	// Validate certificate type
	switch certType {
	case CERT_NULL, CERT_HASHCASH, CERT_HIDDEN, CERT_SIGNED, CERT_MULTIPLE, CERT_KEY:
		// Valid type
	default:
		return nil, fmt.Errorf("invalid certificate type: %d", certType)
	}

	// For NULL certificates, payload should be empty
	if certType == CERT_NULL && len(payload) > 0 {
		return nil, errors.New("NULL certificates must have empty payload")
	}
	length, _ := NewIntegerFromInt(len(payload), 2)

	cert := &Certificate{
		kind:    Integer([]byte{certType}),
		len:     *length,
		payload: make([]byte, len(payload)),
	}

	// Copy payload if present
	if len(payload) > 0 {
		copy(cert.payload, payload)
	}

	return cert, nil
}

func GetSignatureTypeFromCertificate(cert Certificate) (int, error) {
	if cert.Type() != CERT_KEY {
		return 0, fmt.Errorf("unexpected certificate type: %d", cert.Type())
	}
	if len(cert.payload) < 2 {
		return 0, fmt.Errorf("certificate payload too short to contain signature type")
	}
	sigType := int(binary.BigEndian.Uint16(cert.payload[0:2]))
	return sigType, nil
}
