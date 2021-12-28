//Package log initilizes all the log functionalities
package log

import (
	"os"

	"github.com/sanservices/kit/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var lg *zap.SugaredLogger

//Initialize prepares basic items for Log process
func Initialize(cfg *config.Info) {
	highPriorityOutput := zapcore.Lock(os.Stdout)
	stdoutEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	core := zapcore.NewTee(
		zapcore.NewCore(stdoutEncoder, highPriorityOutput, highPriorityChecker(cfg)),
	)

	lg = zap.New(core).Sugar()

}

//Logger returns an instance of current logger package
func Logger() *zap.SugaredLogger {
	return lg
}

func lowPriorityChecker(cfg *config.Info) zap.LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		if lvl >= zapcore.ErrorLevel || cfg.LogLevel == "debug" {
			return false
		}
		return true
	}
}

func highPriorityChecker(cfg *config.Info) zap.LevelEnablerFunc {
	return func(level zapcore.Level) bool {
		return !lowPriorityChecker(cfg)(level)
	}
}
