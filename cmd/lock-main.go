/*
 * MinIO Client (C) 2019-2020 MinIO, Inc.
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
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio/pkg/console"
)

var (
	lockFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "clear, c",
			Usage: "clears previously stored object lock configuration",
		},
	}
)

var lockCmd = cli.Command{
	Name:   "lock",
	Usage:  "set and get object lock configuration",
	Action: mainLock,
	Before: initBeforeRunningCmd,
	Flags:  append(lockFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [governance | compliance] VALIDITY

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
VALIDITY:
  This argument must be formatted like Nd or Ny where 'd' denotes days and 'y' denotes years e.g. 10d, 3y.

EXAMPLES:
   1. Set object lock configuration
     $ {{.HelpName}} myminio/mybucket compliance 30d

   2. Get object lock configuration
     $ {{.HelpName}} myminio/mybucket

   3. Clear object lock configuration
     $ {{.HelpName}} --clear myminio/mybucket
`,
}

// Structured message depending on the type of console.
type lockCmdMessage struct {
	Enabled  string              `json:"enabled"`
	Mode     minio.RetentionMode `json:"mode"`
	Validity string              `json:"validity"`
	Status   string              `json:"status"`
}

// Colorized message for console printing.
func (m lockCmdMessage) String() string {
	if m.Mode == "" {
		return "Object lock configuration cleared successfully"
	}

	return fmt.Sprintf("%s mode is enabled for %s", console.Colorize("Mode", m.Mode), console.Colorize("Validity", m.Validity))
}

// JSON'ified message for scripting.
func (m lockCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(m, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// lock - set/get object lock configuration.
func lock(urlStr string, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, clearLock bool) error {
	client, err := NewClient(urlStr)
	if err != nil {
		FatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	ctx, cancelLock := context.WithCancel(GlobalContext)
	defer cancelLock()

	if clearLock || mode != "" {
		err = client.SetObjectLockConfig(ctx, mode, validity, unit)
		FatalIf(err, "Cannot enable object lock configuration on the specified bucket.")
	} else {
		mode, validity, unit, err = client.GetObjectLockConfig(ctx)
		FatalIf(err, "Cannot get object lock configuration on the specified bucket.")
	}

	printMsg(lockCmdMessage{
		Enabled:  "Enabled",
		Mode:     mode,
		Validity: fmt.Sprintf("%d%s", validity, unit),
		Status:   "success",
	})

	return nil
}

func parseRetentionValidity(validityStr string, m minio.RetentionMode) (uint64, minio.ValidityUnit, *probe.Error) {
	if !m.IsValid() {
		return 0, "", errInvalidArgument().Trace(fmt.Sprintf("invalid retention mode '%v'", m))
	}

	unitStr := string(validityStr[len(validityStr)-1])
	validityStr = validityStr[:len(validityStr)-1]
	validity, e := strconv.ParseUint(validityStr, 10, 64)
	if e != nil {
		return 0, "", probe.NewError(e).Trace(validityStr)
	}

	var unit minio.ValidityUnit
	switch unitStr {
	case "d", "D":
		unit = minio.Days
	case "y", "Y":
		unit = minio.Years
	default:
		return 0, "", errInvalidArgument().Trace(unitStr)
	}

	return validity, unit, nil
}

// main for lock command.
func mainLock(ctx *cli.Context) error {
	console.SetColor("Mode", color.New(color.FgCyan, color.Bold))
	console.SetColor("Validity", color.New(color.FgYellow))

	// lock specific flags.
	clearLock := ctx.Bool("clear")

	args := ctx.Args()

	var urlStr string
	var mode minio.RetentionMode
	var validity uint64
	var unit minio.ValidityUnit
	var err *probe.Error

	switch l := len(args); l {
	case 1:
		urlStr = args[0]
	case 3:
		urlStr = args[0]
		if clearLock {
			FatalIf(errInvalidArgument().Trace(urlStr), "clear flag must be passed with target alone")
		}

		mode = minio.RetentionMode(strings.ToUpper(args[1]))
		validity, unit, err = parseRetentionValidity(args[2], mode)
		if err != nil {
			FatalIf(err.Trace(args...), "unable to parse input arguments")
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "lock", 1)
	}

	return lock(urlStr, mode, validity, unit, clearLock)
}
