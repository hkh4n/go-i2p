// Package router_info implements the I2P RouterInfo common data structure
package router_info

import (
	"errors"
	"github.com/go-i2p/go-i2p/lib/util/logger"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"

	. "github.com/go-i2p/go-i2p/lib/common/data"
	. "github.com/go-i2p/go-i2p/lib/common/router_address"
	. "github.com/go-i2p/go-i2p/lib/common/router_identity"
	. "github.com/go-i2p/go-i2p/lib/common/signature"
)

var log = logger.GetLogger()

const ROUTER_INFO_MIN_SIZE = 439

const (
	MIN_GOOD_VERSION = 58
	MAX_GOOD_VERSION = 99
)

/*
[RouterInfo]
Accurate for version 0.9.49

Description
Defines all of the data that a router wants to public for the network to see. The
RouterInfo is one of two structures stored in the network database (the other being
LeaseSet), and is keyed under the SHA256 of the contained RouterIdentity.

Contents
RouterIdentity followed by the Date, when the entry was published

+----+----+----+----+----+----+----+----+
| router_ident                          |
+                                       +
|                                       |
~                                       ~
~                                       ~
|                                       |
+----+----+----+----+----+----+----+----+
| published                             |
+----+----+----+----+----+----+----+----+
|size| RouterAddress 0                  |
+----+                                  +
|                                       |
~                                       ~
~                                       ~
|                                       |
+----+----+----+----+----+----+----+----+
| RouterAddress 1                       |
+                                       +
|                                       |
~                                       ~
~                                       ~
|                                       |
+----+----+----+----+----+----+----+----+
| RouterAddress ($size-1)               |
+                                       +
|                                       |
~                                       ~
~                                       ~
|                                       |
+----+----+----+----+-//-+----+----+----+
|psiz| options                          |
+----+----+----+----+-//-+----+----+----+
| signature                             |
+                                       +
|                                       |
+                                       +
|                                       |
+                                       +
|                                       |
+                                       +
|                                       |
+----+----+----+----+----+----+----+----+

router_ident :: RouterIdentity
                length -> >= 387 bytes

published :: Date
             length -> 8 bytes

size :: Integer
        length -> 1 byte
        The number of RouterAddresses to follow, 0-255

addresses :: [RouterAddress]
             length -> varies

peer_size :: Integer
             length -> 1 byte
             The number of peer Hashes to follow, 0-255, unused, always zero
             value -> 0

options :: Mapping

signature :: Signature
             length -> 40 bytes
*/

// RouterInfo is the represenation of an I2P RouterInfo.
//
// https://geti2p.net/spec/common-structures#routerinfo
type RouterInfo struct {
	router_identity RouterIdentity
	published       *Date
	size            *Integer
	addresses       []*RouterAddress
	peer_size       *Integer
	options         *Mapping
	signature       *Signature
}

// Bytes returns the RouterInfo as a []byte suitable for writing to a stream.
func (router_info RouterInfo) Bytes() (bytes []byte, err error) {
	log.Debug("Converting RouterInfo to bytes")
	bytes = append(bytes, router_info.router_identity.KeysAndCert.Bytes()...)
	bytes = append(bytes, router_info.published.Bytes()...)
	bytes = append(bytes, router_info.size.Bytes()...)
	for _, router_address := range router_info.addresses {
		bytes = append(bytes, router_address.Bytes()...)
	}
	bytes = append(bytes, router_info.peer_size.Bytes()...)
	bytes = append(bytes, router_info.options.Data()...)
	bytes = append(bytes, []byte(*router_info.signature)...)
	log.WithField("bytes_length", len(bytes)).Debug("Converted RouterInfo to bytes")
	return bytes, err
}

func (router_info RouterInfo) String() string {
	log.Debug("Converting RouterInfo to string")
	str := "Certificate: " + string(router_info.router_identity.KeysAndCert.Bytes())
	str += "Published: " + string(router_info.published.Bytes())
	str += "Addresses:" + string(router_info.size.Bytes())
	for index, router_address := range router_info.addresses {
		str += "Address " + strconv.Itoa(index) + ": " + router_address.String()
	}
	str += "Peer Size: " + string(router_info.peer_size.Bytes())
	str += "Options: " + string(router_info.options.Data())
	str += "Signature: " + string([]byte(*router_info.signature))
	log.WithField("string_length", len(str)).Debug("Converted RouterInfo to string")
	return str
}

