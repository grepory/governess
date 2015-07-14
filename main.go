package main

import (
	"fmt"
	"net/url"
	"os"

	"code.google.com/p/go-uuid/uuid"

	"github.com/codegangsta/cli"
	"github.com/opsee/governess/messaging"
)

// This whole function is serious jank, but expedience.
func govern(c *cli.Context) {
	channel := fmt.Sprintf("governess-%s", uuid.NewUUID().String())
	consumer, err := messaging.NewConsumer(c.String("topic"), channel)
	if err != nil {
		panic(err)
	}

	nsqlookupd, err := url.Parse(c.String("nsqd"))
	if err != nil {
		panic(err)
	}
	consumer.ConnectToNSQLookupd(nsqlookupd)

	docker, err := url.Parse(c.String("docker"))
	if err != nil {
		panic(err)
	}

	g := NewGoverness(c.String("tag"), consumer, docker)
	g.Govern()
}

func main() {
	app := cli.NewApp()
	app.Name = "governess"
	app.Usage = "For the children!"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "topic, p",
			Value: "deploy",
			Usage: "NSQ topic to listen to for deploys",
		},
		cli.StringFlag{
			Name:  "docker, d",
			Value: "unix:///var/run/docker.sock",
			Usage: "URI for Docker Remote API (mutually exclusive w/ --docker)",
		},
		cli.StringFlag{
			Name:  "nsqd, n",
			Value: "nsqlookupd:4161",
			Usage: "URI for NSQLookupd",
		},
		cli.StringFlag{
			Name:  "tag, t",
			Value: "latest",
			Usage: "Tag that governess watches for in repository push events.",
		},
	}

	app.Action = govern

	app.Run(os.Args)
}
