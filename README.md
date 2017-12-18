
# logger
    import "github.com/xkeyideal/log4go"

本日志库参考了beego/logs中的file log的实现，并在此基础上做了如下改进:
1. 日志的参数设置全部使用函数调用的方式实现，不采用json方式
2. 将Debug, Info, Notice级别的日志定义为常规日志，存入log_filename.log文件中
3. 将Warn, Fatal级别的日志定义为错误日志，存入log_filename.log.wf文件中
4. 日志的设置更加方便灵活
5. 为日志的输出添加服务名和主机名，更加便捷的日志区分

xklog日志库支持5种日志级别：
```
	1）Debug
	2）Info
	3）Notice
	4）Warn
	5）Fatal
```

日志级别优先级：

	Debug < Info < Notice < Warn < Fatal
即如果定义日志级别为Debug：则Debug、Info、Notice、Warn、Fatal等级别的日志都会输出；反之，如果定义日志级别为Info，则Debug不会输出，其它级别的日志都会输出。

完整示例代码，请参考：github.com/xkeyideal/log4go/example/example.go

## Constants
``` go
const (
	LogDefCallDepth = 2                     //默认的代码栈个数
	FileDefMaxLines   = 1000000             //文件默认的最大行数，超过即写入新文件
	FileDefMaxSize    = 1 << 28             //256 MB 文件默认的最大存储，超过即写入新文件
	FileDefMaxDays    = 7                   //文件保存的天数，超过即删除
	MaxFileNum        = 1000                //文件保存的天数内可写的文件数目，超过则报错
)
```

## Variables
``` go
var (
	levelTextArray = []string{
		LevelDebug:  "DEBUG",
		LevelInfo:   "INFO",
		LevelNotice: "NOTICE",
		LevelWarn:   "WARN",
		LevelFatal:  "FATAL",
	}
)
```


## type Logger
``` go
type Logger struct {
	wg                  sync.WaitGroup
	level               int
	service             string              //服务名
	asynchronous        bool                //是否启动日志异步写入，默认不开启
	enableFuncCallDepth bool                //是否打印调用日志的文件名和函数名，默认开启
	logFuncCallDepth    int                 //调用日志的文件名和函数名的栈深度，默认为XkLogDefCallDepth
	msgChan             chan *logMsg        //日志异步写入的channel
	signalChan          chan string         //信号channel
	hostname            string              //日志所在机器的主机名
	fileLog             *FileLog            //写入日志的结构体
}

type logMsg struct {                        //日志异步写入的结构体
	level int
	msg   string
	when  time.Time
}

var logMsgPool *sync.Pool                   //日志异步写入的对象池
```

## type FileLog
``` go
type FileLog struct {
	sync.Mutex                             //锁

	maxLines               int             //文件可写的最大行数
	normalMaxLinesCurLines int             //常规日志当前写入文件的行数
	errMaxLinesCurLines    int             //错误日志当前写入文件的行数

	// Rotate at size
	maxSize              int               //文件可写的最大存储空间
	normalMaxSizeCurSize int               //常规日志当前写入文件的存储空间
	errMaxSizeCurSize    int               //错误日志当前写入文件的存储空间

	// Rotate daily
	daily         bool                     //文件是否按天记录，默认开启
	maxDays       int64                    //历史日志文件最多保留的天数，默认FileDefMaxDays天
	dailyOpenDate int                      //日志功能开启的日期

	rotate bool                            //是否开启日志文件Rotate存储

	file     *os.File                      //常规日志文件的描述符
	errFile  *os.File                      //错误日志文件的描述符
	filepath string                        //日志文件的存储目录
	filename string                        //日志文件的名称
}
```

### func NewXkLog
``` go
//初始化Logger，需要传入日志异步写入的msgChan的大小channelLen，该日志所在项目的名称service
func NewLogger(channelLen int, service string) *Logger
```
生成一个日志实例，msgChan的大小为1000, service用来标识业务的服务名log_test。
比如：logger := Logger.NewLogger(1000,"log_test")

### func (log \*Logger) Open
``` go
//设置日志文件的存储目录,文件名和日志的级别
func (log *Logger) Open(filepath, filename, level string) error
```
初始化日志文件的存储路径和文件名，并且初始化日志的级别。
比如: err := logger.Open("/home/owner/tmp/logs","log_filename","Debug")
日志的存储路径为/home/owner/tmp/logs；
日志的文件名为：log_filename，常规日志会存在log_filename.log文件中，错误日志会存在log_filename.log.wf文件中；
日志的存储级别为Debug


### func (log \*Logger) Flush
``` go
//将日志flush至硬盘
func (log *Logger) Flush()
```
Flush日志库。注意：在调用Close()关闭日志库时，会自动flush日志写入到硬盘中，不需要显示的调用一次Flush()

### func (log *Logger) Close
``` go
//结束日志写入
func (log *Logger) Close()
```
关闭日志库。注意：如果没有调用Close()关闭日志库的话，将会造成文件句柄泄露

### func (log \*Logger) SetLevel
``` go
//设置日志的级别
func (log *Logger) SetLevel(level string) *Logger
```


### func (log \*Logger) GetLevel()
``` go
func (log *Logger) GetLevel() string
```


### func (log \*Logger) SetFuncCallDepth
``` go
//设置函数栈的深度
func (log *Logger) SetFuncCallDepth(depth int) *Logger
```


### func (log \*Logger) EnableFuncCallDepth
``` go
//设置打印函数栈信息
func (log *Logger) EnableFuncCallDepth(flag bool) *Logger
```


### func (log \*Logger) SetMaxDays
``` go
//设置日志最长保留天数
func (log *Logger) SetMaxDays(day int64) *Logger
```


### func (log \*Logger) GetMaxDays
``` go
func (log *Logger) GetMaxDays() int64
```


### func (log \*Logger) SetMaxLines *Logger
``` go
//设置日志文件最大行数
func (log *Logger) SetMaxLines(line int)
```


### func (log \*Logger) GetMaxLines
``` go
func (log *Logger) GetMaxLines() int
```


### func (log \*Logger) EnableRotate
``` go
//设置日志Rotate
func (log *Logger) EnableRotate(flag bool) *Logger
```

### func (log \*Logger) EnableDaily
``` go
//设置日志按天打印
func (log *Logger) EnableDaily(flag bool) *Logger
```

### func (log \*Logger) Debug
``` go
//打印Debug日志
func (log *Logger) Debug(format string, v ...interface{})
```

### func (log \*Logger) Info
``` go
//打印Info日志
func (log *Logger) Info(format string, v ...interface{})
```


### func (log \*Logger) Notice
``` go
//打印Notice日志
func (log *Logger) Notice(format string, v ...interface{})
```


### func (log \*Logger) Warn
``` go
//打印Warn日志
func (log *Logger) Warn(format string, v ...interface{})
```


### func (log \*Logger) Fatal
``` go
//打印Fatal日志
func (log *Logger) Fatal(format string, v ...interface{})
```

- - -
