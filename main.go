package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/mrobinsn/go-rtorrent/rtorrent"

	log "github.com/Sirupsen/logrus"
)

var (
	name    = "rTorrent XMLRPC CLI"
	version = "0.4.0"
	app     = initApp()
	conn    *rtorrent.RTorrent
)

func initApp() *cli.App {
	app := cli.NewApp()

	app.Name = name
	app.Version = version
	app.Authors = []cli.Author{
		{Name: "Michael Robinson", Email: "mrobinson@outlook.com"},
	}

	// Global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "endpoint",
			Usage: "rTorrent endpoint",
			Value: "http://myrtorrent/RPC2",
		},
		cli.BoolFlag{
			Name:  "disable-cert-check",
			Usage: "disable certificate checking on this endpoint, useful for testing",
		},
		cli.StringFlag{
			Name:  "username",
			Usage: "rTorrent basic auth username",
			Value: "",
		},
		cli.StringFlag{
			Name:  "password",
			Usage: "rTorrent basic auth password",
			Value: "",
		},
	}

	app.Before = setupConnection

	app.Commands = []cli.Command{{
		Name:   "get-ip",
		Usage:  "retrieves the IP for this rTorrent instance",
		Action: getIP,
		Before: setupConnection,
	}, {
		Name:   "get-name",
		Usage:  "retrieves the name for this rTorrent instance",
		Action: getName,
		Before: setupConnection,
	}, {
		Name:   "get-totals",
		Usage:  "retrieves the up/down totals for this rTorrent instance",
		Action: getTotals,
		Before: setupConnection,
	}, {
		Name:   "get-torrents",
		Usage:  "retrieves the torrents from this rTorrent instance",
		Action: getTorrents,
		Before: setupConnection,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "view",
				Usage: "view to use, known values: main, started, stopped, hashing, seeding",
				Value: string(rtorrent.ViewMain),
			},
		},
	}, {
		Name:   "get-files",
		Usage:  "retrieves the files for a specific torrent",
		Action: getFiles,
		Before: setupConnection,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "hash",
				Usage: "hash of the torrent",
				Value: "unknown",
			},
		},
	},
	}

	return app
}

func main() {
	app.Run(os.Args)
}

func setupConnection(c *cli.Context) error {
	rTorrentConn := rtorrent.New(c.GlobalString("endpoint"), c.GlobalBool("disable-cert-check"))
	username := c.GlobalString("username")
	password := c.GlobalString("password")

	if username != "" || password != "" {
		rTorrentConn.SetAuth(username, password)
	}

	conn = rTorrentConn
	return nil
}

func getIP(c *cli.Context) {
	ip, err := conn.IP()
	if err != nil {
		log.WithError(err).Error("error getting rTorrent IP")
	} else {
		fmt.Println(ip)
	}
}

func getName(c *cli.Context) {
	name, err := conn.Name()
	if err != nil {
		log.WithError(err).Error("error getting rTorrent name")
	} else {
		fmt.Println(name)
	}
}

func getTotals(c *cli.Context) {
	// Get Down Total
	downTotal, err := conn.DownTotal()
	if err != nil {
		log.WithError(err).Error("error getting rTorrent down total")
	} else {
		fmt.Printf("%d\n", downTotal)
	}

	// Get Up Total
	upTotal, err := conn.UpTotal()
	if err != nil {
		log.WithError(err).Error("error getting rTorrent up total")
	} else {
		fmt.Printf("%d\n", upTotal)
	}
}

func getTorrents(c *cli.Context) {
	torrents, err := conn.GetTorrents(rtorrent.View(c.String("view")))
	if err != nil {
		log.WithError(err).Error("error getting torrents")
	} else {
		for _, torrent := range torrents {
			fmt.Println(torrent.Pretty())
		}
	}
}

func getFiles(c *cli.Context) {
	files, err := conn.GetFiles(rtorrent.Torrent{Hash: c.String("hash")})
	if err != nil {
		log.WithError(err).Error("error getting files")
	} else {
		log.WithFields(log.Fields{
			"torrent_hash": c.String("hash"),
			"num":          len(files),
		}).Info("found files", len(files))
		for _, file := range files {
			fmt.Println(file.Pretty())
		}
	}
}