// RouterIdentity returns the router identity as *RouterIdentity.
func (router_info *RouterInfo) RouterIdentity() *RouterIdentity {
	return &router_info.router_identity
}

// IndentHash returns the identity hash (sha256 sum) for this RouterInfo.
func (router_info *RouterInfo) IdentHash() Hash {
	log.Debug("Calculating IdentHash for RouterInfo")
	data, _ := router_info.RouterIdentity().KeyCertificate.Data()
	hash := HashData(data)
	log.WithField("hash", hash).Debug("Calculated IdentHash for RouterInfo")
	return HashData(data)
}

// Published returns the date this RouterInfo was published as an I2P Date.
func (router_info *RouterInfo) Published() *Date {
	return router_info.published
}

// RouterAddressCount returns the count of RouterAddress in this RouterInfo as a Go integer.
func (router_info *RouterInfo) RouterAddressCount() int {
	count := router_info.size.Int()
	log.WithField("count", count).Debug("Retrieved RouterAddressCount from RouterInfo")
	return count
}

// RouterAddresses returns all RouterAddresses for this RouterInfo as []*RouterAddress.
func (router_info *RouterInfo) RouterAddresses() []*RouterAddress {
	log.WithField("address_count", len(router_info.addresses)).Debug("Retrieved RouterAddresses from RouterInfo")
	return router_info.addresses
}

// PeerSize returns the peer size as a Go integer.
func (router_info *RouterInfo) PeerSize() int {
	// Peer size is unused:
	// https://geti2p.net/spec/common-structures#routeraddress
	return 0
}

// Options returns the options for this RouterInfo as an I2P Mapping.
func (router_info RouterInfo) Options() (mapping Mapping) {
	return *router_info.options
}

// Signature returns the signature for this RouterInfo as an I2P Signature.
func (router_info RouterInfo) Signature() (signature Signature) {
	return *router_info.signature
}

// Network implements net.Addr
func (router_info RouterInfo) Network() string {
	return "i2p"
}

// ReadRouterInfo returns RouterInfo from a []byte.
// The remaining bytes after the specified length are also returned.
// Returns a list of errors that occurred during parsing.
func ReadRouterInfo(bytes []byte) (info RouterInfo, remainder []byte, err error) {
	log.WithField("input_length", len(bytes)).Debug("Reading RouterInfo from bytes")

	info.router_identity, remainder, err = ReadRouterIdentity(bytes)
	if err != nil {
		log.WithFields(logrus.Fields{
			"at":           "(RouterInfo) ReadRouterInfo",
			"data_len":     len(bytes),
			"required_len": ROUTER_INFO_MIN_SIZE,
			"reason":       "not enough data",
		}).Error("error parsing router info")
		err = errors.New("error parsing router info: not enough data")
		return
	}
	info.published, remainder, err = NewDate(remainder)
	if err != nil {
		log.WithFields(logrus.Fields{
			"at":           "(RouterInfo) ReadRouterInfo",
			"data_len":     len(remainder),
			"required_len": DATE_SIZE,
			"reason":       "not enough data",
		}).Error("error parsing router info")
		err = errors.New("error parsing router info: not enough data")
	}
	info.size, remainder, err = NewInteger(remainder, 1)
	if err != nil {
		log.WithFields(logrus.Fields{
			"at":           "(RouterInfo) ReadRouterInfo",
			"data_len":     len(remainder),
			"required_len": info.size.Int(),
			"reason":       "read error",
		}).Error("error parsing router info size")
	}
	for i := 0; i < info.size.Int(); i++ {
		address, more, err := ReadRouterAddress(remainder)
		remainder = more
		if err != nil {
			log.WithFields(logrus.Fields{
				"at":       "(RouterInfo) ReadRouterInfo",
				"data_len": len(remainder),
				//"required_len": ROUTER_ADDRESS_SIZE,
				"reason": "not enough data",
			}).Error("error parsing router address")
			err = errors.New("error parsing router info: not enough data")
		}
		info.addresses = append(info.addresses, &address)
	}
	info.peer_size, remainder, err = NewInteger(remainder, 1)
	if err != nil {
		log.WithError(err).Error("Failed to read PeerSize")
		return
	}
	var errs []error
	info.options, remainder, errs = NewMapping(remainder)
	if len(errs) != 0 {
		log.WithFields(logrus.Fields{
			"at":       "(RouterInfo) ReadRouterInfo",
			"data_len": len(remainder),
			//"required_len": MAPPING_SIZE,
			"reason": "not enough data",
		}).Error("error parsing router info")
		estring := ""
		for _, e := range errs {
			estring += e.Error() + " "
		}
		err = errors.New("error parsing router info: " + estring)
	}
	info.signature, remainder, err = NewSignature(remainder)
	if err != nil {
		log.WithFields(logrus.Fields{
			"at":       "(RouterInfo) ReadRouterInfo",
			"data_len": len(remainder),
			//"required_len": MAPPING_SIZE,
			"reason": "not enough data",
		}).Error("error parsing router info")
		err = errors.New("error parsing router info: not enough data")
	}

	log.WithFields(logrus.Fields{
		"router_identity":  info.router_identity,
		"published":        info.published,
		"address_count":    len(info.addresses),
		"remainder_length": len(remainder),
	}).Debug("Successfully read RouterInfo")

	return
}

