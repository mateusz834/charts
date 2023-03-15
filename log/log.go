package log

import (
	"fmt"
	"log/syslog"
	"os"
	"time"
)

type Logger interface {
	Debug(msg string)
	Error(msg string)
}

type ConsoleLogger struct{}

func (l *ConsoleLogger) formarMsg(msg string) string {
	const format = time.DateTime + ".000000 -0700"
	return time.Now().Format(format) + " " + msg + "\n"
}

func (l *ConsoleLogger) Debug(msg string) {
	os.Stdout.WriteString(l.formarMsg(msg))
}

func (l *ConsoleLogger) Error(msg string) {
	os.Stderr.WriteString(l.formarMsg(msg))
}

type SyslogLogger struct{ writer *syslog.Writer }

func NewSyslogLogger() Logger {
	writer, err := syslog.New(syslog.LOG_LOCAL0, "")
	if err != nil {
		c := &ConsoleLogger{}
		c.Error(fmt.Sprintf("Failed to create syslog logger: %v", err))
		return c
	}
	return &SyslogLogger{writer: writer}
}

func (l *SyslogLogger) Debug(msg string) {
	if err := l.writer.Debug(msg); err != nil {
		c := &ConsoleLogger{}
		c.Debug(fmt.Sprintf("failed to write to syslog logger: %v: msg: %v", err, msg))
	}
}

func (l *SyslogLogger) Error(msg string) {
	if err := l.writer.Err(msg); err != nil {
		c := &ConsoleLogger{}
		c.Error(fmt.Sprintf("failed to write to syslog logger: %v: msg: %v", err, msg))
	}
}
