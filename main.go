// The client: the launcher and every command. It lives at the root of the
// module so that `go install <module>@latest` names the binary after the
// module, which is the product's name.
package main

import (
	"os"

	"github.com/iAmLakshya/simstim/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
