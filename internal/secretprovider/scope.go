package secretprovider

type scopedAPI struct {
	base      SecretAPI
	region    string
	projectID string
}

func (s *scopedAPI) ListSecrets(req ListSecretsInput) ([]SecretRecord, error) {
	if req.Region == "" {
		req.Region = s.region
	}
	if req.ProjectID == "" {
		req.ProjectID = s.projectID
	}
	return s.base.ListSecrets(req)
}

func (s *scopedAPI) AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error) {
	if req.Region == "" {
		req.Region = s.region
	}
	return s.base.AccessSecretVersion(req)
}

func (s *scopedAPI) CreateSecret(req CreateSecretInput) (*SecretRecord, error) {
	if req.Region == "" {
		req.Region = s.region
	}
	if req.ProjectID == "" {
		req.ProjectID = s.projectID
	}
	return s.base.CreateSecret(req)
}

func (s *scopedAPI) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	if req.Region == "" {
		req.Region = s.region
	}
	return s.base.CreateSecretVersion(req)
}
