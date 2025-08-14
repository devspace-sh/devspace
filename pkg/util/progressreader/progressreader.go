package progressreader

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
	"time"
)

// ProgressReader wraps an existing io.ReadCloser.
type ProgressReader struct {
	io.ReadCloser
	Ctx           devspacecontext.Context
	total         int64 // Total # of bytes transferred
	transferEnd   time.Time
	transferStart time.Time
	transferStep  time.Time
}

// Override Read method in order to print some progress
// during reading.
func (r *ProgressReader) Read(p []byte) (int, error) {
	if r.transferStart.IsZero() {
		// initialize start timer
		r.transferStart = time.Now()
	}

	curr, err := r.ReadCloser.Read(p)
	r.total += int64(curr)

	if err == nil {
		// initialize the transfer current step, or check if at least 1s is passed
		// in order to not spam progress updates too much
		if r.transferStep.IsZero() || time.Now().Second() >= (r.transferStep.Second()+3) {
			r.transferStep = time.Now()

			r.Ctx.Log().Info("Uploaded " + toHumanReadable(r.total) + " " + r.Rate() + "\r")
		}
	}

	if err == io.EOF {
		r.transferEnd = time.Now()
	}

	return curr, err
}

// Rate returns the rate of progress in b/s
func (r *ProgressReader) Rate() string {
	end := r.transferEnd
	if end.IsZero() {
		end = time.Now()
	}
	return toHumanReadable(int64((float64(r.total) / (end.Sub(r.transferStart).Seconds())))) + "/s"
}

// convert bytes input to a human readable format (Gb,Mb,Kb,b)
func toHumanReadable(input int64) string {
	conversion := float64(input / 1024)

	if conversion < 0 {
		return fmt.Sprintf("%d b", input)
	}

	if conversion < 1000 {
		return fmt.Sprintf("%.2f Kb", conversion)
	}

	conversion = conversion / 1024
	if conversion < 1000 {
		return fmt.Sprintf("%.2f Mb", conversion)
	}

	return fmt.Sprintf("%.2f Gb", conversion/1024)
}
