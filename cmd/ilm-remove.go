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

	"github.com/minio/cli"
	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "id of the lifecycle rule",
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "force flag is to be used when deleting all lifecycle configuration rules for the bucket",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "delete all lifecycle configuration rules of the bucket, force flag enforced",
	},
}

var ilmRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove (if any) existing lifecycle configuration rule with the id",
	Action: mainILMRemove,
	Before: initBeforeRunningCmd,
	Flags:  append(ilmRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

DESCRIPTION:
  Remove a lifecycle configuration rule for the bucket by ID, optionally you can remove
  all the lifecycle rules on a bucket with '--all --force' option.

EXAMPLES:
  1. Remove the lifecycle management configuration rule given by ID "Documents" for testbucket on alias 'myminio'. ID is case sensitive.
     {{.Prompt}} {{.HelpName}} --id "Documents" myminio/testbucket

  2. Remove ALL the lifecycle management configuration rules for testbucket on alias 'myminio'. 
     Because the result is complete removal, the use of --force flag is enforced.
     {{.Prompt}} {{.HelpName}} --all --force myminio/testbucket
`,
}

type ilmRmMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Target string `json:"target"`
	All    bool   `json:"all"`
}

func (i ilmRmMessage) String() string {
	msg := "Rule ID `" + i.ID + "` from target " + i.Target + " removed."
	if i.All {
		msg = "Rules for `" + i.Target + "` removed."
	}
	return console.Colorize(ilmThemeResultSuccess, msg)
}

func (i ilmRmMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func checkILMRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "remove", globalErrorExitStatus)
	}

	ilmAll := ctx.Bool("all")
	ilmForce := ctx.Bool("force")
	forceChk := (ilmAll && ilmForce) || (!ilmAll && !ilmForce)
	if !forceChk {
		FatalIf(errInvalidArgument(),
			"It is mandatory to specify --all and --force flag together for mc "+ctx.Command.FullName()+".")
	}
	if ilmAll && ilmForce {
		return
	}

	ilmID := ctx.String("id")
	if ilmID == "" {
		FatalIf(errInvalidArgument().Trace(ilmID), "ilm ID cannot be empty")
	}
}

func mainILMRemove(cliCtx *cli.Context) error {
	ctx, cancelILMImport := context.WithCancel(GlobalContext)
	defer cancelILMImport()

	checkILMRemoveSyntax(cliCtx)
	setILMDisplayColorScheme()
	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := NewClient(urlStr)
	FatalIf(err.Trace(args...), "Unable to initialize client for "+urlStr+".")

	ilmCfg, err := client.GetLifecycle(ctx)
	FatalIf(err.Trace(urlStr), "Unable to fetch lifecycle rules")

	ilmAll := cliCtx.Bool("all")
	ilmForce := cliCtx.Bool("force")

	if ilmAll && ilmForce {
		ilmCfg.Rules = nil // Remove all rules
	} else {
		ilmCfg, err = ilm.RemoveILMRule(ilmCfg, cliCtx.String("id"))
		FatalIf(err.Trace(urlStr, cliCtx.String("id")), "Unable to remove rule by id")
	}

	FatalIf(client.SetLifecycle(ctx, ilmCfg).Trace(urlStr), "Unable to set lifecycle rules")

	printMsg(ilmRmMessage{
		Status: "success",
		ID:     cliCtx.String("id"),
		All:    ilmAll,
		Target: urlStr,
	})

	return nil
}
