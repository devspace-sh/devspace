package logutil

import (
	"github.com/daviddengcn/go-colortext"
	"io"
	"time"
)

var waitInterval = time.Millisecond * 150

type LoadingText struct {
	Log     io.Writer
	Message string

	loadingRune int
	isShown     bool
	isDone      bool
	stopChan    chan bool
}

func NewLoadingText(message string, log io.Writer) *LoadingText {
	loadingText := &LoadingText{
		Log:     log,
		Message: message,

		stopChan: make(chan bool),
	}

	go func() {
		loadingText.render(false)

		for {
			select {
			case <-loadingText.stopChan:
				return
			case <-time.After(waitInterval):
				loadingText.render(false)
			}
		}
	}()

	return loadingText
}

func (l *LoadingText) getLoadingChar() string {
	var loadingChar string

	switch l.loadingRune {
	case 0:
		loadingChar = "|"
	case 1:
		loadingChar = "/"
	case 2:
		loadingChar = "-"
	case 3:
		loadingChar = "\\"
	}

	l.loadingRune++

	if l.loadingRune > 3 {
		l.loadingRune = 0
	}

	return loadingChar
}

func (l *LoadingText) render(isDone bool) {
	if l.isShown == false {
		l.isShown = true
	} else {
		l.Log.Write([]byte("\r"))
	}

	if isDone {
		ct.Foreground(ct.Green, false)
		l.Log.Write([]byte("[DONE] √ "))
		ct.ResetColor()

		l.Log.Write([]byte(l.Message))

	} else {
		ct.Foreground(ct.Red, false)
		l.Log.Write([]byte("[WAIT] "))
		ct.ResetColor()

		l.Log.Write([]byte(l.getLoadingChar() + " " + l.Message))
	}

	if isDone {
		l.Log.Write([]byte("\n"))
	}
}

func (l *LoadingText) Done() {
	if !l.isDone {
		l.isDone = true
		l.stopChan <- true
		l.render(true)
	}
}

func PrintDoneMessage(message string, log io.Writer) {
	ct.Foreground(ct.Green, false)
	log.Write([]byte("[DONE] √ "))
	ct.ResetColor()

	log.Write([]byte(message + "\n"))
}

func PrintFailMessage(message string, log io.Writer) {
	ct.Foreground(ct.Red, false)
	log.Write([]byte("[FAIL] X "))
	ct.ResetColor()

	log.Write([]byte(message + "\n"))
}
