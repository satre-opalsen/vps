package utils

import (
	"strings"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/types"
)

const base36Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const codeMask uint32 = 0x2B4D6F8A // Mask to scramble sequential messageIDs into 6-digit alphanumeric codes

func EncodeMessageID(id int) string {
	val := uint32(id) ^ (codeMask % 2176782336)
	var buf [6]byte
	for i := 5; i >= 0; i-- {
		buf[i] = base36Chars[val%36]
		val /= 36
	}
	return string(buf[:])
}

func DecodeMessageID(code string) int {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 {
		return 0
	}
	var val uint32
	for i := 0; i < 6; i++ {
		idx := strings.IndexByte(base36Chars, code[i])
		if idx == -1 {
			return 0
		}
		val = val*36 + uint32(idx)
	}
	return int(val ^ (codeMask % 2176782336))
}

func PackFile(fileName string, fileSize int64, mimeType string, fileID int64) string {
	return (&types.HashableFileStruct{FileName: fileName, FileSize: fileSize, MimeType: mimeType, FileID: fileID}).Pack()
}

func GetShortHash(fullHash string) string {
	return fullHash[:config.ValueOf.HashLength]
}

func CheckHash(inputHash string, expectedHash string) bool {
	return inputHash == GetShortHash(expectedHash)
}
