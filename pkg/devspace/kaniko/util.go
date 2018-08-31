package kaniko

import (
	"io"
	"regexp"
	"sync"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/processutil"
)

// OutputFormat a regex and a replacement for outputs
type OutputFormat struct {
	Regex       *regexp.Regexp
	Replacement string
}

func formatKanikoOutput(stdout io.ReadCloser, stderr io.ReadCloser) string {
	wg := &sync.WaitGroup{}
	lastLine := ""
	outputFormats := []OutputFormat{
		{
			Regex:       regexp.MustCompile(`.* msg="Downloading base image (.*)"`),
			Replacement: " FROM $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="(Unpacking layer: \d+)"`),
			Replacement: ">> $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="cmd: Add \[(.*)\]"`),
			Replacement: " ADD $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="cmd: copy \[(.*)\]"`),
			Replacement: " COPY $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="dest: (.*)"`),
			Replacement: ">> destination: $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="args: \[-c (.*)\]"`),
			Replacement: " RUN $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Replacing CMD in config with \[(.*)\]"`),
			Replacement: " CMD $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Changed working directory to (.*)"`),
			Replacement: " WORKDIR $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Taking snapshot of full filesystem..."`),
			Replacement: " Packaging layers",
		},
	}

	kanikoLogRegex := regexp.MustCompile(`^time="(.*)" level=(.*) msg="(.*)"`)
	buildPrefix := "build >"

	printFormattedOutput := func(originalLine string) {
		line := []byte(originalLine)

		for _, outputFormat := range outputFormats {
			line = outputFormat.Regex.ReplaceAll(line, []byte(outputFormat.Replacement))
		}

		lineString := string(line)

		if len(line) != len(originalLine) {
			log.Done(buildPrefix + lineString)
		} else if kanikoLogRegex.Match(line) == false {
			log.Info(buildPrefix + ">> " + lineString)
		}

		lastLine = string(kanikoLogRegex.ReplaceAll([]byte(originalLine), []byte("$3")))
	}

	processutil.RunOnEveryLine(stdout, printFormattedOutput, 500, wg)
	processutil.RunOnEveryLine(stderr, printFormattedOutput, 500, wg)

	wg.Wait()

	return lastLine
}
