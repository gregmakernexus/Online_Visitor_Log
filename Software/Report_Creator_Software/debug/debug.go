// package sheet provides a simplified api to google sheets.
package debug

// debug is intended to function like log, but introduces
// a logging level to supress too many debug messages.  Also allows
// for debug messages to be turn off.
import (
	"fmt"
	"os"
	"path"
	"runtime"
	"time"
)

type DebugClient struct {
	debugLevel int
	msgLevel   int
	caller     string
	time       string
}

func NewLogClient(v int) *DebugClient {
	l := new(DebugClient)
	l.debugLevel = v
	return l
}

func (d *DebugClient) V(v int) *DebugClient {
	d.caller = getCallerInfo()
	t := time.Now()
	d.time = t.Format("2006-01-02 15:04:05")
	d.msgLevel = v
	return d
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (d *DebugClient) Printf(format string, v ...any) {
	if d.msgLevel <= d.debugLevel {
		fmt.Printf("%v %v: ", d.time, d.caller)
		fmt.Printf(format, v...)
	}
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Println.
func (d *DebugClient) Println(v ...any) {
	if d.msgLevel <= d.debugLevel {
		fmt.Printf("%v %v: ", d.time, d.caller)
		fmt.Println(v...)
	}
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func (d *DebugClient) Fatal(v ...any) {
	fmt.Printf("%v %v: ", d.time, d.caller)
	fmt.Printf("%v", v...)
	os.Exit(0)

}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func (d *DebugClient) Fatalf(format string, v ...any) {
	fmt.Printf("%v %v: ", d.time, d.caller)
	fmt.Printf(format, v...)
	os.Exit(0)
}
func getCallerInfo() string {

	_, file, lineNo, ok := runtime.Caller(2)
	if !ok {
		return "runtime.Caller() failed"
	}
	// funcName := runtime.FuncForPC(pc).Name()
	fileName := path.Base(file) // The Base function returns the last element of the path
	return fmt.Sprintf("%v:%v ", fileName, lineNo)
}
