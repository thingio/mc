// +build go1.13

/*
 * MinIO Client (C) 2014-2019 MinIO, Inc.
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

/*
 * Below main package has canonical imports for 'go get' and 'go build'
 * to work with all other clones of github.com/minio/mc repository. For
 * more information refer https://golang.org/doc/go1.4#canonicalimports
 */

package main // import "github.com/minio/mc"
import (
	mc "github.com/minio/mc/cmd"
	"strings"
)

func main() {
	//mc.Main(os.Args)
	setRomteStorgeCommands := strings.Fields("mc config host add myminio http://192.168.10.128:9000 minioadmin minioadmin")
	mirrorIgnoreDelete := strings.Fields("mc mirror --ignore-delete /home/kevin/Documents/minio/data myminio/testbucket")
	mc.Main(setRomteStorgeCommands)
	mc.Main(mirrorIgnoreDelete)
}
