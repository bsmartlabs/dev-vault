package mapping

import "github.com/bsmartlabs/dev-vault/internal/secrettype"

type Format string

const (
	FormatRaw    Format = "raw"
	FormatDotenv Format = "dotenv"
)

type Mode string

const (
	ModePull Mode = "pull"
	ModePush Mode = "push"
)

func (m Mode) IsSupportedCommandMode() bool {
	return m == ModePull || m == ModePush
}

func (m Mode) AllowsPull() bool {
	return m == ModePull
}

func (m Mode) AllowsPush() bool {
	return m == ModePush
}

func (m Mode) AllowsCommand(commandMode Mode) bool {
	return m == commandMode
}

type Entry struct {
	File   string          `json:"file"`
	Format Format          `json:"format,omitempty"` // raw|dotenv
	Path   string          `json:"path,omitempty"`   // default "/"
	Mode   Mode            `json:"mode,omitempty"`   // pull|push
	Type   secrettype.Name `json:"type,omitempty"`   // expected secret type
}
