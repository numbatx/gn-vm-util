package mandosvalueinterpreter

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	twos "github.com/numbatx/gn-bigint/twos-complement"
	fr "github.com/numbatx/gn-vm-util/test-util/mandos/json/fileresolver"
	oj "github.com/numbatx/gn-vm-util/test-util/orderedjson"
)

var strPrefixes = []string{"str:", "``", "''"}

const addrPrefix = "address:"
const filePrefix = "file:"
const keccak256Prefix = "keccak256:"

const u64Prefix = "u64:"
const u32Prefix = "u32:"
const u16Prefix = "u16:"
const u8Prefix = "u8:"
const i64Prefix = "i64:"
const i32Prefix = "i32:"
const i16Prefix = "i16:"
const i8Prefix = "i8:"

// ValueInterpreter provides context for computing Mandos values.
type ValueInterpreter struct {
	FileResolver fr.FileResolver
}

// InterpretSubTree attempts to produce a value based on a JSON subtree.
// Subtrees are composed of strings, lists and maps.
// The idea is to intuitively represent serialized objects.
// Lists are evaluated by concatenating their items' representations.
// Maps are evaluated by concatenating their values' representations (keys are ignored).
// See InterpretString on how strings are being interpreted.
func (vi *ValueInterpreter) InterpretSubTree(obj oj.OJsonObject) ([]byte, error) {
	if str, isStr := obj.(*oj.OJsonString); isStr {
		return vi.InterpretString(str.Value)
	}

	if list, isList := obj.(*oj.OJsonList); isList {
		var concat []byte
		for _, item := range list.AsList() {
			value, err := vi.InterpretSubTree(item)
			if err != nil {
				return []byte{}, err
			}
			concat = append(concat, value...)
		}
		return concat, nil
	}

	if mp, isMap := obj.(*oj.OJsonMap); isMap {
		var concat []byte
		for _, kvp := range mp.OrderedKV {
			// keys are ignored, they do not form the value but act like documentation
			value, err := vi.InterpretSubTree(kvp.Value)
			if err != nil {
				return []byte{}, err
			}
			concat = append(concat, value...)
		}
		return concat, nil
	}

	return []byte{}, errors.New("cannot interpret given JSON subtree as value")
}

// InterpretString resolves a string to a byte slice according to the Mandos value format.
// Supported rules are:
// - numbers: decimal, hex, binary, signed/unsigned
// - fixed length numbers: "u32:5", "i8:-3", etc.
// - ascii strings as "str:...", "“...", "”..."
// - "true"/"false"
// - "address:..."
// - "file:..."
// - "keccak256:..."
// - concatenation using |
func (vi *ValueInterpreter) InterpretString(strRaw string) ([]byte, error) {
	if len(strRaw) == 0 {
		return []byte{}, nil
	}

	// file contents
	// TODO: make this part of a proper parser
	if strings.HasPrefix(strRaw, filePrefix) {
		if vi.FileResolver == nil {
			return []byte{}, errors.New("parser FileResolver not provided")
		}
		fileContents, err := vi.FileResolver.ResolveFileValue(strRaw[len(filePrefix):])
		if err != nil {
			return []byte{}, err
		}
		return fileContents, nil
	}

	// keccak256
	// TODO: make this part of a proper parser
	if strings.HasPrefix(strRaw, keccak256Prefix) {
		arg, err := vi.InterpretString(strRaw[len(keccak256Prefix):])
		if err != nil {
			return []byte{}, fmt.Errorf("cannot parse keccak256 argument: %w", err)
		}
		hash, err := keccak256(arg)
		if err != nil {
			return []byte{}, fmt.Errorf("error computing keccak256: %w", err)
		}
		return hash, nil
	}

	// concatenate values of different formats
	// TODO: make this part of a proper parser
	parts := strings.Split(strRaw, "|")
	if len(parts) > 1 {
		concat := make([]byte, 0)
		for _, part := range parts {
			eval, err := vi.InterpretString(part)
			if err != nil {
				return []byte{}, err
			}
			concat = append(concat, eval...)
		}
		return concat, nil
	}

	if strRaw == "false" {
		return []byte{}, nil
	}

	if strRaw == "true" {
		return []byte{0x01}, nil
	}

	// allow ascii strings, for readability
	for _, strPrefix := range strPrefixes {
		if strings.HasPrefix(strRaw, strPrefix) {
			str := strRaw[len(strPrefix):]
			return []byte(str), nil
		}
	}

	// address
	if strings.HasPrefix(strRaw, addrPrefix) {
		addrName := strRaw[len(addrPrefix):]
		return address([]byte(addrName))
	}

	// fixed width numbers
	parsed, result, err := vi.tryInterpretFixedWidth(strRaw)
	if err != nil {
		return nil, err
	}
	if parsed {
		return result, nil
	}

	// general numbers, arbitrary length
	return vi.interpretNumber(strRaw, 0)
}

