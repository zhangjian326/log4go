package log4go

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileLog struct {
	sync.Mutex

	maxLines               int
	normalMaxLinesCurLines int
	errMaxLinesCurLines    int

	// Rotate at size
	maxSize              int
	normalMaxSizeCurSize int
	errMaxSizeCurSize    int

	// Rotate daily
	daily         bool
	maxDays       int64
	dailyOpenDate int

	rotate bool

	file     *os.File
	errFile  *os.File
	filepath string
	filename string
}

func newFileLog() *FileLog {
	return &FileLog{
		filename: "log_filename",
		maxLines: FileDefMaxLines,
		maxSize:  FileDefMaxSize,
		daily:    true,
		maxDays:  FileDefMaxDays,
		rotate:   true,
	}
}

func (fileLog *FileLog) openFile(filename string) (*os.File, error) {

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return nil, fmt.Errorf("open %s failed, err:%v", filename, err)
	}

	return file, err
}

func (fileLog *FileLog) startLogger() error {

	normalLog := fileLog.filepath + "/" + fileLog.filename + ".log"
	file, err := fileLog.openFile(normalLog)
	if err != nil {
		return err
	}
	if fileLog.file != nil {
		fileLog.file.Close()
	}
	fileLog.file = file

	warnLog := normalLog + ".wf"
	errFile, err := fileLog.openFile(warnLog)
	if err != nil {
		fileLog.file.Close()
		fileLog.file = nil
		return err
	}
	if fileLog.errFile != nil {
		fileLog.errFile.Close()
	}
	fileLog.errFile = errFile

	return fileLog.initFd()
}

func (fileLog *FileLog) initFd() error {
	fileLog.dailyOpenDate = time.Now().Day()

	normalFd := fileLog.file
	fInfo, err := normalFd.Stat()
	if err != nil {
		return fmt.Errorf("get normalfile stat err:%v", err)
	}
	fileLog.normalMaxSizeCurSize = int(fInfo.Size())
	fileLog.normalMaxLinesCurLines = 0
	if fInfo.Size() > 0 {
		normalLog := fileLog.filepath + "/" + fileLog.filename + ".log"
		count, err := fileLog.lines(normalLog)
		if err != nil {
			return err
		}
		fileLog.normalMaxLinesCurLines = count
	}

	errFd := fileLog.errFile
	fInfo, err = errFd.Stat()
	if err != nil {
		return fmt.Errorf("get errfile stat err:%v", err)
	}
	fileLog.errMaxSizeCurSize = int(fInfo.Size())
	fileLog.errMaxLinesCurLines = 0
	if fInfo.Size() > 0 {
		errLog := fileLog.filepath + "/" + fileLog.filename + ".log.wf"
		count, err := fileLog.lines(errLog)
		if err != nil {
			return err
		}
		fileLog.errMaxLinesCurLines = count
	}
	return nil
}

func (fileLog *FileLog) lines(filepath string) (int, error) {
	fd, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	buf := make([]byte, 32768) // 32k
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count, nil
}

func (fileLog *FileLog) needRotate(size int, day int, normal bool) bool {

	curLines := fileLog.normalMaxLinesCurLines
	curSize := fileLog.normalMaxSizeCurSize
	if normal == false {
		curLines = fileLog.errMaxLinesCurLines
		curSize = fileLog.errMaxSizeCurSize
	}

	return (fileLog.maxLines > 0 && curLines >= fileLog.maxLines) ||
		(fileLog.maxSize > 0 && curSize >= fileLog.maxSize) ||
		(fileLog.daily && day != fileLog.dailyOpenDate)
}

func (fileLog *FileLog) WriteMsg(service, hostname string, when time.Time, msg string, level int) error {
	h, d := formatTimeHeader(when)

	msg = fmt.Sprintf("[%s] [%s] [%s] [%s] %s\n", strings.TrimSpace(string(h)), service, hostname, levelTextArray[level], msg)
	var msgLength int
	msgLength = len(msg)

	var err error
	if fileLog.rotate {
		fileLog.Lock()
		if level >= LevelWarn {
			if fileLog.needRotate(msgLength, d, false) {
				if err = fileLog.doRotate(when, false); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogErrWriter(%q): %s\n", fileLog.filename, err)
				}
			}
		} else {
			if fileLog.needRotate(msgLength, d, true) {
				if err = fileLog.doRotate(when, true); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", fileLog.filename, err)
				}
			}
		}
		fileLog.Unlock()
	}

	fileLog.Lock()
	if level >= LevelWarn {
		_, err = fileLog.errFile.Write([]byte(msg))
		if err == nil {
			fileLog.errMaxLinesCurLines++
			fileLog.errMaxSizeCurSize += msgLength
		}
	} else {
		_, err = fileLog.file.Write([]byte(msg))
		if err == nil {
			fileLog.normalMaxLinesCurLines++
			fileLog.normalMaxSizeCurSize += msgLength
		}
	}
	fileLog.Unlock()
	return err
}

