package processutil

import (
	"fmt"
	"io"
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
