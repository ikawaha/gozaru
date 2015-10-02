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
	interval = 5 * time.Second
)

var (
	schedule = otemoto.TimeTable{
		{Hour: 15, Minute: 0, Message: ":coffee: おやつの時間〜"},
		{Hour: 18, Minute: 30, Message: ":octocat: もう帰ろうよー"},
	}
)

func Run(bot *otemoto.Bot, notify <-chan struct{}, done chan<- struct{}) {
	msgch := make(chan otemoto.Message, 1)
	errch := make(chan error, 1)
	go func() {
		for {
			msg, err := bot.GetMessage()
			if err != nil {
				errch <- err
			}
			msgch <- msg
		}
	}()
loop:
	for {
		select {
		case <-notify:
			log.Println("slackbot, receive exit message...")
			break loop
		case err := <-errch:
			log.Printf("receive error, %v", err)
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
		case <-time.Tick(interval):
			go bot.CheckSchedule()
		}
	}
	done <- struct{}{}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: bot slack-bot-token\n")
		os.Exit(1)
	}

	bot, err := otemoto.New(os.Args[1], schedule)
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	fmt.Println("^C exits\n")

	sys := make(chan os.Signal, 1)
	signal.Notify(sys, syscall.SIGINT)
	exit := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	go Run(bot, exit, done) // slack event loop
loop:
	for {
		select {
		case <-sys:
			log.Println("received ^C ...")
			exit <- struct{}{}
			break loop
		}
	}
	<-done
	log.Println("done")
}
