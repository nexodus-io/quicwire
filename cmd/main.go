package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

	quicmesh "github.com/packetdrop/quicmesh/internal"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

const (
	qnetLogEnv    = "QMESH_LOGLEVEL"
	tunnelOptions = "Tunnel Options"
	miscOptions   = "Misc Options"
)

func qnetRun(cCtx *cli.Context, logger *zap.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer cancel()
	if cCtx.String("cpuprofile") != "" {
		pprofPath := cCtx.String("cpuprofile")
		f, err := os.Create(pprofPath)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Fatal(err.Error())
		}
		defer pprof.StopCPUProfile()
	}

	if cCtx.String("memprofile") != "" {
		pprofPath := cCtx.String("memprofile")
		f, err := os.Create(pprofPath)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			logger.Fatal(err.Error())
		}
	}

	qmesh, err := quicmesh.NewQuicMesh(
		logger.Sugar(),
		cCtx.String("config-file"),
		cCtx.Bool("disable-client"),
		cCtx.Bool("disable-server"),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	wg := &sync.WaitGroup{}

	if err := qmesh.Start(ctx, wg); err != nil {
		logger.Fatal(err.Error())
	}
	<-ctx.Done()
	qmesh.Stop()
	wg.Wait()

	return nil
}

// https://www.rfc-editor.org/rfc/rfc9221.html
func main() {
	// set the log level
	debug := os.Getenv(qnetLogEnv)
	var logger *zap.Logger
	var err error
	if debug != "" {
		logger, err = zap.NewDevelopment()
		logger.Info("Debug logging enabled")
	} else {
		logCfg := zap.NewProductionConfig()
		logCfg.DisableStacktrace = true
		logger, err = logCfg.Build()
	}
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Overwrite usage to capitalize "Show"
	cli.HelpFlag.(*cli.BoolFlag).Usage = "Show help"
	// flags are stored in the global flags variable
	app := &cli.App{
		Name:  "qmesh",
		Usage: "Agent to configure encrypted mesh networking using QUIC protocol.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config-file",
				Value:    "",
				Usage:    "Quic network configuration file",
				Required: true,
				Category: tunnelOptions,
			},
			&cli.BoolFlag{
				Name:     "disable-client",
				Value:    false,
				Usage:    "Disable client function",
				Required: false,
				Category: tunnelOptions,
			},
			&cli.BoolFlag{
				Name:     "disable-server",
				Value:    false,
				Usage:    "Disable server function",
				Required: false,
				Category: tunnelOptions,
			},
			&cli.StringFlag{
				Name:     "cpuprofile",
				Value:    "",
				Usage:    "Enable cpu profiling and dump pprof data to provided file",
				Required: false,
				Category: miscOptions,
			},
			&cli.StringFlag{
				Name:     "memprofile",
				Value:    "",
				Usage:    "Enable memory profiling and dump pprof data to provided file",
				Required: false,
				Category: miscOptions,
			},
		},
		Action: func(cCtx *cli.Context) error {
			return qnetRun(cCtx, logger)
		},
	}
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err.Error())
	}
}
