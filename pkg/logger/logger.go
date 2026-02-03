package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitConsoleLogger() {
	var err error
	Log, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

func InitFileLogger() {

	logdir := filepath.Join("..", "internal", "app", "log")
	logPath := filepath.Join(logdir, "app.log")

	if Log != nil {
		_ = Log.Sync() // сброс логгер, если были какие то другие
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	Log = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(file),
		zap.InfoLevel,
	))

}

func init() {
	InitConsoleLogger()
}