func (fileLog *FileLog) doRotate(logTime time.Time, normal bool) error {
	filepath := ""
	if normal == true {
		filepath = fileLog.filepath + "/" + fileLog.filename + ".log"
	} else {
		filepath = fileLog.filepath + "/" + fileLog.filename + ".log.wf"
	}
	_, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	// file exists
	// Find the next available number
	num := 1
	fName := ""
	if fileLog.maxLines > 0 || fileLog.maxSize > 0 {
		for ; err == nil && num < MaxFileNum; num++ {
			if normal == true {
				fName = fileLog.filepath + "/" + fileLog.filename + fmt.Sprintf(".%s.%03d.log", logTime.Format("2006-01-02"), num)
			} else {
				fName = fileLog.filepath + "/" + fileLog.filename + fmt.Sprintf(".%s.%03d.log.wf", logTime.Format("2006-01-02"), num)
			}
			_, err = os.Lstat(fName)
		}
	} else {
		if normal == true {
			fName = fmt.Sprintf("%s/%s.%s.log", fileLog.filepath, fileLog.filename, logTime.Format("2006-01-02"))
		} else {
			fName = fmt.Sprintf("%s/%s.%s.log.wf", fileLog.filepath, fileLog.filename, logTime.Format("2006-01-02"))
		}
		_, err = os.Lstat(fName)
	}

	// return error if the last file checked still existed
	if err == nil {
		return fmt.Errorf("Rotate: Cannot find free log number to rename %s", fileLog.filename)
	}

	// close fileWriter before rename
	if normal == true {
		fileLog.file.Close()
	} else {
		fileLog.errFile.Close()
	}

	// Rename the file to its new found name
	// even if occurs error,we MUST guarantee to  restart new logger
	renameErr := os.Rename(filepath, fName)
	// re-start logger
	startLoggerErr := fileLog.startLogger()

	go fileLog.deleteOldLog()

	if renameErr != nil {
		return fmt.Errorf("Rotate: %s", renameErr.Error())
	}
	if startLoggerErr != nil {
		return fmt.Errorf("Rotate StartLogger: %s", startLoggerErr.Error())
	}

	return nil
}

func (fileLog *FileLog) deleteOldLog() {
	dir := filepath.Dir(fileLog.filepath)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log '%s', error: %v\n", path, r)
			}
		}()

		if !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-86400*fileLog.maxDays) {
			if strings.HasPrefix(filepath.Base(path), fileLog.filename) &&
				(strings.HasSuffix(filepath.Base(path), ".log") || strings.HasSuffix(filepath.Base(path), ".log.wf")) {
				os.Remove(path)
			}
		}
		return
	})
}

func (fileLog *FileLog) Destroy() {
	if fileLog.errFile != nil {
		fileLog.errFile.Close()
		fileLog.errFile = nil
	}
	if fileLog.file != nil {
		fileLog.file.Close()
		fileLog.file = nil
	}
}

// Flush flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (fileLog *FileLog) Flush() {
	if fileLog.errFile != nil {
		fileLog.errFile.Sync()
	}
	if fileLog.file != nil {
		fileLog.file.Sync()
	}
}

func formatTimeHeader(when time.Time) ([]byte, int) {
	y, mo, d := when.Date()
	h, mi, s := when.Clock()
	//len(2006/01/02 15:03:04)==19
	var buf [20]byte
	t := 3
	for y >= 10 {
		p := y / 10
		buf[t] = byte('0' + y - p*10)
		y = p
		t--
	}
	buf[0] = byte('0' + y)
	buf[4] = '/'
	if mo > 9 {
		buf[5] = '1'
		buf[6] = byte('0' + mo - 9)
	} else {
		buf[5] = '0'
		buf[6] = byte('0' + mo)
	}
	buf[7] = '/'
	t = d / 10
	buf[8] = byte('0' + t)
	buf[9] = byte('0' + d - t*10)
	buf[10] = ' '
	t = h / 10
	buf[11] = byte('0' + t)
	buf[12] = byte('0' + h - t*10)
	buf[13] = ':'
	t = mi / 10
	buf[14] = byte('0' + t)
	buf[15] = byte('0' + mi - t*10)
	buf[16] = ':'
	t = s / 10
	buf[17] = byte('0' + t)
	buf[18] = byte('0' + s - t*10)
	buf[19] = ' '

	return buf[0:], d
}
