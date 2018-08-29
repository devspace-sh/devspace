package processutil

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

func Pipe(reader io.Reader, writer io.Writer, bufferSize int, wg *sync.WaitGroup) {
	pipeFunction := func() {
		for true {
			buffer := make([]byte, bufferSize)

			byteCount, readErr := reader.Read(buffer)

			if byteCount > 0 {
				writer.Write(buffer[:byteCount])
			} else if readErr != nil {
				if readErr == io.EOF {
					break
				} else {
					fmt.Println(readErr)
				}
			}
		}
	}

	if wg != nil {
		wg.Add(1)

		go func() {
			pipeFunction()
			wg.Done()
		}()
	} else {
		pipeFunction()
	}
}

type LineFunction func(line string)

func RunOnEveryLine(reader io.Reader, lineFunction LineFunction, bufferSize int, wg *sync.WaitGroup) {
	pushLinesToFunction := func(lineBuffer string) string {
		lines := strings.Split(lineBuffer, "\n")
		lastLineIndex := len(lines) - 1

		for _, line := range lines[:lastLineIndex] {
			lineFunction(line)
		}
		return lines[lastLineIndex]
	}

	pipeFunction := func() {
		lineBuffer := ""

		for true {
			buffer := make([]byte, bufferSize)

			byteCount, readErr := reader.Read(buffer)

			if byteCount > 0 {
				lineBuffer = lineBuffer + string(buffer[:byteCount])

				lineBuffer = pushLinesToFunction(lineBuffer)
			} else if readErr != nil {
				if readErr == io.EOF {
					if len(lineBuffer) > 0 {
						_ = pushLinesToFunction(lineBuffer)
					}
					break
				} else {
					fmt.Println(readErr)
				}
			}
		}
	}

	if wg != nil {
		wg.Add(1)

		go func() {
			pipeFunction()
			wg.Done()
		}()
	} else {
		pipeFunction()
	}
}
