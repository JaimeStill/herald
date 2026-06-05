package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const defaultScope = "https://cognitiveservices.azure.com/.default"

// AzureTokenSource acquires bearer tokens using Azure managed identity.
// Uses ManagedIdentityCredential for direct managed identity authentication.
// Token caching is handled internally by the Azure SDK.
type AzureTokenSource struct {
	cred  azcore.TokenCredential
	scope string
}

// NewAzureTokenSource creates a new AzureTokenSource.
// scope defaults to "https://cognitiveservices.azure.com/.default" if empty.
// clientID is optional — when provided, it configures user-assigned managed identity
// via ManagedIdentityCredentialOptions.ID. When empty, system-assigned identity is used.
func NewAzureTokenSource(scope, clientID string) (*AzureTokenSource, error) {
	if scope == "" {
		scope = defaultScope
	}

	opts := &azidentity.ManagedIdentityCredentialOptions{}

	if clientID != "" {
		opts.ID = azidentity.ClientID(clientID)
	}

	cred, err := azidentity.NewManagedIdentityCredential(opts)
	if err != nil {
		return nil, fmt.Errorf("create managed identity credential: %w", err)
	}

	return &AzureTokenSource{
		cred:  cred,
		scope: scope,
	}, nil
}

// GetToken acquires a bearer token for the configured scope.
// The Azure SDK handles token caching and refresh internally.
func (s *AzureTokenSource) GetToken(ctx context.Context) (string, error) {
	token, err := s.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{s.scope},
	})
	if err != nil {
		return "", fmt.Errorf("acquire token for scope %q: %w", s.scope, err)
	}

	return token.Token, nil
}
