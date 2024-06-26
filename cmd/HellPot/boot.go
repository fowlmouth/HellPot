package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/yunginnanet/HellPot/internal/config"
	"github.com/yunginnanet/HellPot/internal/extra"
	"github.com/yunginnanet/HellPot/internal/logger"
)

const (
	defaultConfigWarningDelaySecs = 10
	red                           = "\033[31m"
	reset                         = "\033[0m"
)

func writeConfig(target string) (*config.Parameters, bool) {
	var f *os.File
	var err error
	f, err = os.Create(target) // #nosec G304 -- go home gosec, you're drunk
	if err != nil {
		println("failed to create config file: " + err.Error())
		return nil, false
	}
	if _, err = io.Copy(f, config.Defaults.IO); err != nil {
		println("failed to write default config to file: " + err.Error())
		_ = f.Close()
		return nil, false
	}
	if err = f.Sync(); err != nil {
		panic(err)
	}
	println("wrote default config to " + target)
	var newConf *config.Parameters
	if newConf, err = config.Setup(f); err != nil {
		println("failed to setup config with newly written file: " + err.Error())
		_ = f.Close()
		return nil, false
	}
	_ = f.Close()
	newConf.UsingDefaults = true
	return newConf, true
}

func searchConfig() string {
	var resolvedConf string
	uconf, _ := os.UserConfigDir()
	if uconf == "" && os.Getenv("HOME") != "" {
		uconf = filepath.Join(os.Getenv("HOME"), ".config")
	}

	for _, path := range []string{
		"/etc/HellPot/config.toml",
		"/usr/local/etc/HellPot/config.toml",
		"./config.toml",
		filepath.Join(uconf, "HellPot", "config.toml"),
	} {
		if _, err := os.Stat(path); err == nil {
			resolvedConf = path
			break
		}
	}
	return resolvedConf
}

func readConfig(resolvedConf string) (*config.Parameters, error) {
	var err error
	var setupErr error
	var f *os.File

	if resolvedConf == "" {
		return nil, fmt.Errorf("%w: provided config file is an empty string", io.EOF)
	}

	var runningConfig *config.Parameters

	f, err = os.Open(resolvedConf) // #nosec G304 go home gosec, you're drunk
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()
	if err == nil {
		runningConfig, setupErr = config.Setup(f)
	}
	switch {
	case setupErr != nil:
		println("failed to setup config: " + setupErr.Error())
		err = setupErr
	case err != nil:
		println("failed to open config file for reading: " + err.Error())
		println("trying to create it....")
		newRunningConfig, wroteOK := writeConfig(resolvedConf)
		if wroteOK {
			return newRunningConfig, nil
		}
		println("failed to create config file, cannot continue")
		return nil, fmt.Errorf("failed to create config file: %w", err)
	case runningConfig == nil:
		err = errors.New("unknown failure resulting in missing configuration, cannot continue")
		return nil, err
	}

	return runningConfig, err
}

func resolveConfig() (runningConfig *config.Parameters, usingDefaults bool, resolvedConf string, err error) {
	if config.CLIFlags != nil {
		conf := config.CLIFlags.Lookup("config")
		if conf != nil && conf.Value.String() != "" {
			resolvedConf = conf.Value.String()
			if runningConfig, err = readConfig(resolvedConf); err != nil {
				return runningConfig, false, resolvedConf, err
			}
		}
	}

	if resolvedConf == "" && os.Getenv("HELLPOT_CONFIG_FILE") != "" {
		resolvedConf = os.Getenv("HELLPOT_CONFIG_FILE")
	}

	if resolvedConf == "" {
		resolvedConf = searchConfig()
	}

	if runningConfig, err = readConfig(resolvedConf); err != nil && !errors.Is(err, io.EOF) {
		return runningConfig, false, "", err
	}

	if runningConfig == nil {
		if runningConfig, err = config.Setup(nil); err != nil || runningConfig == nil {
			if err == nil {
				err = errors.New("unknown failure resulting in missing configuration, cannot continue")
			}
			return runningConfig, false, "", err
		}
		return runningConfig, true, "", nil
	}

	return runningConfig, false, resolvedConf, nil
}

func initLogger(runningConfig *config.Parameters) (log zerolog.Logger, logFile string, err error) {
	if log, err = logger.New(&runningConfig.Logger); err != nil {
		return
	}

	logFile = runningConfig.Logger.ActiveLogFileName

	return
}

func setup(stopChan chan os.Signal) (log zerolog.Logger, logFile string,
	resolvedConf string, runningConfig *config.Parameters, err error) {

	config.InitCLI()

	var usingDefaults bool

	defer func() {
		if runningConfig == nil && err == nil {
			err = errors.New("running configuration is nil, cannot continue")
			return
		}
		if usingDefaults && os.Getenv("HELLPOT_TEST_MODE") == "" {
			log.Warn().Msg("using default configuration!")
			print(red + "continuing with default configuration in ")
			for i := defaultConfigWarningDelaySecs; i > 0; i-- {
				print(strconv.Itoa(i))
				for i := 0; i < 5; i++ {
					time.Sleep(200 * time.Millisecond)
					print(".")
				}
			}
			print(reset + "\n")
		}
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

		if //goland:noinspection GoNilness
		!runningConfig.Logger.NoColor {
			extra.Banner()
		}
		if absResolvedConf, err := filepath.Abs(resolvedConf); err == nil {
			resolvedConf = absResolvedConf
		}
	}()

	switch {
	case config.CLIFlags.Lookup("config") == nil, config.CLIFlags.Lookup("genconfig").Value.String() == "":
		if runningConfig, usingDefaults, resolvedConf, err = resolveConfig(); err != nil {
			return
		}
		log, logFile, err = initLogger(runningConfig)
	default:
		println("loading configuration file: " + config.CLIFlags.Lookup("config").Value.String())
		if runningConfig, err = readConfig(config.CLIFlags.Lookup("config").Value.String()); err != nil {
			return
		}
		resolvedConf = config.CLIFlags.Lookup("config").Value.String()
		log, logFile, err = initLogger(runningConfig)
	}
	return
}
