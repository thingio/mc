/*
 * MinIO Client (C) 2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"errors"

	"github.com/minio/cli"
	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var ilmExportCmd = cli.Command{
	Name:   "export",
	Usage:  "export lifecycle configuration in JSON format",
	Action: mainILMExport,
	Before: initBeforeRunningCmd,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Exports lifecycle configuration in JSON format to STDOUT.

EXAMPLES:
  1. Export lifecycle configuration for 'testbucket' to 'lifecycle.json' file.
     {{.Prompt}} {{.HelpName}} myminio/testbucket > lifecycle.json

  2. Print lifecycle configuration for 'testbucket' to STDOUT.
     {{.Prompt}} {{.HelpName}} play/testbucket
`,
}

type ilmExportMessage struct {
	Status    string                     `json:"status"`
	Target    string                     `json:"target"`
	ILMConfig ilm.LifecycleConfiguration `json:"ilmConfig"`
}

func (i ilmExportMessage) String() string {
	msgBytes, e := json.MarshalIndent(i.ILMConfig, "", " ")
	FatalIf(probe.NewError(e), "Unable to export ILM configuration")

	return string(msgBytes)
}

func (i ilmExportMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal ILM message")

	return string(msgBytes)
}

// checkILMExportSyntax - validate arguments passed by user
func checkILMExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "export", globalErrorExitStatus)
	}
}

func mainILMExport(cliCtx *cli.Context) error {
	ctx, cancelILMExport := context.WithCancel(GlobalContext)
	defer cancelILMExport()

	checkILMExportSyntax(cliCtx)
	setILMDisplayColorScheme()

	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := NewClient(urlStr)
	FatalIf(err.Trace(args...), "Unable to initialize client for "+urlStr+".")

	ilmCfg, err := client.GetLifecycle(ctx)
	FatalIf(err.Trace(args...), "Unable to get lifecycle configuration")
	if len(ilmCfg.Rules) == 0 {
		FatalIf(probe.NewError(errors.New("lifecycle configuration not set")).Trace(urlStr),
			"Unable to export lifecycle configuration")
	}

	printMsg(ilmExportMessage{
		Status:    "success",
		Target:    urlStr,
		ILMConfig: ilmCfg,
	})

	return nil
}
