package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/aiocloud/stream/app/conf"
)

var logger *logrus.Logger

func Init() error {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		ForceQuote:      true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	logger.SetLevel(logrus.InfoLevel)

	outputPath := conf.GetOutputLog()
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		fmt.Errorf("[Sream][Log] Failed to create log dir: %w", err)
		return nil
	}
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {

		return err
	}

	logger.SetOutput(output)

	return nil
}

func Debug(args ...interface{}) {

	logger.Debugln(args...)
}

func Info(args ...interface{}) {

	logger.Infoln(args...)
}

func Infof(format string, args ...interface{}) {
    logger.Infof(format, args...)
}

func Error(args ...interface{}) {

	logger.Errorln(args...)
}

func Warn(args ...interface{}) {

	logger.Warnln(args...)
}

func Fatal(args ...interface{}) {

	logger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
    logger.Fatalf(format, args...)
}

func GetWriter() *io.PipeWriter {

	return logger.Writer()
}