// targetWidth = 0 means minimum length that can contain the result
func (vi *ValueInterpreter) interpretNumber(strRaw string, targetWidth int) ([]byte, error) {
	// signed numbers
	if strRaw[0] == '-' || strRaw[0] == '+' {
		numberBytes, err := vi.interpretUnsignedNumber(strRaw[1:])
		if err != nil {
			return []byte{}, err
		}
		number := big.NewInt(0).SetBytes(numberBytes)
		if strRaw[0] == '-' {
			number = number.Neg(number)
		}
		if targetWidth == 0 {
			return twos.ToBytes(number), nil
		}

		return twos.ToBytesOfLength(number, targetWidth)
	}

	// unsigned numbers
	if targetWidth == 0 {
		return vi.interpretUnsignedNumber(strRaw)
	}

	return vi.interpretUnsignedNumberFixedWidth(strRaw, targetWidth)
}

func (vi *ValueInterpreter) interpretUnsignedNumber(strRaw string) ([]byte, error) {
	str := strings.ReplaceAll(strRaw, "_", "") // allow underscores, to group digits
	str = strings.ReplaceAll(str, ",", "")     // also allow commas to group digits

	// hex, the usual representation
	if strings.HasPrefix(strRaw, "0x") || strings.HasPrefix(strRaw, "0X") {
		str := strRaw[2:]
		if len(str)%2 == 1 {
			str = "0" + str
		}
		return hex.DecodeString(str)
	}

	// binary representation
	if strings.HasPrefix(strRaw, "0b") || strings.HasPrefix(strRaw, "0B") {
		result := new(big.Int)
		var parseOk bool
		result, parseOk = result.SetString(str[2:], 2)
		if !parseOk {
			return []byte{}, fmt.Errorf("could not parse binary value: %s", strRaw)
		}

		return result.Bytes(), nil
	}

	// default: parse as BigInt, base 10
	result := new(big.Int)
	var parseOk bool
	result, parseOk = result.SetString(str, 10)
	if !parseOk {
		return []byte{}, fmt.Errorf("could not parse base 10 value: %s", strRaw)
	}

	return result.Bytes(), nil
}

func (vi *ValueInterpreter) interpretUnsignedNumberFixedWidth(strRaw string, targetWidth int) ([]byte, error) {
	numberBytes, err := vi.interpretUnsignedNumber(strRaw)
	if err != nil {
		return []byte{}, err
	}
	if targetWidth == 0 {
		return numberBytes, nil
	}

	if len(numberBytes) > targetWidth {
		return []byte{}, fmt.Errorf("representation of %s does not fit in %d bytes", strRaw, targetWidth)
	}
	return twos.CopyAlignRight(numberBytes, targetWidth), nil
}

func (vi *ValueInterpreter) tryInterpretFixedWidth(strRaw string) (bool, []byte, error) {
	if strings.HasPrefix(strRaw, u64Prefix) {
		r, err := vi.interpretUnsignedNumberFixedWidth(strRaw[len(u64Prefix):], 8)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, u32Prefix) {
		r, err := vi.interpretUnsignedNumberFixedWidth(strRaw[len(u32Prefix):], 4)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, u16Prefix) {
		r, err := vi.interpretUnsignedNumberFixedWidth(strRaw[len(u16Prefix):], 2)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, u8Prefix) {
		r, err := vi.interpretUnsignedNumberFixedWidth(strRaw[len(u8Prefix):], 1)
		return true, r, err
	}

	if strings.HasPrefix(strRaw, i64Prefix) {
		r, err := vi.interpretNumber(strRaw[len(i64Prefix):], 8)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, i32Prefix) {
		r, err := vi.interpretNumber(strRaw[len(i32Prefix):], 4)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, i16Prefix) {
		r, err := vi.interpretNumber(strRaw[len(i16Prefix):], 2)
		return true, r, err
	}
	if strings.HasPrefix(strRaw, i8Prefix) {
		r, err := vi.interpretNumber(strRaw[len(i8Prefix):], 1)
		return true, r, err
	}

	return false, []byte{}, nil
}
