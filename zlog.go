package zlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log ...
type Log struct {
	Logger      *zap.Logger
	SugarLogger *zap.SugaredLogger
}

// Init ...
func Init(outputdir, perfix, level string) *Log {
	logLevel, err := parseLevel(level)
	if err != nil {
		fmt.Printf("unknown log level:[%v]\n", err)
	}

	//↓↓为分文件写日志内容的方式
	// var infopath = perfix + "_info.log"
	// var errorpath = perfix + "_error.log"
	//↓↓为单文件写日志内容的方式
	var filename = perfix + ".log"

	_, err = os.Stat(outputdir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(outputdir, os.ModePerm)
			if err != nil {
				fmt.Printf("mkdir failed![%v]\n", err)
			}
		}
	}

	//日志配置参数
	config := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "ts",
		CallerKey:     "file", //"caller"
		StacktraceKey: "trace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	}

	// 实现判断日志等级的interface
	// ↓↓分文件写内容的方式
	// infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
	// 	return lvl < zapcore.WarnLevel && lvl >= logLevel
	// })

	// warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
	// 	return lvl >= zapcore.WarnLevel && lvl >= logLevel
	// })
	//↓↓单文件写内容的方式
	fileLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= logLevel
	})

	// 获取 info、warn等日志文件的io.Writer 抽象 getWriter() 在下方实现
	// infoHook := getWriter(outputdir, infopath)
	// errorHook := getWriter(outputdir, errorpath)
	fileHook := getWriter(outputdir, filename)

	// 最后创建具体的Logger
	core := zapcore.NewTee(
		//↓↓分文件写内容的方式
		// zapcore.NewCore(zapcore.NewConsoleEncoder(config), zapcore.AddSync(infoHook), infoLevel),  //Info以下文件输出
		// zapcore.NewCore(zapcore.NewConsoleEncoder(config), zapcore.AddSync(errorHook), warnLevel), //Warn以上文件输出
		//↓↓单文件写内容的方式
		zapcore.NewCore(zapcore.NewConsoleEncoder(config), zapcore.AddSync(fileHook), fileLevel),                               //单文件写全部日志，filelevel控制写入内容级别
		zapcore.NewCore(zapcore.NewConsoleEncoder(config), zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), fileLevel), //终端输出
	)

	// 需要传入 zap.AddCaller() 才会显示打日志点的文件名和行数, 有点小坑
	Logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel))
	SugarLogger := Logger.Sugar()
	defer Logger.Sync()

	logobj := &Log{
		Logger:      Logger,
		SugarLogger: SugarLogger,
	}
	return logobj
}

func getWriter(outputdir, filename string) io.Writer {
	// 生成rotatelogs的Logger 实际生成的文件名 demo.log.YYmmddHH
	// demo.log是指向最新日志的链接
	// 保存7天内的日志，每1小时(整点)分割一次日志
	hook, err := rotatelogs.New(
		// 没有使用go风格反人类的format格式
		outputdir+filename+".%Y%m%d",
		rotatelogs.WithLinkName(filename),
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour*24),
	)
	if err != nil {
		panic(err)
	}
	return hook
}

func parseLevel(s string) (zapcore.Level, error) {
	//格式化Log级别参数
	s = strings.ToLower(s)
	switch s {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "dpanic":
		return zapcore.DPanicLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		err := errors.New("unknown log level")
		return zapcore.DebugLevel, err
	}
}

//FormatString ...
func (l *Log) FormatString(key, value string) zapcore.Field {
	return zap.String(key, value)
}
