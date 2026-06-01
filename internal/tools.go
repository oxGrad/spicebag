//go:build tools

package internal

import (
	_ "github.com/BurntSushi/toml"
	_ "github.com/mark3labs/mcp-go/mcp"
	_ "github.com/stretchr/testify/assert"
	_ "gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)
