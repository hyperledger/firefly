// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"encoding/base64"
	"net/http"
	"strings"
)

const (
	ffLogo16b64 = `iVBORw0KGgoAAAANSUhEUgAAABAAAAAJCAYAAAA7KqwyAAAACXBIWXMAAA7DAAAOwwHHb6hkAAAAGXRFWHRTb2Z0d2FyZQB3d3cuaW5rc2NhcGUub3Jnm+48GgAAABR0RVh0VGl0bGUAR3JvdXAgMiBDb3B5IDLy1GqnAAABbUlEQVQokXWRsUtbcRzE774vJqXJZKjtIDYOghWVTqJTp0ZBRARHd1dtRdsK7aDgX+HqLiLqgw6FkuJYIyKC8HTRWKRJg6FJ3vtdh1r6KK833n3u+x2O+I9eDgbvPGBAZFvQuV8urCdxLA5fZzvUTMfNCOoXuSbTexNNTmuR4VXGMYhzbWZaLA5dHAB6CujT30gPCXoSBwCAxJngmgAbsd8vAF5YK98zSXALsEY9l1rwy73zDG1ZsgDSJaBLOZ0ztGW/3Dtfz6UWAGsQ3PrdvdfEUDDmpLcw3sDxcWdnZcSZfTC5tHPeavV7vgTDLZy6jNzYLxe+AAARU3H4Ohu1wgdeRzgyM7u53dd33DSLePR11PZ258bDlp146dRP/+jJ3Z9OKn7gPribeB6wVs2z9LmYEwiPrmmifTztvv13BUuaJmqz5ln0Y/H1GywurcCJ9dCFtcQZk0wAmB49nup6dLUuwVW+da/sHD7zk7hffDua5kGYlhAAAAAASUVORK5CYII=`
	ffLogo32b64 = `iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAACXBIWXMAAA7DAAAOwwHHb6hkAAAAGXRFWHRTb2Z0d2FyZQB3d3cuaW5rc2NhcGUub3Jnm+48GgAAABR0RVh0VGl0bGUAR3JvdXAgMiBDb3B5IDLy1GqnAAADvklEQVRYhe2VXWyTZRTH/+d5v9oVFhiwZEOylo/FybqZNVnAqJnSbUxNQBNIXNQLnWPMKSZgNGSuzkiiyfxIgDFE0YiOxAt2gQKlSBbjroQoK8OYqS0baCbKJltp+7bvc7xgQ6Kb7G3UC9Pf3fucc/7/k5PnOS+QJUuWDFh7e8S9evWwc+q7bvmg4feeX5qJlppJkZTUkxuzjlR7I8sJIm4RDwgpnwRQbFeL7BbUrhzOY7IGAcwDcR9AOSAqB/OEKXOW9A7kT9jRE3aS/aXREpD1LQgmCWw6HvbcfUe4qBISWwFmXcTP3Vs+vNiOJq259cICRbPcs0oGHmHwRijYAIuSALmuRTgGkIsgP4GgDpb4dFZ6UkSoxnv+TYCftdP1PwUBb6jjc8T23HFrDhMaAOw0df1ll5myZiOQAvcTABVUNpv8mK4pumm2AXiaGO9cmau0Xr+Etd5oPQNdDJwWadQHv3H/NL0Mk788WiMkbQPgnzw8IQV3nDjjPg4QT1dVWxItkCq6CfAR0BQMu7snp/AHa8q/L1ak8jGAQiJ+LNjvOTYV8/lOaXnmovUgq41Y3AbCEbboVRJIMPEWYtSD+Aew6DDH5Qe9UU/iuvnK6D0s8BEDo0LQxuCZooGp2F+eYZU74tBz8RqYWkC8iyy1XSryUWJsBaGAmAVY9qSBpz47u3QEAOrKLtwiydzDku4HyAQwCuK9pmbs0lPJFjC1MqHboelNh08XXr3Rb8Y9UF0WeZiY9gLIIcaoJO4EiBYu/Pl5VZh86dfCtJSinZicROntBYVDfPlyvppI5GyTjDzB1MyE+QCuMvGmUL/n4HQ+M+6BUL/nICuWj8ENuqEXhcKeAEATDiMm9x+oMhqbXnE59PgOh3OitbF5h7HvvWqHqqQZjCuhsCegG3oRgxtYsXwzmQM3WcWhr5cNAhi88WxsbIF26ssqPPjQfvj9hzShWHC5xvF57wNIJh3KVN7kqN//O/2bNvBniKAwCC+1vY2Kij40NbfDTBno2h3AuYEKKEJCMNla77ZWMTOs+fN+Se3cvQ6JhBObG4+hZfNhaFoSe/bdB8OIW5J42mc4Exn9DVcUh/H6WxvQ98Va6FoSlatOZiIDwOYECHxxZGSx2nPocbYsBXfedRSVq04iaTrw4YEtMhabq0jmi/Y0bcFU7R16ThFWYFH+j9TyzIvORNyFzs5A4rexvCSnxQvBs56uf7GBa9SWRAugyXYwPQECM+NdRU23Hv1qxaVM9DKmrnRoWU3pd0v+U9MsWf53/A40fXWPXEBGEwAAAABJRU5ErkJggg==`
)

var ffLogo16, ffLogo32 []byte

func init() {
	ffLogo16, _ = base64.StdEncoding.DecodeString(ffLogo16b64)
	ffLogo32, _ = base64.StdEncoding.DecodeString(ffLogo32b64)
}

func favIcons(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("content-type", "image/png")
	if strings.Contains(req.URL.Path, "16") {
		_, _ = res.Write(ffLogo16)
	} else {
		_, _ = res.Write(ffLogo32)
	}
}
