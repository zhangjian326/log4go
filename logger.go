/*
日志级别优先级：

 Debug < Info < Notice < Warn < Fatal
*/

package log4go

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelNotice
	LevelWarn
	LevelFatal
)

const (
	LogDefCallDepth = 2
	FileDefMaxLines = 1000000
	FileDefMaxSize  = 1 << 28 //256 MB
	FileDefMaxDays  = 7
	MaxFileNum      = 1000
)

var (
	levelTextArray = []string{
		LevelDebug:  "DEBUG",
		LevelInfo:   "INFO",
		LevelNotice: "NOTICE",
		LevelWarn:   "WARN",
		LevelFatal:  "FATAL",
	}
)

type Logger struct {
	wg                  sync.WaitGroup
	level               int
	service             string
	asynchronous        bool
	enableFuncCallDepth bool
	logFuncCallDepth    int
	msgChan             chan *logMsg
	signalChan          chan string
	hostname            string
	fileLog             *FileLog
}

type logMsg struct {
	level int
	msg   string
	when  time.Time
}

var logMsgPool *sync.Pool

func NewLogger(channelLen int, service string) *Logger {

	log := &Logger{
		level:               LevelDebug,
		service:             service,
		asynchronous:        true,
		enableFuncCallDepth: false,
		logFuncCallDepth:    LogDefCallDepth,
		msgChan:             make(chan *logMsg, channelLen),
		signalChan:          make(chan string, 1),
		fileLog:             newFileLog(),
	}

	logMsgPool = &sync.Pool{
		New: func() interface{} {
			return &logMsg{}
		},
	}

	log.wg.Add(1)
	go log.startLogger()

	return log
}

func isDir(filename string) (bool, error) {

	if len(filename) <= 0 {
		return false, fmt.Errorf("invalid dir")
	}

	stat, err := os.Stat(filename)
	if err != nil {
		return false, fmt.Errorf("invalid path:" + filename)
	}

	if !stat.IsDir() {
		return false, nil
	}

	return true, nil
}

func (log *Logger) Open(filepath, filename, level string) error {
	isDir, err := isDir(filepath)
	if err != nil || !isDir {
		err = os.MkdirAll(filepath, 0755)
		if err != nil {
			return fmt.Errorf("Mkdir failed, err:%v", err)
		}
	}

	log.level = log.levelFromStr(level)
	log.fileLog.filename = filename
	log.fileLog.filepath = filepath
	hostname, _ := os.Hostname()
	log.hostname = hostname

	return log.fileLog.startLogger()
}

func (log *Logger) levelFromStr(level string) int {
	resultLevel := LevelDebug
	lower := strings.ToLower(level)
	switch lower {
	case "debug":
		resultLevel = LevelDebug
	case "info":
		resultLevel = LevelInfo
	case "notice":
		resultLevel = LevelNotice
	case "warn":
		resultLevel = LevelWarn
	case "fatal":
		resultLevel = LevelFatal
	default:
		resultLevel = LevelInfo
	}
	return resultLevel
}

func (log *Logger) SetLevel(level string) *Logger {
	log.level = log.levelFromStr(level)
	return log
}

func (log *Logger) GetLevel() string {
	return levelTextArray[log.level]
}

func (log *Logger) SetFuncCallDepth(depth int) *Logger {
	log.logFuncCallDepth = depth
	return log
}

func (log *Logger) EnableFuncCallDepth(flag bool) *Logger {
	log.enableFuncCallDepth = flag
	return log
}

func (log *Logger) SetMaxDays(day int64) *Logger {
	log.fileLog.maxDays = day
	return log
}

func (log *Logger) GetMaxDays() int64 {
	return log.fileLog.maxDays
}

func (log *Logger) SetMaxLines(line int) *Logger {
	log.fileLog.maxLines = line
	return log
}

func (log *Logger) GetMaxLines() int {
	return log.fileLog.maxLines
}

