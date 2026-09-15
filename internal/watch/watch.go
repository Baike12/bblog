package watch

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher 监听内容目录变化，去抖后回调一次。
type Watcher struct {
	dir      string
	debounce time.Duration
	onChange func()
	watcher  *fsnotify.Watcher
	done     chan struct{}
}

// New 创建监听器，onChange 会在去抖窗口结束后被调用。
func New(dir string, debounce time.Duration, onChange func()) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{dir: dir, debounce: debounce, onChange: onChange, watcher: w, done: make(chan struct{})}, nil
}

// Start 注册目录并在后台开始处理事件。
func (w *Watcher) Start() error {
	if err := w.addRecursive(w.dir); err != nil {
		return err
	}
	go w.loop()
	return nil
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if err := w.watcher.Add(path); err != nil {
			log.Printf("warn: 监听目录失败 %s: %v", path, err)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	var timer *time.Timer
	var timerC <-chan time.Time

	fire := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(w.debounce)
		timerC = timer.C
	}

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addRecursive(event.Name)
				}
			}
			fire()
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("warn: 文件监听错误: %v", err)
		case <-timerC:
			timerC = nil
			w.onChange()
		}
	}
}

// Close 停止监听。
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}
