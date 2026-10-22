package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/netip"
	"strings"

	"golang.org/x/text/cases"
)

type Address struct {
	Name       string
	Line1      string
	Line2      string
	City       string
	Region     string
	PostalCode string
	Country    string
}

type Fingerprinter struct{ key []byte }

func NewFingerprinter(key string) Fingerprinter { return Fingerprinter{key: []byte(key)} }

func normalize(value string) string {
	return strings.Join(strings.Fields(cases.Fold().String(value)), " ")
}

// The recipient name is deliberately excluded: the fingerprint identifies a place, not a person.
func NormalizeAddress(a Address) string {
	parts := make([]string, 0, 6)
	for _, part := range []string{a.Line1, a.Line2, a.City, a.Region, a.PostalCode, a.Country} {
		if normalized := normalize(part); normalized != "" {
			parts = append(parts, normalized)
		}
	}
	return strings.Join(parts, " ")
}

func NetworkPrefix(ip netip.Addr) (netip.Prefix, bool) {
	ip = ip.WithZone("").Unmap()
	if !ip.IsValid() {
		return netip.Prefix{}, false
	}
	bits := 24
	if ip.Is6() {
		bits = 48
	}
	prefix, err := ip.Prefix(bits)
	if err != nil {
		return netip.Prefix{}, false
	}
	return prefix, true
}

func (f Fingerprinter) sum(value string) string {
	mac := hmac.New(sha256.New, f.key)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func (f Fingerprinter) Address(a Address) string { return f.sum(NormalizeAddress(a)) }

func (f Fingerprinter) Network(ip netip.Addr) (string, bool) {
	prefix, ok := NetworkPrefix(ip)
	if !ok {
		return "", false
	}
	return f.sum(prefix.String()), true
}
