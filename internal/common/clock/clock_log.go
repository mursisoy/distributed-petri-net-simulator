package clock

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LogPriority controls the minimum priority of logging events which
// will be logged.
type LogPriority int

// LogPriority enum provides all the valid Priority Levels that can be
// used to log events with.
const (
	DEBUG LogPriority = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// prefixLookup translates priority enums into strings
var prefixLookup = [...]string{
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARNING",
	ERROR:   "ERROR",
	FATAL:   "FATAL",
}

// LogConfig controls the logging parameters of Log and is taken as
// input to Log initialization. See defaults in GetDefaultConfig.
type ClockLogConfig struct {

	// EncodingStrategy for customizable interoperability
	EncodingStrategy func(interface{}) ([]byte, error)
	// DecodingStrategy for customizable interoperability
	DecodingStrategy func([]byte, interface{}) error
	// Priority determines the minimum priority event to log
	Priority LogPriority

	LogFilename string
	FileOutput  bool
}

type ClockLogger struct {
	pid string

	// Priority level at which all events are logged
	priority LogPriority

	// Logfile name
	logfile *os.File

	// Internal logger for printing errors
	logger *log.Logger

	fileOutput bool

	clock *Clock
}

// InitGoVector returns a Log which generates a logs prefixed with
// processid, to a file name logfilename.log. Any old log with the same
// name will be trucated. Config controls logging options. See LogConfig for more details.
func NewClockLog(pid string, config ClockLogConfig) *ClockLogger {

	clockLog := &ClockLogger{}
	clockLog.pid = pid
	clockLog.priority = config.Priority
	clockLog.fileOutput = config.FileOutput
	clockLog.clock = NewClock(pid)

	//Starting File IO . If Log exists, Log Will be deleted and A New one will be created
	var mw io.Writer

	if config.LogFilename != "" && clockLog.fileOutput {
		logname := config.LogFilename
		logFile, err := os.Create(logname)
		if err != nil {
			log.Fatal(err)
		}
		clockLog.logfile = logFile
		mw = io.MultiWriter(os.Stdout, logFile)
	} else {
		mw = io.MultiWriter(os.Stdout)
	}
	clockLog.logger = log.New(mw, "", log.Lshortfile|log.Lmicroseconds)

	return clockLog
}

func (cl *ClockLogger) GetPid() string {
	return cl.pid
}

func (cl *ClockLogger) Close() {
	if cl.fileOutput {
		if err := cl.logfile.Close(); err != nil {
			log.Printf("Failed to close log file: %v", err)
		}
	}
}

// Logs a DEBUG message along with a processID and a vector clock
func (cl *ClockLogger) LogMergeDebugf(clock ClockMap, format string, v ...any) ClockMap {
	return cl.LogMergef(LogPriority(DEBUG), clock, format, v...)
}

// Logs an INFO message along with a processID and a vector clock
func (cl *ClockLogger) LogMergeInfof(clock ClockMap, format string, v ...any) ClockMap {
	return cl.LogMergef(LogPriority(INFO), clock, format, v...)
}

// Logs an ERROR message along with a processID and a vector clock
func (cl *ClockLogger) LogMergeErrorf(clock ClockMap, format string, v ...any) ClockMap {
	return cl.LogMergef(LogPriority(ERROR), clock, format, v...)
}

// Logs a DEBUG message along with a processID and a vector clock
func (cl *ClockLogger) LogDebugf(format string, v ...any) ClockMap {
	return cl.Logf(LogPriority(DEBUG), format, v...)
}

// Logs an INFO message along with a processID and a vector clock
func (cl *ClockLogger) LogInfof(format string, v ...any) ClockMap {
	return cl.Logf(LogPriority(INFO), format, v...)
}

// Logs an ERROR message along with a processID and a vector clock
func (cl *ClockLogger) LogErrorf(format string, v ...any) ClockMap {
	return cl.Logf(LogPriority(ERROR), format, v...)
}

// Ticks and merge de clock
func (cl *ClockLogger) LogMergef(level LogPriority, clock ClockMap, format string, v ...any) ClockMap {
	var clockMap ClockMap
	if level >= cl.priority {
		clockMap = cl.clock.TickAndMerge(clock)
		cl.logger.Output(3, fmt.Sprintf("[%s] - %s\n%s %s", prefixLookup[cl.priority], fmt.Sprintf(format, v...), cl.pid, clockMap))
	} else {
		clockMap = cl.clock.GetClock()
	}
	return clockMap
}

func (cl *ClockLogger) Logf(level LogPriority, format string, v ...any) ClockMap {
	var clockMap ClockMap
	if level >= cl.priority {
		clockMap = cl.clock.Tick()
		cl.logger.Output(3, fmt.Sprintf("[%s] - %s\n%s %s", prefixLookup[cl.priority], fmt.Sprintf(format, v...), cl.pid, clockMap))
	} else {
		clockMap = cl.clock.GetClock()
	}
	return clockMap
}
