package gozaru

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/ikawaha/slackbot"
	"github.com/kurehajime/dajarep"
	"github.com/mattn/go-haiku"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	_ = tokenizer.SysDic()
}

var (
	r575 = []int{5, 7, 5}
)

type TimeTable []struct {
	Hour, Minute int
	Message      string
	notified     bool
}

type Bot struct {
	*slackbot.Bot
	schedule TimeTable
}

type Message struct {
	slackbot.Message
}

func New(token string, tt TimeTable) (*Bot, error) {
	b, err := slackbot.New(token)
	if err != nil {
		return nil, err
	}
	t := make(TimeTable, len(tt), len(tt))
	copy(t, tt)
	return &Bot{
		Bot:      b,
		schedule: t,
	}, nil
}

func (b Bot) GetMessage() (Message, error) {
	m, err := b.Bot.GetMessage()
	return Message{m}, err
}

func (b Bot) PostMessage(m Message) error {
	return b.Bot.PostMessage(m.Message)
}

func (b Bot) Tokenize(m Message) {
	sen := m.TextBody()
	t := tokenizer.New()
	tokens := t.Tokenize(sen)
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "```")
	for i := 1; i < len(tokens); i++ {
		if tokens[i].Class == tokenizer.DUMMY {
			fmt.Fprintf(&buf, "%s\n", tokens[i].Surface)
			continue
		}
		features := strings.Join(tokens[i].Features(), ",")
		fmt.Fprintf(&buf, "%s\t%v\n", tokens[i].Surface, features)
	}
	fmt.Fprintln(&buf, "```")
	m.Text = buf.String()
	if e := b.PostMessage(m); e != nil {
		log.Printf("tokenize, post error, %v", e)
	}
}

func (b Bot) Dajarep(m Message, sleep time.Duration) {
	t := m.TextBody()
	dug, daj := dajarep.Dajarep(t)
	log.Printf("msg: %v, dajare: %+v, debug: %+v\n", t, daj, dug)
	if len(daj) < 1 {
		return
	}
	const tpl = "ねぇねぇ，%v\nいまの ```%v``` ってダジャレ？ダジャレ？"
	m.Text = fmt.Sprintf(tpl, b.UserName(m.UserID), t)
	time.Sleep(time.Duration(rand.Int63n(int64(sleep))))
	b.PostMessage(m)
}

func (b Bot) Haiku(m Message, sleep time.Duration) {
	t := m.TextBody()
	hs := haiku.Find(t, r575)
	log.Printf("msg: %v, haiku: %+v\n", t, hs)
	if len(hs) < 1 {
		return
	}
	var tmp []string
	for _, h := range hs {
		tmp = append(tmp, fmt.Sprintf("```%v```", h))
	}
	m.Text = strings.Join(tmp, "\n")
	m.Text += " 575だ"
	time.Sleep(time.Duration(rand.Int63n(int64(sleep))))
	b.PostMessage(m)
}

func (b *Bot) CheckSchedule() {
	h, m, _ := time.Now().Clock()
	for i := 0; i < len(b.schedule); i++ {
		s := &b.schedule[i]
		if !s.notified && h == s.Hour && m == s.Minute {
			s.notified = true
			for c := range b.Channels {
				go b.PostMessage(Message{
					slackbot.Message{
						Type:    "message",
						UserID:  b.ID,
						Channel: c,
						Text:    s.Message,
					},
				})
			}
		}
		if s.notified && m != s.Minute {
			s.notified = false
		}
	}
}
