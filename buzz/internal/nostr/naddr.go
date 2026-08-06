package nostr

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
)

const (
	naddrHRP          = "naddr"
	naddrTLVIdent     = 0
	naddrTLVAuthor    = 2
	naddrTLVKind      = 3
	naddrKindByteSize = 4
)

func EncodeNaddr(kind uint32, pubkeyHex, identifier string) (string, error) {
	pubkeyHex = strings.ToLower(strings.TrimSpace(pubkeyHex))
	pubkey, err := hex.DecodeString(pubkeyHex)
	if err != nil {
		return "", fmt.Errorf("decode naddr author pubkey: %w", err)
	}
	if len(pubkey) != 32 {
		return "", fmt.Errorf("naddr author pubkey must be 32 bytes")
	}
	if identifier == "" {
		return "", fmt.Errorf("naddr identifier is required")
	}
	raw := make([]byte, 0, len(identifier)+len(pubkey)+naddrKindByteSize+6)
	raw, err = appendNaddrTLV(raw, naddrTLVIdent, []byte(identifier))
	if err != nil {
		return "", err
	}
	raw, err = appendNaddrTLV(raw, naddrTLVAuthor, pubkey)
	if err != nil {
		return "", err
	}
	kindBytes := make([]byte, naddrKindByteSize)
	binary.BigEndian.PutUint32(kindBytes, kind)
	raw, err = appendNaddrTLV(raw, naddrTLVKind, kindBytes)
	if err != nil {
		return "", err
	}
	data, err := bech32.ConvertBits(raw, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("encode naddr data: %w", err)
	}
	return bech32.Encode(naddrHRP, data)
}

func DecodeNaddr(naddr string) (kind uint32, pubkeyHex string, identifier string, err error) {
	naddr = strings.TrimSpace(naddr)
	hrp, data, err := bech32.Decode(naddr)
	if err != nil && strings.Contains(err.Error(), "invalid bech32 string length") {
		hrp, data, err = decodeBech32NoLimit(naddr)
	}
	if err != nil {
		return 0, "", "", fmt.Errorf("decode naddr: %w", err)
	}
	if hrp != naddrHRP {
		return 0, "", "", fmt.Errorf("bech32 human-readable part %q is not naddr", hrp)
	}
	raw, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return 0, "", "", fmt.Errorf("decode naddr data: %w", err)
	}
	var haveIdentifier, havePubkey, haveKind bool
	for i := 0; i < len(raw); {
		if len(raw)-i < 2 {
			return 0, "", "", fmt.Errorf("malformed naddr TLV: trailing byte")
		}
		t := raw[i]
		l := int(raw[i+1])
		i += 2
		if len(raw)-i < l {
			return 0, "", "", fmt.Errorf("malformed naddr TLV: length exceeds data")
		}
		value := raw[i : i+l]
		i += l
		switch t {
		case naddrTLVIdent:
			if haveIdentifier {
				return 0, "", "", fmt.Errorf("naddr contains duplicate identifier")
			}
			identifier = string(value)
			haveIdentifier = true
		case naddrTLVAuthor:
			if havePubkey {
				return 0, "", "", fmt.Errorf("naddr contains duplicate author pubkey")
			}
			if len(value) != 32 {
				return 0, "", "", fmt.Errorf("naddr author pubkey must be 32 bytes")
			}
			pubkeyHex = hex.EncodeToString(value)
			havePubkey = true
		case naddrTLVKind:
			if haveKind {
				return 0, "", "", fmt.Errorf("naddr contains duplicate kind")
			}
			if len(value) != naddrKindByteSize {
				return 0, "", "", fmt.Errorf("naddr kind must be 4 bytes")
			}
			kind = binary.BigEndian.Uint32(value)
			haveKind = true
		}
	}
	if !haveIdentifier || identifier == "" {
		return 0, "", "", fmt.Errorf("naddr identifier is required")
	}
	if !havePubkey {
		return 0, "", "", fmt.Errorf("naddr author pubkey is required")
	}
	if !haveKind {
		return 0, "", "", fmt.Errorf("naddr kind is required")
	}
	return kind, pubkeyHex, identifier, nil
}

func appendNaddrTLV(raw []byte, typ byte, value []byte) ([]byte, error) {
	if len(value) > 255 {
		return nil, fmt.Errorf("naddr TLV value for type %d exceeds 255 bytes", typ)
	}
	raw = append(raw, typ, byte(len(value)))
	raw = append(raw, value...)
	return raw, nil
}

const naddrCharset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var naddrChecksumGen = []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

func decodeBech32NoLimit(value string) (string, []byte, error) {
	if len(value) < 8 {
		return "", nil, fmt.Errorf("invalid bech32 string length %d", len(value))
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 33 || value[i] > 126 {
			return "", nil, fmt.Errorf("invalid character in string: %q", value[i])
		}
	}
	lower := strings.ToLower(value)
	upper := strings.ToUpper(value)
	if value != lower && value != upper {
		return "", nil, fmt.Errorf("string not all lowercase or all uppercase")
	}
	value = lower
	one := strings.LastIndexByte(value, '1')
	if one < 1 || one+7 > len(value) {
		return "", nil, fmt.Errorf("invalid index of 1")
	}
	hrp := value[:one]
	encoded := value[one+1:]
	decoded := make([]byte, 0, len(encoded))
	for i := 0; i < len(encoded); i++ {
		index := strings.IndexByte(naddrCharset, encoded[i])
		if index < 0 {
			return "", nil, fmt.Errorf("invalid character not part of charset: %v", encoded[i])
		}
		decoded = append(decoded, byte(index))
	}
	if !verifyBech32ChecksumNoLimit(hrp, decoded) {
		return "", nil, fmt.Errorf("checksum failed")
	}
	return hrp, decoded[:len(decoded)-6], nil
}

func verifyBech32ChecksumNoLimit(hrp string, data []byte) bool {
	values := append(expandBech32HRPNoLimit(hrp), bytesToInts(data)...)
	return bech32PolymodNoLimit(values) == 1
}

func bytesToInts(data []byte) []int {
	out := make([]int, len(data))
	for i, b := range data {
		out[i] = int(b)
	}
	return out
}

func expandBech32HRPNoLimit(hrp string) []int {
	out := make([]int, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		out = append(out, int(hrp[i]>>5))
	}
	out = append(out, 0)
	for i := 0; i < len(hrp); i++ {
		out = append(out, int(hrp[i]&31))
	}
	return out
}

func bech32PolymodNoLimit(values []int) int {
	chk := 1
	for _, value := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ value
		for i := 0; i < 5; i++ {
			if (top>>uint(i))&1 == 1 {
				chk ^= naddrChecksumGen[i]
			}
		}
	}
	return chk
}
