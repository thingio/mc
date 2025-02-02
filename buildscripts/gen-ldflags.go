//go:build ignore
// +build ignore

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func genLDFlags(now time.Time) string {
	version := now.Format(time.RFC3339)
	releaseTag := releaseTag(version)
	copyrightYear := fmt.Sprintf("%d", now.Year())

	var ldflagsStr string
	ldflagsStr = "-s -w -X github.com/minio/mc/cmd.Version=" + version + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.CopyrightYear=" + copyrightYear + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.ReleaseTag=" + releaseTag + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.CommitID=" + commitID() + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.ShortCommitID=" + commitID()[:12]
	return ldflagsStr
}

// genReleaseTag prints release tag to the console for easy git tagging.
func releaseTag(version string) string {
	relPrefix := "DEVELOPMENT"
	if prefix := os.Getenv("MC_RELEASE"); prefix != "" {
		relPrefix = prefix
	}

	relTag := strings.Replace(version, " ", "-", -1)
	relTag = strings.Replace(relTag, ":", "-", -1)
	relTag = strings.Replace(relTag, ",", "", -1)
	return relPrefix + "." + relTag
}

// commitID returns the abbreviated commit-id hash of the last commit.
func commitID() string {
	// git log --format="%h" -n1
	var (
		commit []byte
		e      error
	)
	cmdName := "git"
	cmdArgs := []string{"log", "--format=%H", "-n1"}
	if commit, e = exec.Command(cmdName, cmdArgs...).Output(); e != nil {
		fmt.Fprintln(os.Stderr, "Error generating git commit-id: ", e)
		os.Exit(1)
	}

	return strings.TrimSpace(string(commit))
}

func main() {
	fmt.Println(genLDFlags(time.Now().UTC()))
}
