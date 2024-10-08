package main

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/withz/ptun/app/config"
	"github.com/withz/ptun/cmd/node/service"
	"github.com/withz/ptun/pkg/tools"
)

var (
	ClientName string
	ConfigFile string

	rootCmd = &cobra.Command{
		Use:   "",
		Short: "Ptun",
		Run:   Run,
	}

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Ptun",
		Run:   Run,
	}

	confCmd = &cobra.Command{
		Use:   "config",
		Short: "Config",
		Run:   Config,
	}
)

func init() {
	rootCmd.AddCommand(runCmd, confCmd)
	rootCmd.PersistentFlags().StringVarP(&ClientName, "name", "n", "", "verbose output")
	rootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "verbose output")
}

func Run(cmd *cobra.Command, args []string) {
	var err error
	if ConfigFile == "" {
		err = config.InitClient()
	} else {
		err = config.InitClientPath(ConfigFile)
	}

	if err != nil {
		panic(err)
	}

	c := service.NewService()

	logrus.Info("client starting")
	err = c.Start(context.Background())
	if err != nil {
		logrus.Errorf("client start failed, %s", err.Error())
		return
	}
	logrus.Info("client started")

	tools.QuitSignalWait()

	logrus.Info("client shutdowning")
	c.Close()
	logrus.Info("client stopped")
}

func Config(cmd *cobra.Command, args []string) {
	var err error
	if ConfigFile == "" {
		err = config.InitClient()
	} else {
		err = config.InitClientPath(ConfigFile)
	}
	if err != nil {
		panic(err)
	}
	cfg := config.Client()
	p, _ := json.Marshal(cfg)
	logrus.Infof(string(p))
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	// logrus.SetReportCaller(true)

	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err.Error())
	}
}
