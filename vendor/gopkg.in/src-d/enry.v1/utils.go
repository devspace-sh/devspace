package enry

import (
	"bytes"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/enry.v1/data"
)

var (
	auxiliaryLanguages = map[string]bool{
		"Other": true, "XML": true, "YAML": true, "TOML": true, "INI": true,
		"JSON": true, "TeX": true, "Public Key": true, "AsciiDoc": true,
		"AGS Script": true, "VimL": true, "Diff": true, "CMake": true, "fish": true,
		"Awk": true, "Graphviz (DOT)": true, "Markdown": true, "desktop": true,
		"XSLT": true, "SQL": true, "RMarkdown": true, "IRC log": true,
		"reStructuredText": true, "Twig": true, "CSS": true, "Batchfile": true,
		"Text": true, "HTML+ERB": true, "HTML": true, "Gettext Catalog": true,
		"Smarty": true, "Raw token data": true,
	}

	configurationLanguages = map[string]bool{
		"XML": true, "JSON": true, "TOML": true, "YAML": true, "INI": true, "SQL": true,
	}
)

// IsAuxiliaryLanguage returns whether or not lang is an auxiliary language.
func IsAuxiliaryLanguage(lang string) bool {
	_, ok := auxiliaryLanguages[lang]
	return ok
}

// IsConfiguration returns whether or not path is using a configuration language.
func IsConfiguration(path string) bool {
	language, _ := GetLanguageByExtension(path)
	_, is := configurationLanguages[language]
	return is
}

// IsDotFile returns whether or not path has dot as a prefix.
func IsDotFile(path string) bool {
	path = filepath.Clean(path)
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".") && base != "." && base != ".."
}

// IsVendor returns whether or not path is a vendor path.
func IsVendor(path string) bool {
	return data.VendorMatchers.Match(path)
}

// IsDocumentation returns whether or not path is a documentation path.
func IsDocumentation(path string) bool {
	return data.DocumentationMatchers.Match(path)
}

func IsImage(path string) bool {
	extension := filepath.Ext(path)
	if extension == ".png" || extension == ".jpg" || extension == ".jpeg" || extension == ".gif" {
		return true
	}

	return false
}

func GetMimeType(path string, language string) string {
	if mime, ok := data.LanguagesMime[language]; ok {
		return mime
	}

	if IsImage(path) {
		return "image/" + filepath.Ext(path)[1:]
	}

	return "text/plain"
}

const sniffLen = 8000

// IsBinary detects if data is a binary value based on:
// http://git.kernel.org/cgit/git/git.git/tree/xdiff-interface.c?id=HEAD#n198
func IsBinary(data []byte) bool {
	if len(data) > sniffLen {
		data = data[:sniffLen]
	}

	if bytes.IndexByte(data, byte(0)) == -1 {
		return false
	}

	return true
}

// FileCount type stores language name and count of files belonging to the
// language.
type FileCount struct {
	Name  string
	Count int
}

// FileCountList type is a list of FileCounts.
type FileCountList []FileCount

func (fcl FileCountList) Len() int           { return len(fcl) }
func (fcl FileCountList) Less(i, j int) bool { return fcl[i].Count < fcl[j].Count }
func (fcl FileCountList) Swap(i, j int)      { fcl[i], fcl[j] = fcl[j], fcl[i] }
