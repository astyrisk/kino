package extractor

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	urlValidationPrefix = "https"
	urlValidationMinLen = 5
	shiftAmount3        = 3
	shiftAmount5        = 5
	shiftAmount7        = 7
	shiftAmount1        = 1
)

// reverseString reverses a string using rune slices for proper Unicode handling
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// decodeBase64WithVariants attempts different Base64 decoding variants
func decodeBase64WithVariants(input string) ([]byte, error) {
	// Try standard Base64
	if decoded, err := base64.StdEncoding.DecodeString(input); err == nil {
		return decoded, nil
	}
	
	// Try URL encoding variant
	if decoded, err := base64.URLEncoding.DecodeString(input); err == nil {
		return decoded, nil
	}
	
	// Try RawStdEncoding (without padding)
	if decoded, err := base64.RawStdEncoding.DecodeString(input); err == nil {
		return decoded, nil
	}
	
	return nil, fmt.Errorf("base64 decoding failed for all variants")
}

// applyCharacterShift applies a character shift to bytes
func applyCharacterShift(data []byte, shift int) []byte {
	result := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		result[i] = data[i] - byte(shift)
	}
	return result
}

// validateURL checks if a string is a valid HTTPS URL
func validateURL(url string) bool {
	return len(url) >= urlValidationMinLen && strings.HasPrefix(url, urlValidationPrefix)
}

// DecodeString tries multiple decoding functions until finding one
// that produces output starting with "https"
func DecodeString(encoded string) (string, error) {
	decoders := []func(string) (string, error){
		DecodeROT13Base64,
		DecodeReverseBase64Shift,
		DecodeROT3Arithmetic,
		DecodeXORHexReverse,
		DecodeHexXORShiftBase64,
		DecodeReverseBase64Shift5,
		DecodeReverseBase64Shift7,
		DecodeReverseShift1Hex,
		DecodeReverseEvenBase64,
	}

	for _, decoder := range decoders {
		decoded, err := decoder(encoded)
		if err != nil {
			continue
		}

		if validateURL(decoded) {
			return decoded, nil
		}
	}

	return "", errors.New("no decoder produced a valid https URL")
}


func DecodeROT13Base64(encoded string) (string, error) {
	rot13Result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			base := 'A'
			if r >= 'a' {
				base = 'a'
			}
			return base + (r-base+13)%26
		}
		return r
	}, encoded)

	decodedBytes, err := decodeBase64WithVariants(rot13Result)
	if err != nil {
		return "", fmt.Errorf("base64 decoding failed after ROT13: %v", err)
	}

	return string(decodedBytes), nil
}

func DecodeReverseBase64Shift(encoded string) (string, error) {
	reversed := reverseString(encoded)
	
	standardBase64 := strings.NewReplacer("-", "+", "_", "/").Replace(reversed)
	
	decodedBytes, err := base64.StdEncoding.DecodeString(standardBase64)
	if err != nil {
		if len(standardBase64)%4 != 0 {
			standardBase64 += strings.Repeat("=", 4-len(standardBase64)%4)
			decodedBytes, err = base64.StdEncoding.DecodeString(standardBase64)
		}
		if err != nil {
			return "", fmt.Errorf("base64 decode failed: %v", err)
		}
	}

	result := applyCharacterShift(decodedBytes, shiftAmount3)
	return string(result), nil
}

// DecodeROT3Arithmetic implements ROT3 using arithmetic instead of map
func DecodeROT3Arithmetic(input string) (string, error) {
	result := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			// Lowercase: shift by 3, wrap around at 'z'
			return 'a' + (r-'a'+3)%26
		case r >= 'A' && r <= 'Z':
			// Uppercase: shift by 3, wrap around at 'Z'
			return 'A' + (r-'A'+3)%26
		default:
			// Non-alphabetic characters unchanged
			return r
		}
	}, input)

	return result, nil
}

