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
	"github.com/kralicky/kit/pkg/machinery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var FetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch the contents of the remote store",
	Run: func(cmd *cobra.Command, args []string) {
		var config *machinery.KitConfig
		var err error
		if config, err = machinery.ReadConfig(); err != nil {
			log.Fatal(err)
		}
		var client *machinery.RemoteClient
		if client, err = machinery.NewRemoteClient(config); err != nil {
			log.Fatal(err)
		}
		if cache, err := client.LoadRemoteData(); err != nil {
			log.Fatal(err)
		} else {
			if err := cache.WriteToDisk(); err != nil {
				log.Fatal(err)
			}
		}
		log.Info("Done.")
	},
}
