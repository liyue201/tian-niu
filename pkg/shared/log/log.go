package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func init() {
	ioWriter := io.MultiWriter(os.Stdout)
	logrus.SetOutput(ioWriter)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&MyFormatter{})
	return
}

type MyFormatter struct{}

func (m *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string

	if entry.HasCaller() {
		//fName := filepath.Base(entry.Caller.File)
		fName := pkFile(entry.Caller.File)
		newLog = fmt.Sprintf("[%s] [%s] [%s:%d] %s\n",
			timestamp, entry.Level, fName, entry.Caller.Line, entry.Message)
	} else {
		newLog = fmt.Sprintf("[%s] [%s] %s\n", timestamp, entry.Level, entry.Message)
	}

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func pkFile(filePath string) string {
	dir := filepath.Dir(filePath)
	filename := filepath.Base(dir) + "/" + filepath.Base(filePath)
	return filename
}
