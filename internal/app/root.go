package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/formation-res/open-location-hub-cli/internal/build"
	"github.com/formation-res/open-location-hub-cli/internal/cli"
	"github.com/formation-res/open-location-hub-cli/internal/openapi"
	"github.com/formation-res/open-location-hub-cli/internal/output"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	envFile := cli.DefaultEnvFile()
	envValues, _ := cli.LoadEnvFile(envFile)
	cfg := &cli.Config{
		BaseURL: cli.ResolveValue(envValues, "OLH_BASE_URL", "http://localhost:8080"),
		Token:   cli.ResolveValue(envValues, "OLH_TOKEN", ""),
		Timeout: 30 * time.Second,
		EnvFile: envFile,
		OAuth: cli.OAuthConfig{
			TokenURL:     cli.ResolveValue(envValues, "OLH_OAUTH_TOKEN_URL", ""),
			ClientID:     cli.ResolveValue(envValues, "OLH_OAUTH_CLIENT_ID", ""),
			ClientSecret: cli.ResolveValue(envValues, "OLH_OAUTH_CLIENT_SECRET", ""),
			Username:     cli.ResolveValue(envValues, "OLH_OAUTH_USERNAME", ""),
			Password:     cli.ResolveValue(envValues, "OLH_OAUTH_PASSWORD", ""),
			Scope:        cli.ResolveValue(envValues, "OLH_OAUTH_SCOPE", "openid email profile"),
			GrantType:    cli.ResolveValue(envValues, "OLH_OAUTH_GRANT_TYPE", "password"),
			Audience:     cli.ResolveValue(envValues, "OLH_OAUTH_AUDIENCE", ""),
		},
	}
	printer := output.New(false, false)

	cmd := &cobra.Command{
		Use:           "olh",
		Short:         "CLI for Open Location Hub",
		Long:          "Open Location Hub CLI with typed REST CRUD, ingest helpers, JSON-RPC, and websocket streaming.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			timeout, _ := cmd.Flags().GetDuration("timeout")
			jsonOut, _ := cmd.Flags().GetBool("json")
			noColor, _ := cmd.Flags().GetBool("no-color")
			baseURL, _ := cmd.Flags().GetString("base-url")
			hubEndpoint, _ := cmd.Flags().GetString("hub-endpoint")
			token, _ := cmd.Flags().GetString("token")
			envFileValue, _ := cmd.Flags().GetString("env-file")
			oauthTokenURL, _ := cmd.Flags().GetString("oauth-token-url")
			tokenEndpoint, _ := cmd.Flags().GetString("token-endpoint")
			oauthClientID, _ := cmd.Flags().GetString("oauth-client-id")
			clientID, _ := cmd.Flags().GetString("client-id")
			oauthClientSecret, _ := cmd.Flags().GetString("oauth-client-secret")
			clientSecret, _ := cmd.Flags().GetString("client-secret")
			oauthUsername, _ := cmd.Flags().GetString("oauth-username")
			oauthPassword, _ := cmd.Flags().GetString("oauth-password")
			oauthScope, _ := cmd.Flags().GetString("oauth-scope")
			oauthGrantType, _ := cmd.Flags().GetString("oauth-grant-type")
			oauthAudience, _ := cmd.Flags().GetString("oauth-audience")
			envValues, err := cli.LoadEnvFile(envFileValue)
			if err != nil {
				return err
			}

			cfg.Timeout = timeout
			cfg.JSON = jsonOut
			cfg.NoColor = noColor
			cfg.EnvFile = envFileValue
			baseURL = firstNonEmpty(hubEndpoint, baseURL)
			oauthTokenURL = firstNonEmpty(tokenEndpoint, oauthTokenURL)
			oauthClientID = firstNonEmpty(clientID, oauthClientID)
			oauthClientSecret = firstNonEmpty(clientSecret, oauthClientSecret)
			cfg.BaseURL = resolveFlagOrEnv(flags, "base-url", baseURL, envValues, "OLH_BASE_URL")
			cfg.Token = resolveFlagOrEnv(flags, "token", token, envValues, "OLH_TOKEN")
			cfg.OAuth.TokenURL = resolveFlagOrEnv(flags, "oauth-token-url", oauthTokenURL, envValues, "OLH_OAUTH_TOKEN_URL")
			cfg.OAuth.ClientID = resolveFlagOrEnv(flags, "oauth-client-id", oauthClientID, envValues, "OLH_OAUTH_CLIENT_ID")
			cfg.OAuth.ClientSecret = resolveFlagOrEnv(flags, "oauth-client-secret", oauthClientSecret, envValues, "OLH_OAUTH_CLIENT_SECRET")
			cfg.OAuth.Username = resolveFlagOrEnv(flags, "oauth-username", oauthUsername, envValues, "OLH_OAUTH_USERNAME")
			cfg.OAuth.Password = resolveFlagOrEnv(flags, "oauth-password", oauthPassword, envValues, "OLH_OAUTH_PASSWORD")
			cfg.OAuth.Scope = resolveFlagOrEnv(flags, "oauth-scope", oauthScope, envValues, "OLH_OAUTH_SCOPE")
			cfg.OAuth.GrantType = resolveFlagOrEnv(flags, "oauth-grant-type", oauthGrantType, envValues, "OLH_OAUTH_GRANT_TYPE")
			cfg.OAuth.Audience = resolveFlagOrEnv(flags, "oauth-audience", oauthAudience, envValues, "OLH_OAUTH_AUDIENCE")
			*printer = *output.New(jsonOut, noColor)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().String("base-url", cfg.BaseURL, "Hub base URL or origin, e.g. http://localhost:8080")
	cmd.PersistentFlags().String("hub-endpoint", cfg.BaseURL, "Alias for --base-url")
	cmd.PersistentFlags().String("token", cfg.Token, "Bearer token. Can also be set via OLH_TOKEN")
	cmd.PersistentFlags().String("env-file", cfg.EnvFile, "Optional env file. Defaults to ~/.openlocationhub.env")
	cmd.PersistentFlags().Bool("json", false, "Emit machine-readable JSON output")
	cmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	cmd.PersistentFlags().Duration("timeout", 30*time.Second, "HTTP timeout")
	cmd.PersistentFlags().String("oauth-token-url", cfg.OAuth.TokenURL, "OAuth token endpoint URL")
	cmd.PersistentFlags().String("token-endpoint", cfg.OAuth.TokenURL, "Alias for --oauth-token-url")
	cmd.PersistentFlags().String("oauth-client-id", cfg.OAuth.ClientID, "OAuth client ID")
	cmd.PersistentFlags().String("client-id", cfg.OAuth.ClientID, "Alias for --oauth-client-id")
	cmd.PersistentFlags().String("oauth-client-secret", cfg.OAuth.ClientSecret, "OAuth client secret")
	cmd.PersistentFlags().String("client-secret", cfg.OAuth.ClientSecret, "Alias for --oauth-client-secret")
	cmd.PersistentFlags().String("oauth-username", cfg.OAuth.Username, "OAuth username for password grant")
	cmd.PersistentFlags().String("oauth-password", cfg.OAuth.Password, "OAuth password for password grant")
	cmd.PersistentFlags().String("oauth-scope", cfg.OAuth.Scope, "OAuth scope")
	cmd.PersistentFlags().String("oauth-grant-type", cfg.OAuth.GrantType, "OAuth grant type: password or client_credentials")
	cmd.PersistentFlags().String("oauth-audience", cfg.OAuth.Audience, "OAuth audience, if required by the provider")

	cmd.AddCommand(loginCommand(cfg, printer))
	cmd.AddCommand(versionCommand(printer))
	cmd.AddCommand(authCommand(cfg, printer))
	cmd.AddCommand(newResourceCommand(cfg, printer, zonesSpec()))
	cmd.AddCommand(newResourceCommand(cfg, printer, trackablesSpec()))
	cmd.AddCommand(newResourceCommand(cfg, printer, providersSpec()))
	cmd.AddCommand(newResourceCommand(cfg, printer, fencesSpec()))
	cmd.AddCommand(locationsCommand(cfg, printer))
	cmd.AddCommand(proximitiesCommand(cfg, printer))
	cmd.AddCommand(rpcCommand(cfg, printer))
	cmd.AddCommand(websocketCommand(cfg, printer))
	cmd.AddCommand(openapiCommand(printer))
	cmd.AddCommand(completionCommand(cmd))
	return cmd
}

