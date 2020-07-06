/**
 * @Author: lyszhang
 * @Email: zhangliang@link-logis.com
 * @Date: 2020/5/28 1:54 PM
 */

package logmgr

import (
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"time"
)

func newWriter(name, appname, suffix string) *rotatelogs.RotateLogs {
	// 归档日志路径
	backupPath := path.Join(name, "backup")
	if _, err := os.Stat(backupPath); err != nil {
		exec.Command("mkdir", "-p", backupPath).Output()
	}

	writer, err := rotatelogs.New(
		path.Join(name, "backup", appname+suffix)+".%Y%m%d.%H",
		// WithLinkName为最新的日志建立软连接，以方便随着找到当前日志文件
		rotatelogs.WithLinkName(path.Join(name, appname+suffix)),

		// WithRotationTime设置日志分割的时间，这里设置为一天分割一次
		rotatelogs.WithRotationTime(time.Hour*24),

		// WithMaxAge设置文件清理前的最长保存时间，
		rotatelogs.WithMaxAge(time.Hour*24*30),
	)
	if err != nil {
		log.Errorf("config local file system for logger error: %v", err)
		return nil
	}
	return writer
}

func Init(base, appName string, usage Usage, logType LogType, chaincodeTrace bool) {
	basePath := path.Join(base, GoNamespace(), appName, GoDeployment())
	// Files
	wrAll := newWriter(basePath, appName, "-stdout.log")
	wrInfo := newWriter(basePath, appName, "-info.log")
	wrWarn := newWriter(basePath, appName, "-warn.log")
	wrError := newWriter(basePath, appName, "-error.log")
	wrRuntime := newWriter(path.Join(basePath, "elk"), appName, "-runtime.log")

	// formatter global
	log.SetLevel(log.TraceLevel)
	log.SetReportCaller(false)

	// New hooks
	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		log.InfoLevel:  wrInfo,
		log.WarnLevel:  wrWarn,
		log.ErrorLevel: wrError,
	}, &log.TextFormatter{DisableColors: true})

	lfsAllHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: wrAll,
		log.InfoLevel:  wrAll,
		log.WarnLevel:  wrAll,
		log.ErrorLevel: wrAll,
		log.FatalLevel: wrAll,
		log.PanicLevel: wrAll,
	}, &log.TextFormatter{DisableColors: true})

	lfsRuntimeHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: wrRuntime,
		log.InfoLevel:  wrRuntime,
		log.WarnLevel:  wrRuntime,
		log.ErrorLevel: wrRuntime,
		log.FatalLevel: wrRuntime,
		log.PanicLevel: wrRuntime,
	}, defaultLogFormatter(appName, usage, logType))

	// Add the hook
	log.AddHook(lfsHook)
	log.AddHook(lfsAllHook)
	log.AddHook(lfsRuntimeHook)

	if chaincodeTrace {
		wrCC := newWriter(path.Join(basePath, "elk"), appName, "-chaincode.log")
		lfsChaincodeHook := lfshook.NewHook(lfshook.WriterMap{
			log.TraceLevel: wrCC,
		}, chaincodeLogFormatter(appName, ChaincodeUsage, SvcType))

		log.AddHook(lfsChaincodeHook)
	}
}

func InitFabricLog(base, appName string) {
	basePath := path.Join(base, GoNamespace(), appName, GoDeployment())
	// Files
	wrAll := newWriter(basePath, appName, "-stdout.log")
	wrInfo := newWriter(basePath, appName, "-info.log")
	wrWarn := newWriter(basePath, appName, "-warn.log")
	wrError := newWriter(basePath, appName, "-error.log")
	wrRuntime := newWriter(path.Join(basePath, "elk"), appName, "-runtime.log")

	// formatter global
	log.SetLevel(log.TraceLevel)
	log.SetReportCaller(true)

	// New hooks
	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		log.InfoLevel:  wrInfo,
		log.WarnLevel:  wrWarn,
		log.ErrorLevel: wrError,
	}, &log.TextFormatter{DisableColors: true})

	lfsAllHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: wrAll,
		log.InfoLevel:  wrAll,
		log.WarnLevel:  wrAll,
		log.ErrorLevel: wrAll,
		log.FatalLevel: wrAll,
		log.PanicLevel: wrAll,
	}, &log.TextFormatter{DisableColors: true})

	lfsRuntimeHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: wrRuntime,
		log.InfoLevel:  wrRuntime,
		log.WarnLevel:  wrRuntime,
		log.ErrorLevel: wrRuntime,
		log.FatalLevel: wrRuntime,
		log.PanicLevel: wrRuntime,
	}, fabricLogFormatter())

	// Add the hook
	log.AddHook(lfsHook)
	log.AddHook(lfsAllHook)
	log.AddHook(lfsRuntimeHook)
}
