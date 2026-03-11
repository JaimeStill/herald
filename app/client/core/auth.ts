import {
  InteractionRequiredAuthError,
  PublicClientApplication,
} from "@azure/msal-browser";

import type {
  AccountInfo,
  AuthenticationResult,
  Configuration,
} from "@azure/msal-browser";

let msalInstance: PublicClientApplication | null = null;
let config: AuthConfig | null = null;

interface AuthConfig {
  tenant_id: string;
  client_id: string;
  redirect_uri: string;
  authority: string;
  cache_location?: string;
}

function readConfig(): AuthConfig | null {
  const el = document.getElementById("herald-config");
  if (!el?.textContent) return null;

  return JSON.parse(el.textContent) as AuthConfig;
}

function scope(): string {
  return `api://${config!.client_id}/access_as_user`;
}

/**
 * Auth service wrapping `@azure/msal-browser` for Azure Entra ID authentication.
 *
 * Reads config from the server-injected `<script id="herald-config">` element.
 * When the element is absent (auth disabled), all methods are safe no-ops.
 */
export const Auth = {
  /** Whether auth is configured (config script present in DOM). */
  isEnabled(): boolean {
    return config !== null;
  },

  /** Whether an active MSAL account exists after initialization. */
  isAuthenticated(): boolean {
    return msalInstance?.getActiveAccount() !== null;
  },

  /** Returns the active MSAL account, or null if not authenticated. */
  getAccount(): AccountInfo | null {
    return msalInstance?.getActiveAccount() ?? null;
  },

  /**
   * Reads config from the DOM, creates the MSAL instance, and handles
   * any in-flight redirect. No-op when auth is disabled.
   */
  async init(): Promise<void> {
    config = readConfig();
    if (!config) return;

    const msalConfig: Configuration = {
      auth: {
        clientId: config.client_id,
        authority: config.authority,
        redirectUri: config.redirect_uri,
      },
      cache: {
        cacheLocation: config.cache_location ?? "localStorage",
      },
    };

    msalInstance = new PublicClientApplication(msalConfig);
    await msalInstance.initialize();

    const response: AuthenticationResult | null =
      await msalInstance.handleRedirectPromise();

    if (response?.account) {
      msalInstance.setActiveAccount(response.account);
    } else {
      const accounts = msalInstance.getAllAccounts();
      if (accounts.length === 1) {
        msalInstance.setActiveAccount(accounts[0]);
      }
    }
  },

  /**
   * Acquires an access token silently from the MSAL cache.
   * On interaction-required errors, redirects to login.
   */
  async getToken(forceRefresh?: boolean): Promise<string | null> {
    const account = msalInstance?.getActiveAccount();
    if (!msalInstance || !account) return null;

    try {
      const result = await msalInstance.acquireTokenSilent({
        scopes: [scope()],
        account,
        forceRefresh: forceRefresh ?? false,
      });
      return result.accessToken;
    } catch (e) {
      if (e instanceof InteractionRequiredAuthError) {
        await this.login();
      }
      return null;
    }
  },

  /** Redirects to the Azure Entra login page. No-op when auth is disabled. */
  async login(): Promise<void> {
    if (!msalInstance) return;
    await msalInstance.loginRedirect({ scopes: [scope()] });
  },

  /** Redirects to the Azure Entra logout page. No-op when auth is disabled. */
  async logout(): Promise<void> {
    if (!msalInstance) return;
    await msalInstance.logoutRedirect();
  },
};
