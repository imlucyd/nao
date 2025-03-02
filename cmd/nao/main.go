package main

import (
	"context"
	"io"
	"os"
	"os/user"
	"runtime"

	"github.com/luisnquin/nao/v3/internal"
	"github.com/luisnquin/nao/v3/internal/cmd"
	"github.com/luisnquin/nao/v3/internal/config"
	"github.com/luisnquin/nao/v3/internal/data"
	"github.com/luisnquin/nao/v3/internal/ui"
	"github.com/rs/zerolog"
)

func main() {
	defer func() {
		if v := recover(); v != nil {
			ui.Fatalf("%v", v)
			os.Exit(1)
		}
	}()

	logFile, err := os.OpenFile("/tmp/nao.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, internal.PermReadWrite)
	if err != nil {
		panic(err) // Read-only temp directory aaS
	}

	logFile.WriteString("\n\n")

	var logger zerolog.Logger

	if internal.Debug {
		logger = zerolog.New(io.MultiWriter(logFile, os.Stderr))
	} else {
		logger = zerolog.New(logFile)
	}

	caller, err := user.Current()
	if err != nil {
		logger.Err(err).Msg("unable to get current user, incoming panic")

		panic(err) // Otherwise the program cannot be used and nothing will be broken
	}

	logger.Trace().
		Str("app", internal.AppName).Str("version", internal.Version).Str("kind", internal.Kind).
		Str("runtime", runtime.Version()).Str("os", runtime.GOOS).Str("arch", runtime.GOARCH).Send()

	logger.Debug().Str("username", caller.Username).Str("uid", caller.Uid).
		Str("gid", caller.Gid).Str("home", caller.HomeDir).Strs("input", os.Args).Send()

	logger.Trace().Msg("loading configuration...")

	config, err := config.New(&logger)
	if err != nil {
		logger.Err(err).Msg("an error was encountered while loading configuration...")

		ui.Error(err.Error())
		os.Exit(1)
	}

	logger.Trace().Msg("loading data...")

	data, err := data.NewBuffer(&logger, config)
	if err != nil {
		logger.Err(err).Msg("an error was encountered while loading data...")

		ui.Error(err.Error())
		os.Exit(1)
	}

	logger.Trace().Msg("executing command...")

	ctx := context.Background()

	if err := cmd.Execute(ctx, &logger, config, data); err != nil {
		logger.Err(err).Msg("an error was encountered while executing command...")

		ui.Error(err.Error())
		os.Exit(1)
	}

	logger.Trace().Msg("finished without critical errors")
}
