package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// CredentialsFilter narrows credential listings.
type CredentialsFilter struct {
	Name                    string
	Type                    string
	IncludeDefaultAppliance bool
	CreatedAfter            *time.Time
	CreatedBefore           *time.Time
	OrderColumn             string
	OrderAscending          *bool
	MaxItems                int
}

type credentialsResponse struct {
	Data       []Credential     `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// Credential represents a generic credentials record.
type Credential struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Description  string    `json:"description"`
	Type         string    `json:"type"`
	CreationTime time.Time `json:"creationTime"`
	UniqueID     string    `json:"uniqueId,omitempty"`
}

// Credentials returns credentials records respecting the provided filter.
func (c *Client) Credentials(ctx context.Context, filter CredentialsFilter) ([]Credential, error) {
	results := make([]Credential, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.IncludeDefaultAppliance {
			query.Set("includeDefaultApplianceCreds", "true")
		}
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp credentialsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/credentials", query, &resp); err != nil {
			return nil, err
		}

		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

// CredentialDetail wraps a credentials record with the raw payload.
type CredentialDetail struct {
	Credential Credential
	Raw        map[string]any
}

// CredentialByID fetches a credentials record and returns the decoded payload.
func (c *Client) CredentialByID(ctx context.Context, id string) (*CredentialDetail, error) {
	path := fmt.Sprintf("/api/v1/credentials/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode credential payload: %w", err)
	}

	var cred Credential
	if err := json.Unmarshal(raw, &cred); err != nil {
		return nil, fmt.Errorf("decode credential payload: %w", err)
	}

	return &CredentialDetail{
		Credential: cred,
		Raw:        payload,
	}, nil
}

// CloudCredentialsFilter narrows cloud credential listings.
type CloudCredentialsFilter struct {
	Name        string
	Type        string
	OrderColumn string
	OrderAsc    *bool
	MaxItems    int
}

type cloudCredentialsResponse struct {
	Data       []CloudCredential `json:"data"`
	Pagination paginationResult  `json:"pagination"`
}

// CloudCredential represents cloud credential metadata.
type CloudCredential struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	Type         string    `json:"type"`
	LastModified time.Time `json:"lastModified"`
}

// CloudCredentials lists cloud credential records.
func (c *Client) CloudCredentials(ctx context.Context, filter CloudCredentialsFilter) ([]CloudCredential, error) {
	results := make([]CloudCredential, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAsc != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAsc))
			}
		}

		var resp cloudCredentialsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/cloudCredentials", query, &resp); err != nil {
			return nil, err
		}

		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

// CloudCredentialDetail includes raw payload for a cloud credential.
type CloudCredentialDetail struct {
	Credential CloudCredential
	Raw        map[string]any
}

// CloudCredentialByID fetches a specific cloud credential record.
func (c *Client) CloudCredentialByID(ctx context.Context, id string) (*CloudCredentialDetail, error) {
	path := fmt.Sprintf("/api/v1/cloudCredentials/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode cloud credential payload: %w", err)
	}

	var cred CloudCredential
	if err := json.Unmarshal(raw, &cred); err != nil {
		return nil, fmt.Errorf("decode cloud credential payload: %w", err)
	}

	return &CloudCredentialDetail{
		Credential: cred,
		Raw:        payload,
	}, nil
}

type cloudHelperApplianceResponse struct {
	Data       []CloudHelperAppliance `json:"data"`
	Pagination paginationResult       `json:"pagination"`
}

// CloudHelperAppliance describes a helper appliance.
type CloudHelperAppliance struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	VMName         string `json:"vmName"`
	Location       string `json:"location"`
	StorageAccount string `json:"storageAccount"`
	ResourceGroup  string `json:"resourceGroup"`
	VirtualNetwork string `json:"virtualNetwork"`
	Subnet         string `json:"subnet"`
	SSHPort        int    `json:"SSHPort"`
}

// CloudCredentialHelperAppliances lists helper appliances for an Azure compute credential.
func (c *Client) CloudCredentialHelperAppliances(ctx context.Context, id string) ([]CloudHelperAppliance, error) {
	path := fmt.Sprintf("/api/v1/cloudCredentials/%s/helperAppliances", id)
	var resp cloudHelperApplianceResponse
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// EncryptionPasswordsFilter narrows encryption password listings.
type EncryptionPasswordsFilter struct {
	Hint        string
	OrderColumn string
	OrderAsc    *bool
	MaxItems    int
}

type encryptionPasswordsResponse struct {
	Data       []EncryptionPassword `json:"data"`
	Pagination paginationResult     `json:"pagination"`
}

// EncryptionPassword represents an encryption password metadata record.
type EncryptionPassword struct {
	ID               string    `json:"id"`
	Hint             string    `json:"hint"`
	UniqueID         string    `json:"uniqueId"`
	ModificationTime time.Time `json:"modificationTime"`
	IsImported       bool      `json:"isImported"`
}

// EncryptionPasswords retrieves encryption password metadata.
func (c *Client) EncryptionPasswords(ctx context.Context, filter EncryptionPasswordsFilter) ([]EncryptionPassword, error) {
	results := make([]EncryptionPassword, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Hint != "" {
			query.Set("hintFilter", filter.Hint)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAsc != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAsc))
			}
		}

		var resp encryptionPasswordsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/encryptionPasswords", query, &resp); err != nil {
			return nil, err
		}

		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

// EncryptionPasswordDetail wraps an encryption password detail payload.
type EncryptionPasswordDetail struct {
	Password EncryptionPassword
	Raw      map[string]any
}

// EncryptionPasswordByID fetches an encryption password detail payload.
func (c *Client) EncryptionPasswordByID(ctx context.Context, id string) (*EncryptionPasswordDetail, error) {
	path := fmt.Sprintf("/api/v1/encryptionPasswords/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode encryption password payload: %w", err)
	}

	var pwd EncryptionPassword
	if err := json.Unmarshal(raw, &pwd); err != nil {
		return nil, fmt.Errorf("decode encryption password payload: %w", err)
	}

	return &EncryptionPasswordDetail{
		Password: pwd,
		Raw:      payload,
	}, nil
}

// KMSServersFilter narrows KMS server listings.
type KMSServersFilter struct {
	Name        string
	Type        string
	OrderColumn string
	OrderAsc    *bool
	MaxItems    int
}

type kmsServersResponse struct {
	Data       []KMSServer      `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// KMSServer represents a key management server metadata record.
type KMSServer struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	Type                   string `json:"type"`
	Port                   int    `json:"port"`
	ServerCertificateThumb string `json:"serverCertificateThumbprint"`
	ClientCertificateThumb string `json:"clientCertificateThumbprint"`
}

// KMSServers retrieves KMS servers respecting the filter.
func (c *Client) KMSServers(ctx context.Context, filter KMSServersFilter) ([]KMSServer, error) {
	results := make([]KMSServer, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAsc != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAsc))
			}
		}

		var resp kmsServersResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/kmsServers", query, &resp); err != nil {
			return nil, err
		}

		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

// KMSServerDetail wraps a KMS server with its raw payload.
type KMSServerDetail struct {
	Server KMSServer
	Raw    map[string]any
}

// KMSServerByID fetches an individual KMS server detail.
func (c *Client) KMSServerByID(ctx context.Context, id string) (*KMSServerDetail, error) {
	path := fmt.Sprintf("/api/v1/kmsServers/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode kms server payload: %w", err)
	}

	var server KMSServer
	if err := json.Unmarshal(raw, &server); err != nil {
		return nil, fmt.Errorf("decode kms server payload: %w", err)
	}

	return &KMSServerDetail{
		Server: server,
		Raw:    payload,
	}, nil
}
