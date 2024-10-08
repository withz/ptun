package main

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/withz/ptun/app/config"
	"github.com/withz/ptun/cmd/hub/service"
	"github.com/withz/ptun/pkg/tools"
)

var (
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
	rootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "verbose output")
}

func Run(cmd *cobra.Command, args []string) {
	var err error
	if ConfigFile == "" {
		err = config.InitServer()
	} else {
		err = config.InitServerPath(ConfigFile)
	}
	if err != nil {
		panic(err)
	}

	s := service.NewService()

	logrus.Info("server starting")
	err = s.Start(context.Background())
	if err != nil {
		logrus.Errorf("server start failed, %s", err.Error())
		return
	}
	logrus.Info("server started")

	tools.QuitSignalWait()

	logrus.Info("server shutdowning")
	s.Close()
	logrus.Info("server stopped")
}

func Config(cmd *cobra.Command, args []string) {
	err := config.InitServer()
	if err != nil {
		panic(err)
	}
	p, _ := json.Marshal(config.Server())
	logrus.Infof(string(p))
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	// logrus.SetReportCaller(true)

	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err.Error())
	}
}
