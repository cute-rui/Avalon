package log

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Logger *zap.SugaredLogger
var LogConf LoggerConfig

func InitLogger() {
	var coreArr []zapcore.Core

	//获取编码器
	var encoderConfig zapcore.EncoderConfig
	if LogConf.IsProduction {
		encoderConfig = zap.NewProductionEncoderConfig() //NewJSONEncoder()输出json格式，NewConsoleEncoder()输出普通文本格式
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig() //NewJSONEncoder()输出json格式，NewConsoleEncoder()输出普通文本格式
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        //指定时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder //按级别显示不同颜色，不需要的话取值zapcore.CapitalLevelEncoder就可以了
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	//日志级别
	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //error级别
		return lev >= zap.ErrorLevel
	})
	midPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //info级别
		return lev < zap.ErrorLevel && lev >= zap.DebugLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //debug级别,debug级别是最低的
		return lev <= zap.DebugLevel
	})

	if LogConf.ErrorFile.Enabled {
		//error文件writeSyncer
		errorFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   LogConf.ErrorFile.Filename,   //日志文件存放目录
			MaxSize:    LogConf.ErrorFile.MaxSize,    //文件大小限制,单位MB
			MaxBackups: LogConf.ErrorFile.MaxBackups, //最大保留日志文件数量
			MaxAge:     LogConf.ErrorFile.MaxAge,     //日志文件保留天数
			Compress:   LogConf.ErrorFile.Compress,   //是否压缩处理
		})
		errorFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(errorFileWriteSyncer, zapcore.AddSync(os.Stdout)), highPriority) //第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志

		coreArr = append(coreArr, errorFileCore)
	}

	if LogConf.InfoFile.Enabled {
		//info文件writeSyncer
		infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   LogConf.InfoFile.Filename,   //日志文件存放目录
			MaxSize:    LogConf.InfoFile.MaxSize,    //文件大小限制,单位MB
			MaxBackups: LogConf.InfoFile.MaxBackups, //最大保留日志文件数量
			MaxAge:     LogConf.InfoFile.MaxAge,     //日志文件保留天数
			Compress:   LogConf.InfoFile.Compress,   //是否压缩处理
		})
		infoFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), midPriority) //第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
		coreArr = append(coreArr, infoFileCore)
	}

	if LogConf.DebugFile.Enabled {
		debugFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   LogConf.DebugFile.Filename,   //日志文件存放目录
			MaxSize:    LogConf.DebugFile.MaxSize,    //文件大小限制,单位MB
			MaxBackups: LogConf.DebugFile.MaxBackups, //最大保留日志文件数量
			MaxAge:     LogConf.DebugFile.MaxAge,     //日志文件保留天数
			Compress:   LogConf.DebugFile.Compress,   //是否压缩处理
		})
		debugFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(debugFileWriteSyncer, zapcore.AddSync(os.Stdout)), lowPriority) //第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
		coreArr = append(coreArr, debugFileCore)
	}

	Logger = zap.New(zapcore.NewTee(coreArr...), zap.AddCaller()).Sugar()
}

type LoggerConfig struct {
	IsProduction bool
	ErrorFile    FileConfig
	InfoFile     FileConfig
	DebugFile    FileConfig
}

type FileConfig struct {
	Enabled    bool
	Filename   string //日志文件存放目录，如果文件夹不存在会自动创建
	MaxSize    int    //文件大小限制,单位MB
	MaxBackups int    //最大保留日志文件数量
	MaxAge     int    //日志文件保留天数
	Compress   bool   //是否压缩处理
}
