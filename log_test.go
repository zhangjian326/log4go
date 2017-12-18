package log4go_test

import (
	"bufio"
	"os"
	"strconv"
	"testing"

	"github.com/xkeyideal/log4go"
)

func TestLogger(t *testing.T) {
	logger := log4go.NewLogger(100, "test")
	err := logger.Open("./logs", "test", "Info")
	if err != nil {
		t.Fatal(err)
	}

	logger.Debug("Debug")
	logger.Info("Info")
	logger.Notice("Notice")
	logger.Warn("Warn")
	logger.Fatal("Fatal")

	f, err := os.Open("./logs/test.log")
	b := bufio.NewReader(f)
	lineNum := 0
	for {
		line, _, err := b.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 {
			lineNum++
		}
	}
	var expected = 2
	if lineNum != expected {
		t.Fatal(lineNum, "not "+strconv.Itoa(expected)+" lines")
	}
}

func TestFileRotate(t *testing.T) {
	logger := log4go.NewLogger(100, "test_ro")
	err := logger.Open("./logs", "test_ro", "Debug")
	if err != nil {
		t.Fatal(err)
	}
	logger.SetMaxLines(1)

	logger.Debug("debug")
	logger.Info("info")
	logger.Notice("notice")
	logger.Warn("warning")
	logger.Fatal("fatal")
}

func BenchmarkFile(b *testing.B) {
	logger := log4go.NewLogger(100, "test2")
	err := logger.Open("./logs", "test2", "Info")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		logger.Info("info")
	}
}

func BenchmarkFileAsynchronous(b *testing.B) {
	logger := log4go.NewLogger(1000, "test2")
	err := logger.Open("./logs", "test2", "Info")
	if err != nil {
		b.Fatal(err)
	}
	logger.EnableAsync()
	for i := 0; i < b.N; i++ {
		logger.Info("info")
	}
}
