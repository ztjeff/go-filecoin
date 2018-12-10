package pluginlocalfilecoin

import (
	"io"
)

func (l *Localfilecoin) Events() (io.ReadCloser, error) {
	return nil, nil
}

func (l *Localfilecoin) StderrReader() (io.ReadCloser, error) {
	return l.readerFor("daemon.stderr")
}

func (l *Localfilecoin) StdoutReader() (io.ReadCloser, error) {
	return l.readerFor("daemon.stdout")
}

func (l *Localfilecoin) Heartbeat() (map[string]string, error) {
	return nil, nil
}

func (l *Localfilecoin) Metric(key string) (string, error) {
	return "", nil
}

func (l *Localfilecoin) GetMetricList() []string {
	return []string{}
}

func (l *Localfilecoin) GetMetricDesc(key string) (string, error) {
	return "", nil
}
