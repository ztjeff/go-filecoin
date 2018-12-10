package process

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/ipfs/iptb/testbed/interfaces"
)

type NullWriter struct{}

func NewNullWriter() io.Writer {
	return &NullWriter{}
}

func (nw *NullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

type LogWindow struct {
	log *bufio.Reader
	buf *bytes.Buffer
	max int
}

func NewLogWindow(log io.Reader, max int) *LogWindow {
	return &LogWindow{
		log: bufio.NewReader(log),
		max: max,
	}
}

// drain reads everything from the log into a null writer
func (lw *LogWindow) drain() {
	nw := NewNullWriter()
	io.Copy(nw, lw.log)
}

func (lw *LogWindow) StartCapture() func() io.Reader {
	lw.drain()
	lw.buf = bytes.NewBuffer(make([]byte, 0))
	return lw.stopCapture
}

func (lw *LogWindow) stopCapture() io.Reader {
	for {
		n, err := io.CopyN(lw.buf, lw.log, 1024)
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		if n < 1024 {
			break
		} else {
			bcap := lw.buf.Cap()
			if bcap < lw.max+1024 {
				lw.buf.Grow(1024)
			} else {
				// We can't read anymore
				break
			}
		}
	}

	return lw.buf
}

func (lw *LogWindow) Empty() bool {
	return lw.buf.Len() == 0
}

func (lw *LogWindow) Window() io.Reader {
	return lw.buf
}

func (f *Filecoin) openLogWindow() error {
	mn, ok := f.Core.(testbedi.Metric)
	if !ok {
		return fmt.Errorf("IPTB plugin %s does not implement Metric", f.pluginType)
	}

	stderr, err := mn.StderrReader()
	if err != nil {
		return err
	}

	f.logWindow = NewLogWindow(stderr, 1024*1024*5) // 5MB max log window

	return nil
}
