package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	cmd       *exec.Cmd
	state     sync.Mutex
	eventTime = make(map[string]time.Time)
)

func NewWatcher(paths []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Event:
				isbuild := true

				// Skip TMP files for Sublime Text.
				if checkTMPFile(e.Name) {
					continue
				}
				if !checkIsGoFile(e.Name) {
					continue
				}

				if t, ok := eventTime[e.Name]; ok {
					// if 500ms change many times, then ignore it.
					// for liteide often gofmt code after save.
					if t.Add(time.Millisecond * 500).After(time.Now()) {
						colorLog("[SKIP] %s\n", e.String())
						isbuild = false
					}
				}
				eventTime[e.Name] = time.Now()

				if isbuild {
					colorLog("[EVEN] %s\n", e)
					go Autobuild()
				}
			case err := <-watcher.Error:
				log.Fatal("error:", err)
			}
		}
	}()

	colorLog("[INFO] Initializing watcher...\n")
	for _, path := range paths {
		fmt.Println(path)
		err = watcher.Watch(path)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func Autobuild() {
	state.Lock()
	defer state.Unlock()

	colorLog("[INFO] Start building...\n")
	path, _ := os.Getwd()
	os.Chdir(path)

	var err error
	// For applications use full import path like "github.com/.../.."
	// are able to use "go install" to reduce build time.
	if conf.GoInstall {
		icmd := exec.Command("go", "install")
		icmd.Stdout = os.Stdout
		icmd.Stderr = os.Stderr
		err = icmd.Run()
	}

	if err == nil {
		bcmd := exec.Command("go", "build")
		bcmd.Stdout = os.Stdout
		bcmd.Stderr = os.Stderr
		err = bcmd.Run()
	}

	if err != nil {
		colorLog("[ERRO] ============== Build failed ===================\n")
		return
	}
	colorLog("[SUCC] Build was successful\n")
	Restart(appname)
}

func Kill() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Kill -> ", e)
		}
	}()
	if cmd != nil {
		cmd.Process.Kill()
	}
}

func Restart(appname string) {
	Debugf("kill running process")
	Kill()
	go Start(appname)
}

func Start(appname string) {
	colorLog("[INFO] Restarting %s ...\n", appname)
	if strings.Index(appname, "./") == -1 {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go cmd.Run()
}

// checkTMPFile returns true if the event was for TMP files.
func checkTMPFile(name string) bool {
	if strings.HasSuffix(strings.ToLower(name), ".tmp") {
		return true
	}
	return false
}

// checkIsGoFile returns true if the name HasSuffix ".go".
func checkIsGoFile(name string) bool {
	if strings.HasSuffix(name, ".go") {
		return true
	}
	return false
}
