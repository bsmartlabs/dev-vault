package secretsync

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type ListQuery struct {
	NameContains []string
	NameRegex    *regexp.Regexp
	Path         string
	Type         secretprovider.SecretType
}

type ListRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type MappingTarget struct {
	Name  string
	Entry mapping.Entry
}

type PullResult struct {
	Name     string
	File     string
	Revision uint32
	Type     secretprovider.SecretType
}

type BatchFailure struct {
	Name string
	Err  error
}

type PushOptions struct {
	Description     string
	DisablePrevious bool
	CreateMissing   bool
}

type PushResult struct {
	Name     string
	Revision uint32
}

type BatchSummary struct {
	Operation string
	Failed    int
	Total     int
}

type BatchOperationError struct {
	Operation string
	Failed    int
	Total     int
}

func (e *BatchOperationError) Error() string {
	return fmt.Sprintf("%s completed with failures: %d/%d failed", e.Operation, e.Failed, e.Total)
}

func (s BatchSummary) ErrorOrNil() error {
	if s.Failed == 0 {
		return nil
	}
	return &BatchOperationError{
		Operation: s.Operation,
		Failed:    s.Failed,
		Total:     s.Total,
	}
}

type PullBatchResult struct {
	Succeeded []PullResult
	Failed    []BatchFailure
	Summary   BatchSummary
}

type PushBatchResult struct {
	Succeeded []PushResult
	Failed    []BatchFailure
	Summary   BatchSummary
}

type Config struct {
	Root string
}

type PathResolver func(rootDir string, rel string) (string, error)

type Dependencies struct {
	Now         func() time.Time
	Hostname    func() (string, error)
	ResolvePath PathResolver
}

type Service struct {
	cfg         Config
	api         secretprovider.SecretAPI
	now         func() time.Time
	hostname    func() (string, error)
	resolvePath PathResolver
}

func New(cfg Config, api secretprovider.SecretAPI, deps Dependencies) (Service, error) {
	if api == nil {
		return Service{}, errors.New("secretsync: nil secret api")
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	hostname := deps.Hostname
	if hostname == nil {
		hostname = os.Hostname
	}
	resolvePath := deps.ResolvePath
	if resolvePath == nil {
		resolvePath = resolveFile
	}
	return Service{
		cfg:         cfg,
		api:         api,
		now:         now,
		hostname:    hostname,
		resolvePath: resolvePath,
	}, nil
}
