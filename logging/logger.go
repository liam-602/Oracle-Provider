// Package logging provides structured logging with logrus.
package logging

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Logger is a configured logrus.Logger.
var Logger *logrus.Logger = nil

// StructuredLogger is a structured logrus Logger.
type StructuredLogger struct {
	Logger *logrus.Logger
}

type UTCFormatter struct {
	logrus.Formatter
}
type CustomLogger struct {
	defaultField string
	formatter    logrus.Formatter
}

func (u UTCFormatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

func (c CustomLogger) Format(e *logrus.Entry) ([]byte, error) {
	e.Data["app"] = c.defaultField
	return c.formatter.Format(e)
}

type customLogFormatter struct {
	logrus.TextFormatter
}

func (f *customLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = 31
	case logrus.WarnLevel:
		levelColor = 33
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = 31
	default:
		levelColor = 36
	}
	appName := entry.Data["app"]
	levelColorAppName := 92
	levelColorVariable := 35
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	var params *bytes.Buffer = &bytes.Buffer{}
	for key, value := range entry.Data {
		f := fmt.Sprintf("%v", value)
		if f != "" && key != "app" {
			params.WriteString(fmt.Sprintf("   \x1b[%dm%s\x1b[0m=%s", levelColorVariable, key, f))
		}
	}
	res := fmt.Sprintf("%s|\x1b[%dm%s\x1b[0m|\x1b[%dm%s\x1b[0m|%s%s\n", entry.Time.UTC().Format(time.RFC3339), levelColor, strings.ToUpper(entry.Level.String()), levelColorAppName, appName, entry.Message, params.String())
	b.WriteString(res)
	return b.Bytes(), nil
}

// NewLogger creates and configures a new logrus Logger.
func NewLogger() *logrus.Logger {
	Logger = logrus.New()
	level := viper.GetString("LOG_LEVEL")
	appName := "fresco-oracle"

	Logger.SetFormatter(CustomLogger{
		defaultField: appName,
		formatter: &customLogFormatter{logrus.TextFormatter{
			DisableTimestamp: false,
			DisableColors:    false,
			ForceColors:      true,
			ForceQuote:       true,
			FullTimestamp:    true,
			QuoteEmptyFields: true,
		}},
	})

	if level == "" {
		level = "debug"
	}
	l, err := logrus.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}
	Logger.Level = l
	return Logger
}

// StructuredLoggerEntry is a logrus.FieldLogger.
type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

// Panic prints stack trace
func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}
