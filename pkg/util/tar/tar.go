package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

// ExtractSingleFileTarGz is a function to unpack a tar.gz
func ExtractSingleFileTarGz(archivepath, fileToExtract, extractToPath string) error {
	if archivepath == "" || fileToExtract == "" {
		return errors.New("Empty archivepath or fileToExtract")
	}

	f, err := os.Open(archivepath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzf)

	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := header.Name
		if name == fileToExtract {
			// if header.Typeflag != tar.TypeReg {
			// 	return fmt.Errorf("%s is a directory and not a file", fileToExtract)
			// }

			outFile, err := os.Create(extractToPath)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("%s not found in archive", fileToExtract)
}

// ExtractSingleFileToStringTarGz is a function to unpack a tar.gz
func ExtractSingleFileToStringTarGz(archivepath, fileToExtract string) (string, error) {
	if archivepath == "" || fileToExtract == "" {
		return "", errors.New("Empty archivepath or fileToExtract")
	}

	f, err := os.Open(archivepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}

	tarReader := tar.NewReader(gzf)

	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		name := header.Name
		if name == fileToExtract {
			if header.Typeflag != tar.TypeReg {
				return "", fmt.Errorf("%s is a directory and not a file", fileToExtract)
			}

			buf := new(bytes.Buffer)
			buf.ReadFrom(tarReader)

			return buf.String(), nil
		}
	}

	return "", fmt.Errorf("%s not found in archive", fileToExtract)
}
