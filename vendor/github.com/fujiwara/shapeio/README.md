# shapeio

Traffic shaper for Golang io.Reader and io.Writer

```go
import "github.com/fujiwara/shapeio"

func ExampleReader() {
	// example for downloading http body with rate limit.
	resp, _ := http.Get("http://example.com")
	defer resp.Body.Close()

	reader := shapeio.NewReader(resp.Body)
	reader.SetRateLimit(1024 * 10) // 10KB/sec
	io.Copy(ioutil.Discard, reader)
}

func ExampleWriter() {
	// example for writing file with rate limit.
	src := bytes.NewReader(bytes.Repeat([]byte{0}, 32*1024)) // 32KB
	f, _ := os.Create("/tmp/foo")
	writer := shapeio.NewWriter(f)
	writer.SetRateLimit(1024 * 10) // 10KB/sec
	io.Copy(writer, src)
	f.Close()
}
```

## Usage

#### type Reader

```go
type Reader struct {
}
```


#### func  NewReader

```go
func NewReader(r io.Reader) *Reader
```
NewReader returns a reader that implements io.Reader with rate limiting.

#### func (*Reader) Read

```go
func (s *Reader) Read(p []byte) (int, error)
```
Read reads bytes into p.

#### func (*Reader) SetRateLimit

```go
func (s *Reader) SetRateLimit(l float64)
```
SetRateLimit sets rate limit (bytes/sec) to the reader.

#### type Writer

```go
type Writer struct {
}
```


#### func  NewWriter

```go
func NewWriter(w io.Writer) *Writer
```
NewWriter returns a writer that implements io.Writer with rate limiting.

#### func (*Writer) SetRateLimit

```go
func (s *Writer) SetRateLimit(l float64)
```
SetRateLimit sets rate limit (bytes/sec) to the writer.

#### func (*Writer) Write

```go
func (s *Writer) Write(p []byte) (int, error)
```
Write writes bytes from p.

##  License

The MIT License (MIT)

Copyright (c) 2016 FUJIWARA Shunichiro
