package provider

import (
	"fmt"
	stdlog "log"
	"os"
	"strings"
	"sync"

	nacoslogger "github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

var configureNacosLoggerOnce sync.Once

func configureNacosSDKStdoutLogger(level string) {
	configureNacosLoggerOnce.Do(func() {
		nacoslogger.SetLogger(newNacosStdoutLogger(level))
	})
}

type nacosStdoutLogger struct {
	logger *stdlog.Logger
	level  int
}

func newNacosStdoutLogger(level string) *nacosStdoutLogger {
	return &nacosStdoutLogger{
		logger: stdlog.New(os.Stdout, "", stdlog.LstdFlags|stdlog.Lmicroseconds),
		level:  nacosLogLevel(level),
	}
}

func nacosLogLevel(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warn":
		return 2
	default:
		return 3
	}
}

func (l *nacosStdoutLogger) log(level int, name string, args ...interface{}) {
	if level >= l.level {
		l.logger.Print(append([]interface{}{"[NACOS SDK] ", name, " "}, args...)...)
	}
}

func (l *nacosStdoutLogger) logf(level int, name, format string, args ...interface{}) {
	if level >= l.level {
		l.logger.Printf("[NACOS SDK] %s %s", name, fmt.Sprintf(format, args...))
	}
}

func (l *nacosStdoutLogger) Debug(args ...interface{}) { l.log(0, "DEBUG", args...) }
func (l *nacosStdoutLogger) Info(args ...interface{})  { l.log(1, "INFO", args...) }
func (l *nacosStdoutLogger) Warn(args ...interface{})  { l.log(2, "WARN", args...) }
func (l *nacosStdoutLogger) Error(args ...interface{}) { l.log(3, "ERROR", args...) }

func (l *nacosStdoutLogger) Debugf(format string, args ...interface{}) {
	l.logf(0, "DEBUG", format, args...)
}
func (l *nacosStdoutLogger) Infof(format string, args ...interface{}) {
	l.logf(1, "INFO", format, args...)
}
func (l *nacosStdoutLogger) Warnf(format string, args ...interface{}) {
	l.logf(2, "WARN", format, args...)
}
func (l *nacosStdoutLogger) Errorf(format string, args ...interface{}) {
	l.logf(3, "ERROR", format, args...)
}
func (l *nacosStdoutLogger) Close() error { return nil }
