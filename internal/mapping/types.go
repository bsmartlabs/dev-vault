package mapping

import "github.com/bsmartlabs/dev-vault/internal/secretprovider"

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

func (m Mode) AllowsPull() bool {
	return m == ModePull
}

func (m Mode) AllowsPush() bool {
	return m == ModePush
}

type Entry struct {
	File   string                    `json:"file"`
	Format Format                    `json:"format,omitempty"` // raw|dotenv
	Path   string                    `json:"path,omitempty"`   // default "/"
	Mode   Mode                      `json:"mode,omitempty"`   // pull|push
	Type   secretprovider.SecretType `json:"type,omitempty"`   // expected secret type
}