func versionCommand(printer *output.Printer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printer.Print(map[string]string{
				"version": build.Version,
				"commit":  build.Commit,
				"date":    build.Date,
			})
		},
	}
}

func completionCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenBashCompletionV2(os.Stdout, true)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenZshCompletion(os.Stdout)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenFishCompletion(os.Stdout, true)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})
	return cmd
}

func openapiCommand(printer *output.Printer) *cobra.Command {
	return &cobra.Command{
		Use:   "openapi",
		Short: "Print the vendored OpenAPI contract path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printer.Print(map[string]string{
				"spec": "api/omlox-hub.v0.yaml",
			})
		},
	}
}

func apiClient(cfg *cli.Config) (*openapi.ClientWithResponses, error) {
	if err := cfg.EnsureToken(context.Background()); err != nil {
		return nil, err
	}
	return cfg.APIClient()
}

func authCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication helpers",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "token",
		Short: "Fetch an OAuth access token using env vars or ~/.openlocationhub.env",
		RunE: func(cmd *cobra.Command, args []string) error {
			tokenResp, err := cfg.FetchToken(cmd.Context())
			if err != nil {
				return err
			}
			cfg.Token = tokenResp.AccessToken
			if printer.JSON {
				return printer.Print(tokenResp)
			}
			return printer.Print(tokenResp.AccessToken)
		},
	})
	return cmd
}

func loginCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Validate credentials and save hub auth settings",
		Long:  "Fetches an OAuth token, verifies the hub endpoint with an authenticated request, and saves the resolved settings to the env file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(cfg.BaseURL) == "" {
				return fmt.Errorf("hub endpoint is required")
			}
			if strings.TrimSpace(cfg.OAuth.TokenURL) == "" {
				return fmt.Errorf("token endpoint is required")
			}
			if strings.TrimSpace(cfg.OAuth.ClientID) == "" {
				return fmt.Errorf("client ID is required")
			}
			if strings.TrimSpace(cfg.OAuth.ClientSecret) == "" {
				return fmt.Errorf("client secret is required")
			}

			tokenResp, err := cfg.FetchToken(cmd.Context())
			if err != nil {
				return err
			}
			cfg.Token = tokenResp.AccessToken
			if err := validateHubAccess(cmd.Context(), cfg); err != nil {
				return err
			}

			values := map[string]string{
				"OLH_BASE_URL":            cfg.BaseURL,
				"OLH_OAUTH_TOKEN_URL":     cfg.OAuth.TokenURL,
				"OLH_OAUTH_CLIENT_ID":     cfg.OAuth.ClientID,
				"OLH_OAUTH_CLIENT_SECRET": cfg.OAuth.ClientSecret,
				"OLH_OAUTH_USERNAME":      cfg.OAuth.Username,
				"OLH_OAUTH_PASSWORD":      cfg.OAuth.Password,
				"OLH_OAUTH_SCOPE":         cfg.OAuth.Scope,
				"OLH_OAUTH_GRANT_TYPE":    cfg.OAuth.GrantType,
				"OLH_OAUTH_AUDIENCE":      cfg.OAuth.Audience,
			}
			saveToken, _ := cmd.Flags().GetBool("save-token")
			if saveToken {
				values["OLH_TOKEN"] = cfg.Token
			}
			if err := cli.WriteEnvFile(cfg.EnvFile, values); err != nil {
				return err
			}

			if printer.JSON {
				return printer.Print(map[string]any{
					"saved":      true,
					"env_file":   cfg.EnvFile,
					"hub":        cfg.BaseURL,
					"token_url":  cfg.OAuth.TokenURL,
					"grant_type": cfg.OAuth.GrantType,
				})
			}
			printer.Success("saved login settings to %s", cfg.EnvFile)
			return nil
		},
	}
	cmd.Flags().Bool("save-token", false, "Also persist the fetched access token as OLH_TOKEN")
	return cmd
}