func (router_info *RouterInfo) RouterCapabilities() string {
	log.Debug("Retrieving RouterCapabilities")
	str, err := ToI2PString("caps")
	if err != nil {
		log.WithError(err).Error("Failed to create I2PString for 'caps'")
		return ""
	}
	//return string(router_info.options.Values().Get(str))
	caps := string(router_info.options.Values().Get(str))
	log.WithField("capabilities", caps).Debug("Retrieved RouterCapabilities")
	return caps
}

func (router_info *RouterInfo) RouterVersion() string {
	log.Debug("Retrieving RouterVersion")
	str, err := ToI2PString("router.version")
	if err != nil {
		log.WithError(err).Error("Failed to create I2PString for 'router.version'")
		return ""
	}
	//return string(router_info.options.Values().Get(str))
	version := string(router_info.options.Values().Get(str))
	log.WithField("version", version).Debug("Retrieved RouterVersion")
	return version
}

func (router_info *RouterInfo) GoodVersion() bool {
	log.Debug("Checking if RouterVersion is good")
	version := router_info.RouterVersion()
	v := strings.Split(version, ".")
	if len(v) != 3 {
		log.WithField("version", version).Warn("Invalid version format")
		return false
	}
	if v[0] == "0" {
		if v[1] == "9" {
			val, _ := strconv.Atoi(v[2])
			if val >= MIN_GOOD_VERSION && val <= MAX_GOOD_VERSION {
				return true
			}
		}
	}
	log.WithField("version", version).Warn("Version not in good range")
	return false
}

func (router_info *RouterInfo) UnCongested() bool {
	log.Debug("Checking if RouterInfo is uncongested")
	caps := router_info.RouterCapabilities()
	if strings.Contains(caps, "K") {
		log.WithField("reason", "K capability").Warn("RouterInfo is congested")
		return false
	}
	if strings.Contains(caps, "G") {
		log.WithField("reason", "G capability").Warn("RouterInfo is congested")
		return false
	}
	if strings.Contains(caps, "E") {
		log.WithField("reason", "E capability").Warn("RouterInfo is congested")
		return false
	}
	log.Debug("RouterInfo is uncongested")
	return true
}

func (router_info *RouterInfo) Reachable() bool {
	log.Debug("Checking if RouterInfo is reachable")
	caps := router_info.RouterCapabilities()
	if strings.Contains(caps, "U") {
		log.WithField("reason", "U capability").Debug("RouterInfo is unreachable")
		return false
	}
	//return strings.Contains(caps, "R")
	reachable := strings.Contains(caps, "R")
	log.WithFields(logrus.Fields{
		"reachable": reachable,
		"reason":    "R capability",
	}).Debug("Checked RouterInfo reachability")
	return reachable
}
