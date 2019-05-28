package upgrade

import (
	"errors"
	"log"
	"regexp"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

// Version holds the current version tag
var version string
var rawVersion string

var githubSlug = "devspace-cloud/devspace"
var reVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)

func eraseVersionPrefix(version string) (string, error) {
	indices := reVersion.FindStringIndex(version)
	if indices == nil {
		return version, errors.New("Version not adopting semver")
	}
	if indices[0] > 0 {
		version = version[indices[0]:]
	}

	return version, nil
}

// GetVersion returns the application version
func GetVersion() string {
	return version
}

// GetRawVersion returns the applications raw version
func GetRawVersion() string {
	return rawVersion
}

// SetVersion sets the application version
func SetVersion(verText string) {
	if len(verText) > 0 {
		_version, err := eraseVersionPrefix(verText)
		if err != nil {
			log.Fatalf("Error parsing version: %s", err.Error())
		}

		version = _version
		rawVersion = verText
	}
}

// CheckForNewerVersion checks if there is a newer version on github and returns the newer version
func CheckForNewerVersion() (string, error) {
	latest, found, err := selfupdate.DetectLatest(githubSlug)
	if err != nil {
		return "", err
	}

	v := semver.MustParse(version)
	if !found || latest.Version.Equals(v) {
		return "", nil
	}

	return latest.Version.String(), nil
}

// Upgrade downloads the latest release from github and replaces devspace if a new version is found
func Upgrade() error {
	newerVersion, err := CheckForNewerVersion()
	if err != nil {
		return err
	}
	if newerVersion == "" {
		log.Println("Current binary is the latest version: ", version)
		return nil
	}

	v := semver.MustParse(version)

	log.Println("Downloading newest version...")
	latest, err := selfupdate.UpdateSelf(v, githubSlug)
	if err != nil {
		return err
	}

	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version: ", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		log.Println("Release note:\n", latest.ReleaseNotes)
	}

	return nil
}
