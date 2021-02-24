package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	nats "github.com/nats-io/nats.go"
	"gopkg.in/alecthomas/kingpin.v2"
)

// URL shows the nats-server
var URL = "nats://s1:s1@10.253.0.41:4222"

// ROLE shows what action should I do
var ROLE = "edge"

// SUBJECT shows where i should talk to or hear from
var SUBJECT = "mysqldb2.1"

// STREAM is nats stream name
var STREAM = "mysqldb2"

// CONSUMER is nats consumer name
var CONSUMER = "mysqldb2"

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	url := os.Getenv("NATS_URL")
	role := os.Getenv("ROLE")
	subj := os.Getenv("SUBJECT")
	stream := os.Getenv("STREAM")
	consumer := os.Getenv("CONSUMER")
	log.Printf("I get some environment variables: | server: %s | role: %s | subject: %s | stream: %s | consumer: %s |",
		url, role, subj, stream, consumer)
	if url != "" {
		URL = url
	}
	if role != "" {
		ROLE = role
	}
	if subj != "" {
		SUBJECT = subj
	}
	if stream != "" {
		STREAM = stream
	}
	if consumer != "" {
		CONSUMER = consumer
	}
}

func main() {
	log.Printf("Run with those environment variables: | URL: %s | ROLE: %s | SUBJECT: %s | STREAM: %s | CONSUMER: %s |",
		URL, ROLE, SUBJECT, STREAM, CONSUMER)
	nc, err := nats.Connect(URL)
	if err != nil {
		log.Fatal("Something was wrong when connecting to nats-server: ", err)
	}
	defer nc.Close()

	if strings.ToLower(ROLE) == "edge" {
		for {
			currentTime := time.Now().Format("2006-01-02 15:04:05")
			data := fmt.Sprintf("Edge: Hello, can you hellp me? %s", currentTime)
			msg, err := nc.Request(SUBJECT, []byte(data), 1*time.Second)
			if err != nil {
				log.Fatal("Request error: ", err)
			}
			sleepTime := 5 + rand.Intn(10)
			log.Printf("MSG: %s . Sleep time: %d ", msg.Data, sleepTime)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
	} else if strings.ToLower(ROLE) == "cloud" {
		timeout := 5 * time.Second
		jsopts := []jsm.Option{
			jsm.WithTimeout(timeout),
		}
		mgr, err := jsm.New(nc, jsopts...)
		kingpin.FatalIfError(err, "create jsm failed")

		for {
			req := &api.JSApiConsumerGetNextRequest{Batch: 1, Expires: time.Now().Add(timeout)}
			sub, err := nc.SubscribeSync(nats.NewInbox())
			kingpin.FatalIfError(err, "subscribe failed")
			sub.AutoUnsubscribe(1)

			err = mgr.NextMsgRequest(STREAM, CONSUMER, sub.Subject, req)
			kingpin.FatalIfError(err, "could not request next message")

			fatalIfNotPull := func() {
				cons, err := mgr.LoadConsumer(STREAM, CONSUMER)
				kingpin.FatalIfError(err, "could not load consumer %q", CONSUMER)

				if !cons.IsPullMode() {
					kingpin.Fatalf("consumer %q is not a Pull consumer", CONSUMER)
				}
			}

			msg, err := sub.NextMsg(timeout)
			if err != nil {
				fatalIfNotPull()
			}
			kingpin.FatalIfError(err, "no message received")

			if msg.Header != nil && msg.Header.Get("Status") == "503" {
				fatalIfNotPull()
			}

			info, err := jsm.ParseJSMsgMetadata(msg)
			if err != nil {
				if msg.Reply == "" {
					fmt.Printf("--- subject: %s\n", msg.Subject)
				} else {
					fmt.Printf("--- subject: %s reply: %s\n", msg.Subject, msg.Reply)
				}

			} else {
				fmt.Printf("[%s] subj: %s / tries: %d / cons seq: %d / str seq: %d / pending: %d\n", time.Now().Format("15:04:05"), msg.Subject, info.Delivered(), info.ConsumerSequence(), info.StreamSequence(), info.Pending())
			}

			if len(msg.Header) > 0 {
				fmt.Println("Headers:")
				for h, vals := range msg.Header {
					for _, val := range vals {
						fmt.Printf("  %s: %s\n", h, val)
					}
				}
			}
			fmt.Println("Data:")
			fmt.Println(string(msg.Data))
			fmt.Println()
			time.Sleep(5 * time.Second)

			// subj := fmt.Sprintf("$JS.API.CONSUMER.MSG.NEXT.%s.%s", STREAM, CONSUMER)
			// msg, err := nc.Request(subj, []byte("1"), 1*time.Second)
			// if err != nil {
			// 	log.Fatal("Request error: ", err)
			// }
			// sleepTime := 5 + rand.Intn(10)
			// log.Printf("MSG: %s . Sleep time: %d ", msg.Data, sleepTime)
			// time.Sleep(time.Duration(sleepTime) * time.Second)
		}
	} else {
		log.Fatal("I don't know what to do when I play this role: ", ROLE)
	}
}