func (log *Logger) SetMaxSize(size int) *Logger {
	log.fileLog.maxSize = size
	return log
}

func (log *Logger) GetMaxSize() int {
	return log.fileLog.maxSize
}

func (log *Logger) EnableRotate(flag bool) *Logger {
	log.fileLog.rotate = flag
	return log
}

func (log *Logger) EnableDaily(flag bool) *Logger {
	log.fileLog.daily = flag
	return log
}

func (log *Logger) GetHost() string {
	return log.hostname
}

func (log *Logger) startLogger() {
	gameOver := false
	for {
		select {
		case lm := <-log.msgChan:
			log.writeToFile(lm.when, lm.msg, lm.level)
			logMsgPool.Put(lm)
		case sg := <-log.signalChan:
			// Now should only send "flush" or "close" to log.signalChan
			log.flush()
			if sg == "close" {
				log.fileLog.Destroy()
				gameOver = true
			}
			log.wg.Done()
		}
		if gameOver {
			break
		}
	}
}

func (log *Logger) Flush() {
	if log.asynchronous {
		log.signalChan <- "flush"
		log.wg.Wait() //To wait signalChan execute log.flush()
		log.wg.Add(1) //for call Flush again
		return
	}
	log.flush()
}

// Close close logger, flush all chan data and destroy all adapters in BeeLogger.
func (log *Logger) Close() {
	if log.asynchronous {
		log.signalChan <- "close"
		log.wg.Wait() //To wait signalChan execute log.flush() then close file
	} else {
		log.flush()
		log.fileLog.Destroy()
	}
	close(log.msgChan)
	close(log.signalChan)
}

func (log *Logger) flush() {
	for {
		if len(log.msgChan) > 0 {
			lm := <-log.msgChan
			log.writeToFile(lm.when, lm.msg, lm.level)
			logMsgPool.Put(lm)
			continue
		}
		break
	}
	log.fileLog.Flush()
}

func (log *Logger) writeMsg(logLevel int, msg string) {
	when := time.Now()
	if log.enableFuncCallDepth {
		function := ""
		dir := ""
		pc, file, line, ok := runtime.Caller(log.logFuncCallDepth)
		if !ok {
			dir = "???"
			file = "???"
			function = "???"
			line = 0
		} else {
			//function = runtime.FuncForPC(pc).Name()
			_, function = path.Split(runtime.FuncForPC(pc).Name())
		}

		dir, filename := path.Split(file)
		msg = fmt.Sprintf("[dir:%s file:%s func:%s line:%d] %s", dir, filename, function, line, msg)
		//msg = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "] " + msg
	}

	if log.asynchronous {
		lm := logMsgPool.Get().(*logMsg)
		lm.level = logLevel
		lm.msg = msg
		lm.when = when
		log.msgChan <- lm
	} else {
		log.writeToFile(when, msg, logLevel)
	}
}

func (log *Logger) writeToFile(when time.Time, msg string, level int) {
	log.fileLog.WriteMsg(log.service, log.hostname, when, msg, level)
}

// DEBUG Log DEBUG level message.
func (log *Logger) Debug(format string, v ...interface{}) {
	if LevelDebug < log.level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	log.writeMsg(LevelDebug, msg)
}

// INFO Log INFO level message.
func (log *Logger) Info(format string, v ...interface{}) {
	if LevelInfo < log.level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	log.writeMsg(LevelInfo, msg)
}

// Notice Log Notice level message.
func (log *Logger) Notice(format string, v ...interface{}) {
	if LevelNotice < log.level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	log.writeMsg(LevelNotice, msg)
}

// Warn Log Warn level message.
func (log *Logger) Warn(format string, v ...interface{}) {
	if LevelWarn < log.level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	log.writeMsg(LevelWarn, msg)
}

// Fatal Log Fatal level message.
func (log *Logger) Fatal(format string, v ...interface{}) {
	if LevelFatal < log.level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	log.writeMsg(LevelFatal, msg)
}