func DecodeXORHexReverse(encodedInput string) (string, error) {
	const xorKey = "X9a(O;FMV2-7VO5x;Ao\x05:dN1NoFs?j,"

	reversed := reverseString(encodedInput)

	var bytes []byte
	if len(reversed)%2 != 0 {
		padded := "0" + reversed
		var err error
		bytes, err = hex.DecodeString(padded)
		if err != nil {
			return "", fmt.Errorf("failed to decode padded hex: %v", err)
		}
	} else {
		var err error
		bytes, err = hex.DecodeString(reversed)
		if err != nil {
			return "", fmt.Errorf("failed to decode hex: %v", err)
		}
	}

	result := make([]byte, len(bytes))
	for i := 0; i < len(bytes); i++ {
		result[i] = bytes[i] ^ xorKey[i%len(xorKey)]
	}

	return string(result), nil
}

// DecodeHexXORShiftBase64 implements:
// 1. Convert hex to bytes
// 2. XOR with repeating key
// 3. Subtract 3 from each byte
// 4. Base64 decode
func DecodeHexXORShiftBase64(hexInput string) (string, error) {
	const xorKey = "pWB9V)[*4I`nJpp?ozyB~dbr9yt!_n4u"

	bytes, err := hex.DecodeString(hexInput)
	if err != nil {
		if len(hexInput)%2 != 0 {
			bytes, err = hex.DecodeString("0" + hexInput)
		}
		if err != nil {
			return "", fmt.Errorf("invalid hex: %v", err)
		}
	}

	xored := make([]byte, len(bytes))
	for i := 0; i < len(bytes); i++ {
		xored[i] = bytes[i] ^ xorKey[i%len(xorKey)]
	}

	shifted := applyCharacterShift(xored, shiftAmount3)

	decoded, err := base64.StdEncoding.DecodeString(string(shifted))
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(string(shifted))
		if err != nil {
			return "", fmt.Errorf("base64 decode failed: %v", err)
		}
	}

	return string(decoded), nil
}

// DecodeReverseBase64Shift5 implements:
// 1. Reverse string
// 2. Replace - → + and _ → / (URL-safe to standard Base64)
// 3. Base64 decode
// 4. Subtract 5 from each character
func DecodeReverseBase64Shift5(inputString string) (string, error) {
	reversed := reverseString(inputString)
	
	standardBase64 := strings.NewReplacer("-", "+", "_", "/").Replace(reversed)
	
	decodedBytes, err := base64.StdEncoding.DecodeString(standardBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	result := applyCharacterShift(decodedBytes, shiftAmount5)
	return string(result), nil
}

// DecodeReverseBase64Shift7 implements:
// 1. Reverse string
// 2. Replace - → + and _ → / (URL-safe to standard Base64)
// 3. Base64 decode
// 4. Subtract 7 from each character
func DecodeReverseBase64Shift7(inputString string) (string, error) {
	reversed := reverseString(inputString)
	
	validBase64 := strings.NewReplacer("-", "+", "_", "/").Replace(reversed)
	
	decoded, err := base64.StdEncoding.DecodeString(validBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	result := applyCharacterShift(decoded, shiftAmount7)
	return string(result), nil
}

// DecodeReverseShift1Hex implements:
// 1. Reverse string
// 2. Subtract 1 from each character (ASCII shift)
// 3. Parse hex pairs to ASCII
func DecodeReverseShift1Hex(input string) (string, error) {
	reversed := reverseString(input)
	
	shifted := applyCharacterShift([]byte(reversed), shiftAmount1)
	
	decoded, err := hex.DecodeString(string(shifted))
	if err != nil {
		return "", fmt.Errorf("hex decode failed: %v", err)
	}

	return string(decoded), nil
}

// DecodeReverseEvenBase64 implements:
// 1. Reverse string
// 2. Extract characters at even indices (0, 2, 4...)
// 3. Base64 decode
func DecodeReverseEvenBase64(input string) (string, error) {
	reversed := reverseString(input)
	
	var extracted strings.Builder
	for i := 0; i < len(reversed); i += 2 {
		extracted.WriteByte(reversed[i])
	}

	decoded, err := base64.StdEncoding.DecodeString(extracted.String())
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	return string(decoded), nil
}
