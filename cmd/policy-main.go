/*
 * MinIO Client, (C) 2015-2019 MinIO, Inc.
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
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	policyFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "list recursively",
		},
	}
)

// Manage anonymous access to buckets and objects.
var policyCmd = cli.Command{
	Name:   "policy",
	Usage:  "manage anonymous access to buckets and objects",
	Action: mainPolicy,
	Before: initBeforeRunningCmd,
	Flags:  append(policyFlags, globalFlags...),
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] set PERMISSION TARGET
  {{.HelpName}} [FLAGS] set-json FILE TARGET
  {{.HelpName}} [FLAGS] get TARGET
  {{.HelpName}} [FLAGS] get-json TARGET
  {{.HelpName}} [FLAGS] list TARGET
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
PERMISSION:
  Allowed policies are: [none, download, upload, public].

FILE:
  A valid S3 policy JSON filepath.

EXAMPLES:
   1. Set bucket to "download" on Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} set download s3/burningman2011

   2. Set bucket to "public" on Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} set public s3/shared

   3. Set bucket to "upload" on Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} set upload s3/incoming

   4. Set policy to "public" for bucket with prefix on Amazon S3 cloud storage. 
      {{.Prompt}} {{.HelpName}} set public s3/public-commons/images

   5. Set a custom prefix based bucket policy on Amazon S3 cloud storage using a JSON file.
      {{.Prompt}} {{.HelpName}} set-json /path/to/policy.json s3/public-commons/images

   6. Get bucket permissions.
      {{.Prompt}} {{.HelpName}} get s3/shared
	
   7. Get bucket permissions in JSON format.
      {{.Prompt}} {{.HelpName}} get-json s3/shared

   8. List policies set to a specified bucket.
      {{.Prompt}} {{.HelpName}} list s3/shared

   9. List public object URLs recursively.
      {{.Prompt}} {{.HelpName}} --recursive links s3/shared/
`,
}

// policyRules contains policy rule
type policyRules struct {
	Resource string `json:"resource"`
	Allow    string `json:"allow"`
}

// String colorized access message.
func (s policyRules) String() string {
	return console.Colorize("Policy", s.Resource+" => "+s.Allow+"")
}

// JSON jsonified policy message.
func (s policyRules) JSON() string {
	policyJSONBytes, e := json.MarshalIndent(s, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(policyJSONBytes)
}

// policyMessage is container for policy command on bucket success and failure messages.
type policyMessage struct {
	Operation string                 `json:"operation"`
	Status    string                 `json:"status"`
	Bucket    string                 `json:"bucket"`
	Perms     accessPerms            `json:"permission"`
	Policy    map[string]interface{} `json:"policy,omitempty"`
}

// String colorized access message.
func (s policyMessage) String() string {
	if s.Operation == "set" {
		return console.Colorize("Policy",
			"Access permission for `"+s.Bucket+"` is set to `"+string(s.Perms)+"`")
	}
	if s.Operation == "get" {
		return console.Colorize("Policy",
			"Access permission for `"+s.Bucket+"`"+" is `"+string(s.Perms)+"`")
	}
	if s.Operation == "set-json" {
		return console.Colorize("Policy",
			"Access permission for `"+s.Bucket+"`"+" is set from `"+string(s.Perms)+"`")
	}
	if s.Operation == "get-json" {
		policy, e := json.MarshalIndent(s.Policy, "", " ")
		FatalIf(probe.NewError(e), "Unable to marshal into JSON.")
		return string(policy)
	}
	// nothing to print
	return ""
}

// JSON jsonified policy message.
func (s policyMessage) JSON() string {
	policyJSONBytes, e := json.MarshalIndent(s, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(policyJSONBytes)
}

// policyLinksMessage is container for policy links command
type policyLinksMessage struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// String colorized access message.
func (s policyLinksMessage) String() string {
	return console.Colorize("Policy", string(s.URL))
}

// JSON jsonified policy message.
func (s policyLinksMessage) JSON() string {
	policyJSONBytes, e := json.MarshalIndent(s, "", " ")
	FatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(policyJSONBytes)
}

// checkPolicySyntax check for incoming syntax.
func checkPolicySyntax(ctx *cli.Context) {
	argsLength := len(ctx.Args())
	// Always print a help message when we have extra arguments
	if argsLength > 3 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code.
	}
	// Always print a help message when no arguments specified
	if argsLength < 1 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1)
	}

	firstArg := ctx.Args().Get(0)
	secondArg := ctx.Args().Get(1)

	// More syntax checking
	switch accessPerms(firstArg) {
	case "set":
		// Always expect three arguments when setting a policy permission.
		if argsLength != 3 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
		if accessPerms(secondArg) != accessNone &&
			accessPerms(secondArg) != accessDownload &&
			accessPerms(secondArg) != accessUpload &&
			accessPerms(secondArg) != accessPublic {
			FatalIf(errDummy().Trace(),
				"Unrecognized permission `"+string(secondArg)+"`. Allowed values are [none, download, upload, public].")
		}

	case "set-json":
		// Always expect three arguments when setting a policy permission.
		if argsLength != 3 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
		// Validate the type of input file
		if filepath.Ext(string(secondArg)) != ".json" {
			FatalIf(errDummy().Trace(),
				"Unrecognized policy file format `"+string(secondArg)+"`. Only json files are accepted.")
		}

	case "get", "get-json":
		// get or get-json always expects two arguments
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
	case "list":
		// Always expect an argument after list cmd
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
	case "links":
		// Always expect an argument after links cmd
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "policy", 1)
	}
}

// Convert an accessPerms to a string recognizable by minio-go
func accessPermToString(perm accessPerms) string {
	policy := ""
	switch perm {
	case accessNone:
		policy = "none"
	case accessDownload:
		policy = "readonly"
	case accessUpload:
		policy = "writeonly"
	case accessPublic:
		policy = "readwrite"
	case accessCustom:
		policy = "custom"
	}
	return policy
}

// doSetAccess do set access.
func doSetAccess(ctx context.Context, targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := NewClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	policy := accessPermToString(targetPERMS)
	if err = clnt.SetAccess(ctx, policy, false); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// doSetAccessJSON do set access JSON.
func doSetAccessJSON(ctx context.Context, targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := NewClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	fileReader, e := os.Open(string(targetPERMS))
	if e != nil {
		FatalIf(probe.NewError(e).Trace(), "Unable to set policy for `"+targetURL+"`.")
	}
	defer fileReader.Close()

	const maxJSONSize = 120 * 1024 // 120KiB
	configBuf := make([]byte, maxJSONSize+1)

	n, e := io.ReadFull(fileReader, configBuf)
	if e == nil {
		return probe.NewError(bytes.ErrTooLarge).Trace(targetURL)
	}
	if e != io.ErrUnexpectedEOF {
		return probe.NewError(e).Trace(targetURL)
	}

	configBytes := configBuf[:n]
	if err = clnt.SetAccess(ctx, string(configBytes), true); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// Convert a minio-go permission to accessPerms type
func stringToAccessPerm(perm string) accessPerms {
	var policy accessPerms
	switch perm {
	case "none":
		policy = accessNone
	case "readonly":
		policy = accessDownload
	case "writeonly":
		policy = accessUpload
	case "readwrite":
		policy = accessPublic
	case "custom":
		policy = accessCustom
	}
	return policy
}

// doGetAccess do get access.
func doGetAccess(ctx context.Context, targetURL string) (perms accessPerms, policyStr string, err *probe.Error) {
	clnt, err := NewClient(targetURL)
	if err != nil {
		return "", "", err.Trace(targetURL)
	}
	perm, policyJSON, err := clnt.GetAccess(ctx)
	if err != nil {
		return "", "", err.Trace(targetURL)
	}
	return stringToAccessPerm(perm), policyJSON, nil
}

// doGetAccessRules do get access rules.
func doGetAccessRules(ctx context.Context, targetURL string) (r map[string]string, err *probe.Error) {
	clnt, err := NewClient(targetURL)
	if err != nil {
		return map[string]string{}, err.Trace(targetURL)
	}
	return clnt.GetAccessRules(ctx)
}

// Run policy list command
func runPolicyListCmd(args cli.Args) {
	ctx, cancelPolicyList := context.WithCancel(GlobalContext)
	defer cancelPolicyList()

	targetURL := args.First()
	policies, err := doGetAccessRules(ctx, targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			FatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			FatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}
	for k, v := range policies {
		printMsg(policyRules{Resource: k, Allow: v})
	}
}

// Run policy links command
func runPolicyLinksCmd(args cli.Args, recursive bool) {
	ctx, cancelPolicyLinks := context.WithCancel(GlobalContext)
	defer cancelPolicyLinks()

	// Get alias/bucket/prefix argument
	targetURL := args.First()

	// Fetch all policies associated to the passed url
	policies, err := doGetAccessRules(ctx, targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			FatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			FatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}

	// Extract alias from the passed argument, we'll need it to
	// construct new pathes to list public objects
	alias, path := url2Alias(targetURL)

	isRecursive := recursive
	isIncomplete := false

	// Iterate over policy rules to fetch public urls, then search
	// for objects under those urls
	for k, v := range policies {
		// Trim the asterisk in policy rules
		policyPath := strings.TrimSuffix(k, "*")
		// Check if current policy prefix is related to the url passed by the user
		if !strings.HasPrefix(policyPath, path) {
			continue
		}
		// Check if the found policy has read permission
		perm := stringToAccessPerm(v)
		if perm != accessDownload && perm != accessPublic {
			continue
		}
		// Construct the new path to search for public objects
		newURL := alias + "/" + policyPath
		clnt, err := NewClient(newURL)
		FatalIf(err.Trace(newURL), "Unable to initialize target `"+targetURL+"`.")
		// Search for public objects
		for content := range clnt.List(GlobalContext, isRecursive, isIncomplete, false, DirFirst) {
			if content.Err != nil {
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			}

			if content.Type.IsDir() && isRecursive {
				continue
			}

			// Encode public URL
			u, e := url.Parse(content.URL.String())
			errorIf(probe.NewError(e), "Unable to parse url `"+content.URL.String()+"`.")
			publicURL := u.String()

			// Construct the message to be displayed to the user
			msg := policyLinksMessage{
				Status: "success",
				URL:    publicURL,
			}
			// Print the found object
			printMsg(msg)
		}
	}
}

// Run policy cmd to fetch set permission
func runPolicyCmd(args cli.Args) {
	ctx, cancelPolicy := context.WithCancel(GlobalContext)
	defer cancelPolicy()

	var operation, policyStr string
	var probeErr *probe.Error
	perms := accessPerms(args.Get(1))
	targetURL := args.Get(2)
	if perms.isValidAccessPERM() {
		operation = "set"
		probeErr = doSetAccess(ctx, targetURL, perms)
		if probeErr == nil {
			perms, _, probeErr = doGetAccess(ctx, targetURL)
		}
	} else if perms.isValidAccessFile() {
		probeErr = doSetAccessJSON(ctx, targetURL, perms)
		operation = "set-json"
	} else {
		targetURL = args.Get(1)
		operation = "get"
		if args.First() == "get-json" {
			operation = "get-json"
		}
		perms, policyStr, probeErr = doGetAccess(ctx, targetURL)

	}
	// Upon error exit.
	if probeErr != nil {
		switch probeErr.ToGoError().(type) {
		case APINotImplemented:
			FatalIf(probeErr.Trace(), "Unable to "+operation+" policy of a non S3 url `"+targetURL+"`.")
		default:
			FatalIf(probeErr.Trace(targetURL, string(perms)),
				"Unable to "+operation+" policy `"+string(perms)+"` for `"+targetURL+"`.")
		}
	}
	policyJSON := map[string]interface{}{}
	if policyStr != "" {
		e := json.Unmarshal([]byte(policyStr), &policyJSON)
		FatalIf(probe.NewError(e), "Cannot unmarshal custom policy file.")
	}
	printMsg(policyMessage{
		Status:    "success",
		Operation: operation,
		Bucket:    targetURL,
		Perms:     perms,
		Policy:    policyJSON,
	})
}

func mainPolicy(ctx *cli.Context) error {
	// check 'policy' cli arguments.
	checkPolicySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Policy", color.New(color.FgGreen, color.Bold))

	switch ctx.Args().First() {
	case "set", "set-json", "get", "get-json":
		// policy set [download|upload|public|none] alias/bucket/prefix
		// policy set-json path-to-policy-json-file alias/bucket/prefix
		// policy get alias/bucket/prefix
		// policy get-json alias/bucket/prefix
		runPolicyCmd(ctx.Args())
	case "list":
		// policy list alias/bucket/prefix
		runPolicyListCmd(ctx.Args().Tail())
	case "links":
		// policy links alias/bucket/prefix
		runPolicyLinksCmd(ctx.Args().Tail(), ctx.Bool("recursive"))
	default:
		// Shows command example and exit
		cli.ShowCommandHelpAndExit(ctx, "policy", 1)
	}
	return nil
}
