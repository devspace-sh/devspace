package kaniko

import (
	"io"
	"strings"
)

type kanikoLogger struct {
	out io.Writer
}

// Implement the io.Writer interface
func (k kanikoLogger) Write(p []byte) (n int, err error) {
	str := string(p)

	lines := strings.Split(str, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasSuffix(trimmedLine, ", because it was changed.") {
			continue
		}
		if strings.HasSuffix(trimmedLine, "No matching credentials were found, falling back on anonymous") {
			continue
		}
		if strings.HasPrefix(trimmedLine, "ERROR: logging before flag.Parse:") {
			continue
		}
		if strings.HasSuffix(trimmedLine, "Taking snapshot of full filesystem...") {
			continue
		}
		if strings.HasSuffix(trimmedLine, "Taking snapshot of files...") {
			continue
		}
		if strings.HasSuffix(trimmedLine, "No files changed in this command, skipping snapshotting.") {
			continue
		}
		if strings.Index(trimmedLine, "Error while retrieving image from cache: getting file info") != -1 {
			continue
		}

		newLines = append(newLines, line)
	}

	i, err := k.out.Write([]byte(strings.Join(newLines, "\n")))
	if err != nil {
		return i, err
	}

	return len(p), nil
}