func zonesSpec() resourceSpec {
	return resourceSpec{
		Name:     "zones",
		Singular: "zone",
		ReadArg:  "zone-id",
		WriteArg: "zone-id",
		Example:  "Example: olh zones create -f zone.json",
		List: func(ctx context.Context, cfg *cli.Config) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.ListZonesWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Get: func(ctx context.Context, cfg *cli.Config, id string) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			zoneID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.GetZoneWithResponse(ctx, zoneID)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Create: func(ctx context.Context, cfg *cli.Config, path string) (any, error) {
			body, err := decodePayload[openapi.CreateZoneJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.CreateZoneWithResponse(ctx, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusCreated, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		},
		Update: func(ctx context.Context, cfg *cli.Config, id, path string) (any, error) {
			body, err := decodePayload[openapi.UpdateZoneJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			zoneID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.UpdateZoneWithResponse(ctx, zoneID, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Delete: func(ctx context.Context, cfg *cli.Config, id string) error {
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			zoneID, err := parseUUID(id)
			if err != nil {
				return err
			}
			resp, err := client.DeleteZoneWithResponse(ctx, zoneID)
			if err != nil {
				return err
			}
			return expectStatus(resp.HTTPResponse, http.StatusNoContent, resp.Body)
		},
	}
}

func trackablesSpec() resourceSpec {
	return resourceSpec{
		Name:     "trackables",
		Singular: "trackable",
		ReadArg:  "trackable-id",
		WriteArg: "trackable-id",
		Example:  "Example: olh trackables create -f trackable.json",
		List: func(ctx context.Context, cfg *cli.Config) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.ListTrackablesWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Get: func(ctx context.Context, cfg *cli.Config, id string) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			trackableID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.GetTrackableWithResponse(ctx, trackableID)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Create: func(ctx context.Context, cfg *cli.Config, path string) (any, error) {
			body, err := decodePayload[openapi.CreateTrackableJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.CreateTrackableWithResponse(ctx, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusCreated, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		},
		Update: func(ctx context.Context, cfg *cli.Config, id, path string) (any, error) {
			body, err := decodePayload[openapi.UpdateTrackableJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			trackableID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.UpdateTrackableWithResponse(ctx, trackableID, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Delete: func(ctx context.Context, cfg *cli.Config, id string) error {
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			trackableID, err := parseUUID(id)
			if err != nil {
				return err
			}
			resp, err := client.DeleteTrackableWithResponse(ctx, trackableID)
			if err != nil {
				return err
			}
			return expectStatus(resp.HTTPResponse, http.StatusNoContent, resp.Body)
		},
	}
}

func providersSpec() resourceSpec {
	return resourceSpec{
		Name:     "providers",
		Singular: "provider",
		ReadArg:  "provider-id",
		WriteArg: "provider-id",
		Example:  "Example: olh providers create -f provider.json",
		List: func(ctx context.Context, cfg *cli.Config) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.ListProvidersWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Get: func(ctx context.Context, cfg *cli.Config, id string) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.GetProviderWithResponse(ctx, openapi.ProviderId(id))
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Create: func(ctx context.Context, cfg *cli.Config, path string) (any, error) {
			body, err := decodePayload[openapi.CreateProviderJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.CreateProviderWithResponse(ctx, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusCreated, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		},
		Update: func(ctx context.Context, cfg *cli.Config, id, path string) (any, error) {
			body, err := decodePayload[openapi.UpdateProviderJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.UpdateProviderWithResponse(ctx, openapi.ProviderId(id), body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Delete: func(ctx context.Context, cfg *cli.Config, id string) error {
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			resp, err := client.DeleteProviderWithResponse(ctx, openapi.ProviderId(id))
			if err != nil {
				return err
			}
			return expectStatus(resp.HTTPResponse, http.StatusNoContent, resp.Body)
		},
	}
}

func fencesSpec() resourceSpec {
	return resourceSpec{
		Name:     "fences",
		Singular: "fence",
		ReadArg:  "fence-id",
		WriteArg: "fence-id",
		Example:  "Example: olh fences create -f fence.json",
		List: func(ctx context.Context, cfg *cli.Config) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.ListFencesWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Get: func(ctx context.Context, cfg *cli.Config, id string) (any, error) {
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			fenceID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.GetFenceWithResponse(ctx, fenceID)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Create: func(ctx context.Context, cfg *cli.Config, path string) (any, error) {
			body, err := decodePayload[openapi.CreateFenceJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			resp, err := client.CreateFenceWithResponse(ctx, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusCreated, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		},
		Update: func(ctx context.Context, cfg *cli.Config, id, path string) (any, error) {
			body, err := decodePayload[openapi.UpdateFenceJSONRequestBody](path)
			if err != nil {
				return nil, err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return nil, err
			}
			fenceID, err := parseUUID(id)
			if err != nil {
				return nil, err
			}
			resp, err := client.UpdateFenceWithResponse(ctx, fenceID, body)
			if err != nil {
				return nil, err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		},
		Delete: func(ctx context.Context, cfg *cli.Config, id string) error {
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			fenceID, err := parseUUID(id)
			if err != nil {
				return err
			}
			resp, err := client.DeleteFenceWithResponse(ctx, fenceID)
			if err != nil {
				return err
			}
			return expectStatus(resp.HTTPResponse, http.StatusNoContent, resp.Body)
		},
	}
}

func locationsCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locations",
		Short: "Post provider location updates",
	}
	post := &cobra.Command{
		Use:   "post --file locations.json",
		Short: "POST /v2/providers/locations",
		Long:  "Send an array of Location objects to the hub. JSON or YAML is accepted.",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			body, err := decodePayload[openapi.PostProviderLocationsJSONRequestBody](file)
			if err != nil {
				return err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			resp, err := client.PostProviderLocationsWithResponse(cmd.Context(), body)
			if err != nil {
				return err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusAccepted, resp.Body); err != nil {
				return err
			}
			if printer.JSON {
				return printer.Print(map[string]any{"accepted": true, "status": resp.StatusCode()})
			}
			printer.Success("locations accepted")
			return nil
		},
	}
	post.Flags().StringP("file", "f", "", "Read request body from file or - for stdin")
	must(post.MarkFlagRequired("file"))
	cmd.AddCommand(post)
	return cmd
}

func proximitiesCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proximities",
		Short: "Post provider proximity updates",
	}
	post := &cobra.Command{
		Use:   "post --file proximities.json",
		Short: "POST /v2/providers/proximities",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			body, err := decodePayload[openapi.PostProviderProximitiesJSONRequestBody](file)
			if err != nil {
				return err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			resp, err := client.PostProviderProximitiesWithResponse(cmd.Context(), body)
			if err != nil {
				return err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusAccepted, resp.Body); err != nil {
				return err
			}
			if printer.JSON {
				return printer.Print(map[string]any{"accepted": true, "status": resp.StatusCode()})
			}
			printer.Success("proximities accepted")
			return nil
		},
	}
	post.Flags().StringP("file", "f", "", "Read request body from file or - for stdin")
	must(post.MarkFlagRequired("file"))
	cmd.AddCommand(post)
	return cmd
}

func rpcCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "JSON-RPC control plane commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "available",
		Short: "List available RPC methods",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			resp, err := client.GetRPCAvailableWithResponse(cmd.Context())
			if err != nil {
				return err
			}
			if err := expectStatus(resp.HTTPResponse, http.StatusOK, resp.Body); err != nil {
				return err
			}
			return printer.Print(resp.JSON200)
		},
	})
	call := &cobra.Command{
		Use:   "call --file request.json",
		Short: "Invoke PUT /v2/rpc",
		Long:  "Send a JSON-RPC 2.0 request body to the hub. The response is emitted as raw JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			body, err := decodePayload[openapi.PutRPCJSONRequestBody](file)
			if err != nil {
				return err
			}
			client, err := apiClient(cfg)
			if err != nil {
				return err
			}
			resp, err := client.PutRPCWithResponse(cmd.Context(), body)
			if err != nil {
				return err
			}
			if err := expectStatuses(resp.HTTPResponse, resp.Body, http.StatusOK, http.StatusNoContent); err != nil {
				return err
			}
			if resp.StatusCode() == http.StatusNoContent {
				return printer.Print(map[string]any{"accepted": true, "notification": true})
			}
			var raw any
			if err := json.Unmarshal(resp.Body, &raw); err != nil {
				return err
			}
			return printer.Print(raw)
		},
	}
	call.Flags().StringP("file", "f", "", "Read request body from file or - for stdin")
	must(call.MarkFlagRequired("file"))
	cmd.AddCommand(call)
	return cmd
}

func expectStatus(resp *http.Response, code int, body []byte) error {
	return expectStatuses(resp, body, code)
}

func expectStatuses(resp *http.Response, body []byte, codes ...int) error {
	if resp == nil {
		return fmt.Errorf("no HTTP response")
	}
	for _, code := range codes {
		if resp.StatusCode == code {
			return nil
		}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, msg)
}

func parseUUID(id string) (openapi_types.UUID, error) {
	return uuid.Parse(id)
}

func resolveFlagOrEnv(flags interface{ Changed(string) bool }, flagName, current string, env map[string]string, envKey string) string {
	if flags.Changed(flagName) || aliasChanged(flags, flagName) {
		return current
	}
	return cli.ResolveValue(env, envKey, current)
}

func aliasChanged(flags interface{ Changed(string) bool }, flagName string) bool {
	switch flagName {
	case "base-url":
		return flags.Changed("hub-endpoint")
	case "oauth-token-url":
		return flags.Changed("token-endpoint")
	case "oauth-client-id":
		return flags.Changed("client-id")
	case "oauth-client-secret":
		return flags.Changed("client-secret")
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateHubAccess(ctx context.Context, cfg *cli.Config) error {
	client, err := cfg.APIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetRPCAvailableWithResponse(ctx)
	if err == nil && resp != nil && resp.HTTPResponse != nil {
		switch resp.StatusCode() {
		case http.StatusOK, http.StatusForbidden:
			return nil
		case http.StatusUnauthorized:
			return fmt.Errorf("hub authentication failed: %s", strings.TrimSpace(string(resp.Body)))
		}
	}
	zonesResp, zonesErr := client.ListZonesWithResponse(ctx)
	if zonesErr != nil {
		if err != nil {
			return fmt.Errorf("hub validation failed: %w", err)
		}
		return zonesErr
	}
	switch zonesResp.StatusCode() {
	case http.StatusOK, http.StatusForbidden:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("hub authentication failed: %s", strings.TrimSpace(string(zonesResp.Body)))
	default:
		return fmt.Errorf("hub validation failed with status %d: %s", zonesResp.StatusCode(), strings.TrimSpace(string(zonesResp.Body)))
	}
}
