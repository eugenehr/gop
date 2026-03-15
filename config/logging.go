package config

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	prettylogger "github.com/nentgroup/slog-prettylogger"
	sloglogstash "github.com/samber/slog-logstash/v2"
	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog/v2"
)

// LogLevel log level: debug, info, error.
//
// Уровень логирования: debug, info, warning, error.
type LogLevel string

// LogFormat log format: json or text.
//
// Формат вывода логов: json(-lines) или обычный текст.
type LogFormat string

// LogExtra log extra keys.
//
// Дополнительные атрибуты для присоединения к записям в log-файле.
type LogExtra map[string]string

// LogUrl url to remote logger.
//
// URL для отправки логов rsyslog и logstash.
type LogUrl string

// LocalLogger local log configuration.
//
// Настройки локального логирования.
type LocalLogger struct {
	Level LogLevel `yaml:"level"`
	Extra LogExtra `yaml:"extra"`
}

// FileLogger log configuration with output to file.
//
// Настройки логирования в локальный файл.
type FileLogger struct {
	LocalLogger `yaml:",inline"`
	File        string    `yaml:"file"`
	Format      LogFormat `yaml:"format"`
	f           *os.File
}

// RemoteLogger log configuration with output to rsyslog or logstash.
//
// Настройки отправки логов в удаленный rsyslog и logstash.
type RemoteLogger struct {
	LocalLogger `yaml:",inline"`
	Url         LogUrl `yaml:"url"`
	conn        net.Conn
}

type SyslogLogger struct {
	RemoteLogger `yaml:",inline"`
}

type LogstashLogger struct {
	RemoteLogger `yaml:",inline"`
}

// LoggerConfig holds application logging configuration.
//
// Настройки логирования приложения.
type LoggerConfig struct {
	Console  LocalLogger    `yaml:"console"`
	File     FileLogger     `yaml:"file"`
	Syslog   SyslogLogger   `yaml:"syslog"`
	Logstash LogstashLogger `yaml:"logstash"`
}

func (l LogLevel) parse() slog.Level {
	s := strings.ToLower(string(l))
	switch s {
	case "debug":
		return slog.LevelDebug
	case "", "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		fmt.Printf("Unknown log level '%s'. Must be 'debug', 'info', 'warn', 'warning' or 'error'. Switching to 'info'\n", s)
		return slog.LevelInfo
	}
}

func (f LogFormat) isJson() bool {
	s := strings.ToLower(string(f))
	switch s {
	case "json":
		return true
	case "", "text":
		return false
	default:
		fmt.Printf("Unknown log format '%s'. Must be 'json' or 'text'. Switching to 'text'\n", s)
		return false
	}
}

func (e *LogExtra) addExtra(handler slog.Handler) slog.Handler {
	if e != nil {
		var attrs = make([]slog.Attr, 0, len(*e))
		for key, value := range *e {
			attrs = append(attrs, slog.Attr{
				Key:   key,
				Value: slog.StringValue(value),
			})
		}
		handler = handler.WithAttrs(attrs)
	}
	return handler
}

func (u LogUrl) dial() (net.Conn, error) {
	parsed, err := url.Parse(string(u))
	if err != nil {
		return nil, err
	}
	return net.Dial(parsed.Scheme, parsed.Host)
}

const logFilePerm = os.FileMode(0600)

func (f *FileLogger) newHandler(ctx context.Context, wg *sync.WaitGroup) (handler slog.Handler, err error) {
	if f.File != "" {
		f.f, err = os.OpenFile(f.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, logFilePerm)
		if err != nil {
			return nil, err
		}
		opts := slog.HandlerOptions{
			Level: f.Level.parse(),
		}
		if f.Format.isJson() {
			handler = slog.NewJSONHandler(f.f, &opts)
		} else {
			handler = slog.NewTextHandler(f.f, &opts)
		}
		handler = f.Extra.addExtra(handler)
	}
	wg.Go(func() {
		<-ctx.Done()
		if f.f != nil {
			_ = f.f.Close()
			fmt.Println("file closed")
		}
	})
	return handler, nil
}

func (f *SyslogLogger) newHandler(ctx context.Context, wg *sync.WaitGroup) (handler slog.Handler, err error) {
	if f.Url != "" {
		f.conn, err = f.Url.dial()
		if err == nil {
			handler = f.Extra.addExtra(slogsyslog.Option{
				Level:  f.Level.parse(),
				Writer: f.conn,
			}.NewSyslogHandler())
		}
	}
	wg.Go(func() {
		<-ctx.Done()
		if f.conn != nil {
			_ = f.conn.Close()
			fmt.Println("syslog connection closed")
		}
	})
	return handler, nil
}

func (f *LogstashLogger) newHandler(ctx context.Context, wg *sync.WaitGroup) (handler slog.Handler, err error) {
	if f.Url != "" {
		f.conn, err = f.Url.dial()
		if err == nil {
			opt := sloglogstash.Option{
				Level: f.Level.parse(),
				Conn:  f.conn,
			}
			handler = f.Extra.addExtra(opt.NewLogstashHandler())
		}
	}
	wg.Go(func() {
		<-ctx.Done()
		if f.conn != nil {
			_ = f.conn.Close()
			fmt.Println("logstash connection closed")
		}
	})
	return handler, nil
}

// NewHandler creates a [slog.Handler] for the application.
// The parameters ctx and wg are used to correctly close the log file and connections to rsyslog and logstash backends.
//
// Функция создает обработчик логов для приложения. Параметры ctx и wg используются для корректного
// закрытия файлов и подключений к rsyslog и logstash серверам.
func (c LoggerConfig) NewHandler(ctx context.Context, wg *sync.WaitGroup) (handler slog.Handler, err error) {
	var handlers []slog.Handler
	// local logging to console
	handler = prettylogger.NewHandler(os.Stdout, prettylogger.HandlerOptions{
		SlogOpts:   slog.HandlerOptions{Level: c.Console.Level.parse()},
		TimeFormat: time.DateTime,
		NoColor:    true,
	})
	handlers = append(handlers, c.Console.Extra.addExtra(handler))
	// local logging to file
	handler, err = c.File.newHandler(ctx, wg)
	if err != nil {
		return nil, err
	}
	if handler != nil {
		handlers = append(handlers, handler)
	}
	// remote logging with syslog
	handler, err = c.Syslog.newHandler(ctx, wg)
	if err != nil {
		return nil, err
	}
	if handler != nil {
		handlers = append(handlers, handler)
	}
	// remote logging with logstash
	handler, err = c.Logstash.newHandler(ctx, wg)
	if err != nil {
		return nil, err
	}
	if handler != nil {
		handlers = append(handlers, handler)
	}
	return slogmulti.Fanout(handlers...), nil
}
