/*
Copyright © 2021 Joe Kralicky

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kit

import (
	"os"

	"github.com/kralicky/kit/pkg/machinery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize kit using the existing kubeconfigs stored in ~/.kube/config",
	Run: func(cmd *cobra.Command, args []string) {
		if err := machinery.InitLocal(cmd.Flag("remote").Value.String()); err != nil {
			log.Fatal(err)
		}

		// Read config from disk
		var config *machinery.KitConfig
		var err error
		if config, err = machinery.ReadConfig(); err != nil {
			log.Fatal(err)
		}

		var localData *machinery.LocalData
		// Read local data
		log.Infof("Reading data from %s", config.KubeconfigPath)
		if localData, err = machinery.ReadLocalData(config); err != nil {
			log.Fatal(err)
		} else {
			log.Infof("Found %d contexts", len(localData.Config.Contexts))
		}

		log.Info("Checking remote connection")
		// Check remote connection
		var client *machinery.RemoteClient
		if client, err = machinery.NewRemoteClient(config); err != nil {
			log.Fatal(err)
		}
		if err := client.CheckConnection(); err != nil {
			log.Fatal(err)
		} else {
			log.Info("Remote connection success!")
		}

		if err := machinery.InitRemote(client); err != nil {
			log.Fatal(err)
		}

		// Fetch initial remote data
		log.Info("Checking if remote data exists")
		if data, err := client.LoadRemoteData(); err != nil {
			if machinery.IsNotFound(err) {
				log.Info("No remote data available")
			} else {
				log.Fatal(err)
			}
		} else {
			log.Infof("Fetching %d contexts from remote", len(data.Latest.Contexts))
			if err := data.WriteToDisk(); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	InitCmd.Flags().String("remote", "", "Vault remote URL")
	if vaultAddr, ok := os.LookupEnv("VAULT_ADDR"); ok {
		f := InitCmd.Flag("remote")
		if err := f.Value.Set(vaultAddr); err != nil {
			panic(err)
		}
		f.DefValue = vaultAddr
	} else {
		if err := InitCmd.MarkFlagRequired("remote"); err != nil {
			panic(err)
		}
	}
}
