package main

import (
	v1 "cloudiac/runner/api/v1"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloudiac/cmds/common"
	"cloudiac/configs"
	"cloudiac/runner"
	"cloudiac/utils/logs"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
)

type Option struct {
	common.OptionVersion

	Config     string `short:"c" long:"config"  default:"config.yml" description:"config file"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug message"`
	ReRegister bool   `long:"re-register" description:"Re registration service to Consul"`
}

func main() {
	opt := Option{}
	_, err := flags.Parse(&opt)
	if err != nil {
		os.Exit(1)
	}

	common.ShowVersionIf(opt.Version)
	configs.Init(opt.Config)
	if err := checkConfigs(configs.Get()); err != nil {
		panic(err)
	}
	if err := ensureDirs(); err != nil {
		panic(err)
	}

	conf := configs.Get().Log
	logs.Init(conf.LogLevel, conf.LogMaxDays, "ct-runner")

	common.ReRegisterService(opt.ReRegister, "CT-Runner")
	StartServer()
}

func checkConfigs(c *configs.Config) error {
	cases := []struct {
		name  string
		value string
	}{
		{"runner.default_image", c.Runner.DefaultImage},
		{"runner.storage_path", c.Runner.StoragePath},
		{"runner.asset_path", c.Runner.AssetPath},
		{"runner.plugin_cache_path", c.Runner.PluginCachePath},
	}

	for _, c := range cases {
		if c.value == "" {
			return fmt.Errorf("configuration '%s' is empty", c.name)
		}
	}
	return nil
}

// ensureDirs 确保依赖的目录存在
func ensureDirs() error {
	c := configs.Get().Runner

	var err error
	for _, path := range []string{c.StoragePath, c.AssetPath, c.PluginCachePath, c.AssetProviderPath()} {
		// 确保可以转为绝对路径，因为挂载到容器中时必须使用绝对路径
		path, err = filepath.Abs(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Abs(%s)", path))
		} else if err = os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func StartServer() {
	conf := configs.Get()
	logger := logs.Get()

	name := "ct-runner"
	abs, _ := filepath.Abs(os.Args[0])
	dir := filepath.Dir(abs)
	ext := filepath.Ext(name)
	execName := name[:len(name)-len(ext)]

	logPath := filepath.Join(dir, "logs", execName+".log")
	f, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0666)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	type request struct {
		*http.Request
		doneCh chan struct{}
	}

	requestChan := make(chan request, 32)
	e := gin.Default()

	apiV1 := e.Group("/api/v1")
	apiV1.Any("/check", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
		})
	})
	apiV1.POST("/metrics", func(c *gin.Context) {
		r := request{Request: c.Request, doneCh: make(chan struct{}, 0)}
		requestChan <- r
		<-r.doneCh
	})

	//fp, err := os.OpenFile("./metrics.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	//if err != nil {
	//	panic(err)
	//}

	apiV1.GET("/metrics", func(c *gin.Context) {
		timer := time.NewTimer(time.Millisecond * 100)
		totalRead := int64(0)

		defer func() {
			logger.Debugf("total read %d bytes", totalRead)
			c.Request.Body.Close()
			timer.Stop()
		}()

		for {
			select {
			case <-timer.C:
				return
			case req := <-requestChan:
				//w := io.MultiWriter(c.Writer, fp)
				w := c.Writer
				nr, err := io.Copy(w, req.Body)
				//logger.Infof("copy %d bytes", nr)
				totalRead += nr
				if err != nil {
					logger.Errorf("io copy error: %v", err)
				}
				close(req.doneCh)
			}
		}
	})

	apiV1.POST("/task/run", func(c *gin.Context) {
		id, err := runner.Run(c.Request)
		if err != nil {
			logger.Error(err.Error())
			c.JSON(500, gin.H{
				"err": err.Error(),
			})
		} else {
			c.JSON(200, gin.H{
				"id": id,
			})
		}
	})

	apiV1.GET("/task/status", v1.TaskStatus)
	apiV1.GET("/task/log/follow", v1.TaskLogFollow)

	apiV1.POST("/task/cancel", func(c *gin.Context) {
		logger.Debug(c.Request.Body)
		err := runner.Cancel(c.Request)
		if err != nil {
			c.JSON(500, gin.H{
				"error": err,
			})
		} else {
			c.JSON(200, gin.H{
				"error": nil,
			})
		}
	})

	logger.Infof("starting runner on %v", conf.Listen)
	if err := e.Run(conf.Listen); err != nil {
		logger.Fatalln(err)
	}
}
