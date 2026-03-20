/*
   Copyright @ 2021 bocloud <fushaosong@beyondcent.com>.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

   Original file: https://gitee.com/bocloud-open-source/carina/blob/v0.9.1/utils/log/logger.go
*/

package log

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logPath = "/var/log/openFuyao/bke.log"
	DEBUG   = "DEBUG"
	INFO    = "INFO"
	WARN    = "WARN"
	ERROR   = "ERROR"
	NIL     = ""
	HINT    = "HINT"
)

var sugareLogger *zap.SugaredLogger

// logPath 日志文件路径
// logLevel 日志级别 debug/info/warn/error
// maxSize 单个文件大小,MB
// maxBackups 保存的文件个数
// maxAge 保存的天数， 没有的话不删除
// compress 压缩
// jsonFormat 是否输出为json格式
// AddCaller 显示调用者
// logInConsole 是否同时输出到控制台

func init() {
	hook := lumberjack.Logger{
		Filename:   logPath, // 日志文件路径
		MaxSize:    30,      // megabytes
		MaxBackups: 3,       // 最多保留300个备份
		Compress:   false,   // 是否压缩 disabled by default
	}

	hook.MaxAge = 1

	var syncer zapcore.WriteSyncer
	syncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook))

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	level := zap.InfoLevel
	if os.Getenv("DEBUG") != "" {
		level = zap.DebugLevel
	}
	core := zapcore.NewCore(
		encoder,
		syncer,
		level,
	)

	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugareLogger = log.Sugar()
}

func Debug(args ...interface{}) {
	sugareLogger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	sugareLogger.Debugf(template, args...)
}

func Info(args ...interface{}) {
	sugareLogger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	sugareLogger.Infof(template, args...)
}

func Warn(args ...interface{}) {
	sugareLogger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	sugareLogger.Warnf(template, args...)
}

func Error(args ...interface{}) {
	sugareLogger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	sugareLogger.Errorf(template, args...)
}

func Panic(args ...interface{}) {
	sugareLogger.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	sugareLogger.Panicf(template, args...)
}

func Fatal(args ...interface{}) {
	sugareLogger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	sugareLogger.Fatalf(template, args...)
}

// BKEFormat standard output
func BKEFormat(level, msg string) {
	if level == "" {
		fmt.Println(fmt.Sprintf("[bke] %s", msg))
	} else {
		fmt.Println(fmt.Sprintf("[bke][%s][%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, msg))
	}
}
