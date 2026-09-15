// bblog 是一个以 Markdown 文件为唯一真相的博客服务。
// 子命令：serve 启动站点，check 校验内容目录。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Baike12/bblog/internal/config"
	"github.com/Baike12/bblog/internal/content"
	"github.com/Baike12/bblog/internal/web"
	"github.com/Baike12/bblog/internal/watch"
)

// version 由构建时的 -ldflags "-X main.version=..." 注入。
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "version", "-v", "--version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `用法:
  bblog serve [-config bblog.yaml] [-watch=true] [-debounce 500ms]
  bblog check [-config bblog.yaml] [-content 目录]
  bblog version`)
}

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	configPath := fs.String("config", "bblog.yaml", "配置文件路径")
	enableWatch := fs.Bool("watch", true, "监听内容目录并在变化时重建索引")
	debounce := fs.Duration("debounce", 500*time.Millisecond, "文件变化去抖窗口")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}
	srv, err := web.NewServer(cfg, version)
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}

	if *enableWatch {
		w, watchErr := watch.New(cfg.ContentDir(), *debounce, func() {
			if _, _, reloadErr := srv.Reload(); reloadErr != nil {
				log.Printf("warn: 重建索引失败: %v", reloadErr)
			}
		})
		if watchErr != nil {
			log.Printf("warn: 启动文件监听失败（继续以静态内容运行）: %v", watchErr)
		} else {
			if startErr := w.Start(); startErr != nil {
				log.Printf("warn: 启动文件监听失败（继续以静态内容运行）: %v", startErr)
			} else {
				defer w.Close()
			}
		}
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("bblog %s 启动，监听 %s，基础路径 %q，内容目录 %s",
			version, cfg.Server.Listen, cfg.Site.BasePath, cfg.ContentDir())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("error: 监听失败: %v", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	log.Println("收到退出信号，正在关闭…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("warn: 关闭时出错: %v", err)
		return 1
	}
	return 0
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	configPath := fs.String("config", "bblog.yaml", "配置文件路径")
	contentDir := fs.String("content", "", "内容目录（默认取配置里的 content.dir）")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}
	dir := cfg.ContentDir()
	if *contentDir != "" {
		dir = *contentDir
	}

	posts, pages, fileErrs, err := content.Scan(dir, content.Options{BasePath: cfg.Site.BasePath})
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}
	issues := content.Validate(dir, posts, pages, fileErrs)

	errors, warnings := 0, 0
	for _, issue := range issues {
		fmt.Println(issue.String())
		if issue.Level == content.LevelError {
			errors++
		} else {
			warnings++
		}
	}
	published := 0
	for _, p := range posts {
		if p.Published() {
			published++
		}
	}
	fmt.Printf("检查完成：文章 %d（已发布 %d）、页面 %d、错误 %d、警告 %d\n",
		len(posts), published, len(pages), errors, warnings)
	if errors > 0 {
		return 1
	}
	return 0
}
