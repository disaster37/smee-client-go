package main

import (
	"os"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func run(args []string) error {

	// Logger setting
	log.SetOutput(os.Stdout)

	// CLI settings
	app := cli.NewApp()
	app.Usage = "smee-client that support proxy"
	app.Version = "develop"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Display debug output",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "No print color",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:  "start",
			Usage: "Start smee-client",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "url",
					Usage: "URL of the webhook proxy service. Required. For exemple: https://smee.io/VyOocXe0HCKwlSj)",
				},
				&cli.StringFlag{
					Name:  "target",
					Usage: "Full URL (including protocol and path) of the target service the events will forwarded to. Required. For exemple: http://jenkins.mycompany.local:8080/github-webhook/",
				},
				&cli.StringFlag{
					Name:  "secret",
					Usage: "Secret to be used for HMAC-SHA1 secure hash calculation",
				},
				&cli.DurationFlag{
					Name:  "timeout",
					Usage: "The timeout to wait when access on URL and target. Default to 120s",
					Value: 120 * time.Second,
				},
				&cli.BoolFlag{
					Name:  "self-signed-certificate",
					Usage: "Disable the TLS certificate check only on target",
				},
			},
			Action: startSmee,
		},
	}

	app.Before = func(c *cli.Context) error {

		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		if !c.Bool("no-color") {
			formatter := new(prefixed.TextFormatter)
			formatter.FullTimestamp = true
			formatter.ForceFormatting = true
			log.SetFormatter(formatter)
		}

		return nil
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(args)
	return err
}

func main() {
	err := run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
