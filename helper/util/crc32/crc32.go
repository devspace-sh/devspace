package crc32

import (
	"hash/crc32"
	"io"
	"os"
)

func Checksum(filename string) (uint32, error) {
	tab := crc32.NewIEEE()
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	_, err = io.Copy(tab, file)
	if err != nil {
		return 0, err
	}

	return tab.Sum32(), nil
}
