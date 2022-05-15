package poolgo

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/photowey/poolgo/color"
)

const (
	InfoLevel = iota
	WarnLevel
	ErrorLevel
)

var (
	sequenceNo uint64

	_ Logger = (*logger)(nil)
)

type Record struct {
	ID       string
	Level    string
	Message  string
	Filename string
	LineNo   int
}

type Logger interface {
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)
}

type logger struct {
	lock           sync.Mutex
	output         io.Writer
	loggerTemplate *template.Template
}

func (log *logger) levelTag(level int) string {
	switch level {
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "INFO"
	}
}

func (log *logger) colorLevel(level int) string {
	switch level {
	case InfoLevel:
		return color.BlueBold(log.levelTag(level))
	case ErrorLevel:
		return color.RedBold(log.levelTag(level))
	case WarnLevel:
		return color.YellowBold(log.levelTag(level))
	default:
		return color.BlueBold(log.levelTag(level))
	}
}

func (log *logger) Infof(template string, args ...any) {
	log.log(InfoLevel, template, args)
}

func (log *logger) Warnf(template string, args ...any) {
	log.log(WarnLevel, template, args)
}

func (log *logger) Errorf(template string, args ...any) {
	log.log(ErrorLevel, template, args)
}

func (log *logger) log(level int, message string, args ...any) {
	log.lock.Lock()
	defer log.lock.Unlock()

	record := Record{
		ID:      fmt.Sprintf("%04d", atomic.AddUint64(&sequenceNo, 1)),
		Level:   log.colorLevel(level),
		Message: fmt.Sprintf(message, args...),
	}

	_ = log.loggerTemplate.Execute(log.output, record)
}

func NewLogger(w io.Writer) Logger {
	loggerFormat := `{{Now "2006-01-02 15:04:05.999"}} {{.Level}} ▶ {{.ID}} {{.Message}}{{EndLine}}`
	funcs := template.FuncMap{
		"Now":     Now,
		"EndLine": EndLine,
	}
	loggerTemplate, _ := template.New("logger").Funcs(funcs).Parse(loggerFormat)

	return &logger{
		loggerTemplate: loggerTemplate,
		output:         color.NewColorWriter(w),
	}
}

func Now(layout string) string {
	return time.Now().Format(layout)
}

func EndLine() string {
	return "\n"
}
