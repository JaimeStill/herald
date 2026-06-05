// Package azure implements the Provider interface for Azure OpenAI Service.
//
// It supports deployment-based routing with API key, Entra ID (bearer token),
// and managed identity authentication. The AzureTokenSource handles dynamic
// token acquisition for managed identity scenarios.
//
// Register explicitly when needed:
//
//	import "github.com/tailored-agentic-units/provider/azure"
//	azure.Register()
package azure
