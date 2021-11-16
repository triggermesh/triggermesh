package zendesk

// Credential is interface of API credential
type Credential interface {
	Email() string
	Secret() string
}

// BasicAuthCredential is type of credential for Basic authentication
type BasicAuthCredential struct {
	email    string
	password string
}

// NewBasicAuthCredential creates BasicAuthCredential and returns its pointer
func NewBasicAuthCredential(email string, password string) *BasicAuthCredential {
	return &BasicAuthCredential{
		email:    email,
		password: password,
	}
}

// Email is accessor which returns email address
func (c BasicAuthCredential) Email() string {
	return c.email
}

// Secret is accessor which returns password
func (c BasicAuthCredential) Secret() string {
	return c.password
}

// APITokenCredential is type of credential for API token authentication
type APITokenCredential struct {
	email    string
	apiToken string
}

// NewAPITokenCredential creates APITokenCredential and returns its pointer
func NewAPITokenCredential(email string, apiToken string) *APITokenCredential {
	return &APITokenCredential{
		email:    email,
		apiToken: apiToken,
	}
}

// Email is accessor which returns email address
func (c APITokenCredential) Email() string {
	return c.email + "/token"
}

// Secret is accessor which returns API token
func (c APITokenCredential) Secret() string {
	return c.apiToken
}
