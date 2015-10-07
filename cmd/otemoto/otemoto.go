package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ikawaha/otemoto"
)

func init() {
	tz, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	time.Local = tz
}

const (
	sleep    = 5 * time.Second
	interval = 1 * time.Minute
)

var (
	schedule = otemoto.TimeTable{
		{Hour: 12, Minute: 0, Message: "お昼ですよ！"},
		{Hour: 15, Minute: 0, Message: ":coffee: おやつの時間〜"},
		{Hour: 18, Minute: 30, Message: ":octocat: もう帰ろうよー"},
	}
)

func Run(token string, schedule otemoto.TimeTable, notify <-chan struct{}, done chan<- error) {
	bot, err := otemoto.New(token, schedule)
	if err != nil {
		time.Sleep(1 * time.Minute)
		done <- err
		return
	}
	defer bot.Close()
	fmt.Println("^C exits\n")

	msgch := make(chan otemoto.Message, 1)
	errch := make(chan error, 1)
	quit := make(chan struct{}, 1)
	go func(q chan struct{}) {
		for {
			select {
			case <-q:
				return
			default:
				msg, err := bot.GetMessage()
				if err != nil {
					errch <- err
				}
				msgch <- msg
			}
		}
	}(quit)
	heartbeat := bot.Heartbeat(30*time.Second, 60*time.Second)
	scheduleTimer := time.Tick(interval)
	for {
		select {
		case msg := <-msgch:
			log.Printf("bot_id: %v, msguser_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
			if msg.Type != "message" || msg.SubType != "" {
				continue
			}
			if bot.ID == msg.MentionID() {
				go bot.Tokenize(msg)
			}
			go bot.Dajarep(msg, sleep)
			go bot.Haiku(msg, sleep)
		case err := <-errch:
			log.Printf("receive error, %v", err)
		case <-scheduleTimer:
			go bot.CheckSchedule()
		case err := <-heartbeat:
			log.Printf("receive heartbeat, %v", err)
			quit <- struct{}{}
			done <- err
			return
		case <-notify:
			log.Println("slackbot, receive exit message...")
			quit <- struct{}{}
			done <- nil
			return
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: bot slack-bot-token\n")
		os.Exit(1)
	}

	sys := make(chan os.Signal, 1)
	signal.Notify(sys, syscall.SIGINT)
	quit := make(chan struct{}, 1)
	done := make(chan error, 1)

	token := os.Args[1]
	go Run(token, schedule, quit, done) // slack event loop
loop:
	for {
		select {
		case <-sys:
			log.Println("received ^C ...")
			quit <- struct{}{}
		case err := <-done:
			if err == nil {
				break loop
			}
			go Run(token, schedule, quit, done)
		}
	}
	log.Println("done")
}
