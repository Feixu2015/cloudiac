package handlers

import (
	"bufio"
	"cloudiac/consts"
	"cloudiac/libs/ctx"
	"cloudiac/utils/logs"
	"context"
	"fmt"
	"github.com/gin-contrib/sse"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var old_size = map[string]int{}
var end_str = map[string]string{}

func getEvent(filename string) (*sse.Event, error) {
	if end_str[filename] == "end\n" {
		return nil, io.EOF
	}

	file, err := os.Open(fmt.Sprintf("/tmp/%s", filename))
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	fileinfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileSize := fileinfo.Size()
	buffer := make([]byte, fileSize)

	new_size, err := file.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("read %d %d bytes: %q\n", old_size[filename], new_size, buffer[:new_size])

	time.Sleep(time.Second)
	event := sse.Event{
		Event: "hello",
		Data:  fmt.Sprintf("%s", buffer[old_size[filename]:new_size]),
	}
	old_size[filename] = new_size

	end := new_size - 4
	if end > 0 {
		end_str[filename] = fmt.Sprintf("%s", buffer[end:new_size])
	}

	return &event, nil
}

func TestSSE(c *ctx.GinRequestCtx) {
	contx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chanStream := make(chan string)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-c.Request.Context().Done():
				// client gave up
				done <- true
				return
			case <-contx.Done():
				switch contx.Err() {
				case context.DeadlineExceeded:
					log.Printf("timeout")
				}
				done <- true
				return
			}
		}
	}()

	f, err := os.Open("./ct-c1el3dabtmijbv0jg70g/task-c1eqcmqbtmile018n5ng/runner.log")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)

	var mu sync.RWMutex
	go func() {
		for {
			mu.Lock()
			str, _, err := rd.ReadLine()
			if err != nil {
				if err.Error() == "EOF" {
					time.Sleep(1000)
					mu.Unlock()
					continue
				} else {
					log.Println("Read Error:", err.Error())
					done <- true
					return
				}
			}
			chanStream <- string(str)
			mu.Unlock()
		}
	}()

	count := 0 // to indicate the message id
	isStreaming := c.Stream(func(w io.Writer) bool {
		for {
			select {
			case <-done:
				// when deadline is reached, send 'end' event
				c.SSEvent("end", "end")
				return false
			case msg := <-chanStream:
				// send events to client
				c.Render(-1, sse.Event{
					Id:    strconv.Itoa(count),
					Event: "message",
					Data:  msg,
				})
				count++
				return true
			}
		}
	})
	if !isStreaming {
		log.Println("Stream Closed!")
	}
}

func HelloSse(c *ctx.GinRequestCtx) {
	old_size[c.Param("filename")] = 0
	for {
		event, err := getEvent(c.Param("filename"))

		if err == io.EOF {
			c.Status(http.StatusNoContent)
			return
		}
		if event.Data == "" {
			continue
		}

		select {

		case <-c.Request.Context().Done():
			return

		default:
			_ = event.Render(c.Writer)
			c.Writer.Flush()
		}
	}
}

func TaskLogSSE(c *ctx.GinRequestCtx) {
	loggers := logs.Get()
	contx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chanStream := make(chan string, 0)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-c.Request.Context().Done():
				// client gave up
				done <- true
				return
			case <-contx.Done():
				switch contx.Err() {
				case context.DeadlineExceeded:
					loggers.Error("timeout")
				}
				done <- true
				return
			}
		}
	}()

	logPath := c.Query("logPath")
	path := fmt.Sprintf("%s/%s", logPath, consts.TaskLogName)
	f, err := os.Open(path)
	if err != nil {
		loggers.Error(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	fi, err := f.Stat()
	if err != nil {
		loggers.Error(err)
		return
	}

	go func() {
		for {
			str, _, err := rd.ReadLine()
			if err != nil {
				if err.Error() == "EOF" {
					if time.Now().Unix()-fi.ModTime().Unix() > 60 {
						done <- true
						return
					}
					time.Sleep(1000)
					continue
				} else {
					loggers.Error("Read Error:", err.Error())
					done <- true
					return
				}
			}
			chanStream <- string(str)
		}
	}()

	count := 0 // to indicate the message id
	isStreaming := c.Stream(func(w io.Writer) bool {
		for {
			select {
			case <-done:
				// when deadline is reached, send 'end' event
				c.SSEvent("end", "end")
				return false
			case msg := <-chanStream:
				// send events to client
				c.Render(-1, sse.Event{
					Id:    strconv.Itoa(count),
					Event: "message",
					Data:  msg,
				})
				count++
				return true
			}
		}
	})
	if !isStreaming {
		loggers.Info("Stream Closed!")
	}
}