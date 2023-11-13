package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	Developemnt mode = iota
	Production
)

const (
	Debug level = iota
	Info
	Warn
	Error
	Default = Info
)

type (
	mode  int
	level int
)

func GetGlobal() *Logger {
	return &Logger{
		logger: zap.L().Sugar(),
	}
}

func New(name string, mode mode, minLevel level, filePaths ...string) (*Logger, error) {
	var logger *zap.SugaredLogger
	var err error
	switch mode {
	case Production:
		logger, err = newLogger(name, minLevel, productionEncoderConfig(), filePaths...)
	default:
		logger, err = newLogger(name, minLevel, developmentEncoderConfig(), filePaths...)
	}

	if err != nil {
		return nil, err
	}

	return &Logger{
		logger: logger,
	}, nil
}

func ReplaceGlobals(logger *Logger) {
	zap.ReplaceGlobals(logger.logger.Desugar())
}

func newLogger(name string, minLevel level, cfg zapcore.EncoderConfig, paths ...string) (*zap.SugaredLogger, error) {
	var cores []zapcore.Core

	cores = append(cores, consoleCore(minLevel, cfg))

	if len(paths) != 0 {
		fileCore, err := pathCore(minLevel, cfg, paths...)
		if err != nil {
			return nil, err
		}
		cores = append(cores, fileCore)
	}

	logger := zap.New(zapcore.NewTee(cores...))

	slogger := logger.Named(name).Sugar()

	return slogger, nil
}

func pathCore(level level, encCfg zapcore.EncoderConfig, paths ...string) (zapcore.Core, error) {
	encoder := zapcore.NewJSONEncoder(encCfg)

	files := make([]zapcore.WriteSyncer, len(paths))
	for _, path := range paths {
		files = append(files, zapcore.AddSync(&lumberjack.Logger{
			Filename: path,
			MaxSize:  1000,
		}))
	}

	writer := zapcore.NewMultiWriteSyncer(files...)

	return zapcore.NewCore(encoder, writer, zapLevel(level)), nil
}

func consoleCore(minLevel level, encCfg zapcore.EncoderConfig) zapcore.Core {
	encoder := zapcore.NewConsoleEncoder(encCfg)
	outputCore := zapcore.NewCore(encoder, os.Stdout, infoPriority(minLevel))
	errCore := zapcore.NewCore(encoder, os.Stderr, errorPriority(minLevel))

	return zapcore.NewTee(outputCore, errCore)
}

func developmentEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewDevelopmentEncoderConfig()

	cfg.NameKey = "logger"
	cfg.EncodeName = zapcore.FullNameEncoder

	cfg.MessageKey = "message"

	cfg.TimeKey = "timestamp"
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder

	cfg.LevelKey = "level"
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	cfg.CallerKey = "caller"
	cfg.EncodeCaller = zapcore.ShortCallerEncoder

	cfg.StacktraceKey = "stacktrace"

	cfg.FunctionKey = zapcore.OmitKey

	return cfg
}

func productionEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()

	cfg.NameKey = "logger"
	cfg.EncodeName = zapcore.FullNameEncoder

	cfg.MessageKey = "msg"

	cfg.TimeKey = "ts"
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder

	cfg.LevelKey = "lv"
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder

	cfg.CallerKey = "caller"
	cfg.EncodeCaller = zapcore.ShortCallerEncoder

	cfg.StacktraceKey = "stacktrace"

	cfg.FunctionKey = zapcore.OmitKey

	return cfg
}

func infoPriority(minLevel level) zap.LevelEnablerFunc {
	minLV := zapLevel(minLevel)
	return func(lv zapcore.Level) bool {
		return lv >= minLV && lv < zap.ErrorLevel
	}
}

func errorPriority(minLevel level) zap.LevelEnablerFunc {
	minLV := zapLevel(minLevel)
	return func(lv zapcore.Level) bool {
		return lv >= minLV && lv >= zap.ErrorLevel
	}
}

func zapLevel(level level) zapcore.Level {
	switch level {
	case Debug:
		return zap.DebugLevel
	case Error:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}
