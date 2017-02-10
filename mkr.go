package main

import (
	"os"
	"runtime"

	"github.com/mackerelio/mackerel-agent/config"
	"gopkg.in/urfave/cli.v2"
)

func main() {
	app := &cli.App{}
	app.Name = "mkr"
	app.Version = Version
	app.Usage = "A CLI tool for mackerel.io"
	app.Authors = []*cli.Author{{Name: "Hatena Co., Ltd."}}
	app.Commands = Commands
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "conf",
			Value: config.DefaultConfig.Conffile,
			Usage: "Config file path",
		},
		&cli.StringFlag{
			Name:  "apibase",
			Value: config.DefaultConfig.Apibase,
			Usage: "API Base",
		},
	}

	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu)

	app.Run(os.Args)
}
