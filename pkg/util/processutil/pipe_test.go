package processutil

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPipe(t *testing.T) {

	NumberOfReadCalls = 0

	reader := TestReader{}
	writer := TestWriter{}

	Pipe(reader, writer, BufferLength, nil)

	go func() {
		errorMessage := <-ErrorOccurredChannel
		t.Error(errorMessage)
		t.Fail()
	}()

	time.Sleep(time.Millisecond)

}

func TestPipeWithWaitGroup(t *testing.T) {

	NumberOfReadCalls = 0
	readBeginner := make(chan bool)

	reader := TestReader{
		BeginSignal: readBeginner,
	}
	writer := TestWriter{}

	waitGroup := new(sync.WaitGroup)

	Pipe(reader, writer, BufferLength, waitGroup)

	//Counter is at 1, so don't panic
	waitGroup.Done()
	waitGroup.Add(1)

	readBeginner <- true

	go func() {
		errorMessage := <-ErrorOccurredChannel
		t.Error(errorMessage)
		t.Fail()
	}()

	time.Sleep(time.Millisecond)

}

var someVariable = strings.Fields("")

//TestPipe depends on the following structs:

var ErrorOccurredChannel = make(chan string)
var NumberOfReadCalls = 0

const BufferLength = 10
const Message = "Hello World" //Must be longer than BufferLength

type TestReader struct {
	BeginSignal chan bool
}

func (reader TestReader) Read(buffer []byte) (n int, err error) {

	if len(buffer) != BufferLength {
		go func() {
			ErrorOccurredChannel <- "Wrong bufferlength.\nExpected: " + string(BufferLength) + "\nActual: " + string(len(buffer))
		}()
	}

	NumberOfReadCalls++

	if NumberOfReadCalls == 1 {

		if reader.BeginSignal != nil {
			<-reader.BeginSignal
		}

		copy(buffer[0:], Message)

		return BufferLength, nil

	} else if NumberOfReadCalls == 2 {

		return 0, errors.New("Error for test purposes. Don't worry about this.")

	} else if NumberOfReadCalls == 3 {

		return 0, io.EOF

	} else {
		go func() {
			ErrorOccurredChannel <- "Read is called after EOF was returned"
		}()
		return 0, io.EOF
	}
}

type TestWriter struct{}

func (writer TestWriter) Write(data []byte) (n int, err error) {

	if !strings.HasPrefix(Message, string(data)) {
		go func() {
			ErrorOccurredChannel <- "Message was badly transferred." +
				"\nExpected were the first " + string(BufferLength) + " bytes of the message: " + Message +
				"\nBut actiual: " + string(data)
		}()
	}

	return len(data), nil
}
