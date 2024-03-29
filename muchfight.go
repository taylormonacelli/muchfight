package muchfight

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	LogFormat string `long:"log-format" choice:"text" choice:"json" default:"text" description:"Log format"`
	Verbose   []bool `short:"v" long:"verbose" description:"Show verbose debug information, each -v bumps log level"`
	logLevel  slog.Level
	Search    []string `short:"s" long:"search" description:"Filter files that contain search string."`
}

func Execute() int {
	if err := parseFlags(); err != nil {
		return 1
	}

	if err := setLogLevel(); err != nil {
		return 1
	}

	if err := setupLogger(); err != nil {
		return 1
	}

	if err := run(); err != nil {
		slog.Error("run failed", "error", err)
		return 1
	}

	return 0
}

func parseFlags() error {
	_, err := flags.Parse(&opts)
	return err
}

func run() error {
	searches := opts.Search

	chain := make([]*exec.Cmd, 0, 1)

	cmd := exec.Command(
		"mdfind",
		`kMDItemFSContentChangeDate >= $time.now(-172800) && kMDItemFSName == '*.go'`,
		"-onlyin", "/Users/mtm/pdev",
	)

	chain = append(chain, cmd)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating pipe: %w", err)
	}

	for _, search := range searches {
		cmd = exec.Command(
			"xargs",
			"-d", "\n",
			"-a", "-",
			"rg", "-l", search,
		)

		cmd.Stdin = pipe

		chain = append(chain, cmd)

		pipe, err = cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("error creating pipe: %w", err)
		}
	}

	cmd = chain[len(chain)-1]

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for _, cmd := range chain {
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("error starting %s: %w", cmd.Path, err)
		}
	}

	for _, cmd := range chain {
		err := cmd.Wait()
		if err != nil {
			return fmt.Errorf("error waiting for %s: %w", cmd.Path, err)
		}
	}

	return nil
}
