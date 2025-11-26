export type AuthentikConfig = {
  enabled: boolean;
  issuer?: string;
  authorize?: string;
  token?: string;
  logout?: string;
  clientId?: string;
};

type PKCEContext = {
  verifier: string;
  state: string;
  redirectUri: string;
};

let cachedConfig: AuthentikConfig | null = null;
let configPromise: Promise<AuthentikConfig> | null = null;

const PKCE_STORAGE_KEY = 'authentik_pkce';

const base64UrlEncode = (buffer: ArrayBuffer): string => {
  const bytes = new Uint8Array(buffer);
  let str = '';
  for (const byte of bytes) {
    str += String.fromCharCode(byte);
  }
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+/g, '');
};

const randomString = (length = 64): string => {
  const array = new Uint8Array(length);
  window.crypto.getRandomValues(array);
  return base64UrlEncode(array.buffer);
};

const generateCodeChallenge = async (verifier: string): Promise<string> => {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await window.crypto.subtle.digest('SHA-256', data);
  return base64UrlEncode(digest);
};

const savePKCEContext = (ctx: PKCEContext) => {
  sessionStorage.setItem(PKCE_STORAGE_KEY, JSON.stringify(ctx));
};

export const getPKCEContext = (): PKCEContext | null => {
  if (typeof window === 'undefined') return null;
  const stored = sessionStorage.getItem(PKCE_STORAGE_KEY);
  if (!stored) return null;
  try {
    return JSON.parse(stored) as PKCEContext;
  } catch {
    return null;
  }
};

export const clearPKCEContext = () => {
  if (typeof window === 'undefined') return;
  sessionStorage.removeItem(PKCE_STORAGE_KEY);
};

const fetchConfig = async (): Promise<AuthentikConfig> => {
  const response = await fetch('/api/v1/config', { credentials: 'include' });
  if (!response.ok) {
    throw new Error('Failed to load configuration');
  }
  const data = await response.json();
  const authentik = data?.authentik || {};
  return {
    enabled: authentik.enabled === true,
    issuer: authentik.issuer || '',
    authorize: authentik.authorize || '',
    token: authentik.token || '',
    logout: authentik.logout || '',
    clientId: authentik.clientId || '',
  };
};

export const getAuthentikConfig = async (): Promise<AuthentikConfig> => {
  if (cachedConfig) return cachedConfig;
  if (!configPromise) {
    configPromise = fetchConfig().then((cfg) => {
      cachedConfig = cfg;
      return cfg;
    }).finally(() => {
      configPromise = null;
    });
  }
  return configPromise;
};

export const beginAuthentikLogin = async (redirectUri?: string) => {
  const cfg = await getAuthentikConfig();
  if (!cfg.enabled || !cfg.authorize || !cfg.clientId) {
    throw new Error('Authentik login is not available');
  }

  if (typeof window === 'undefined') {
    throw new Error('Authentik login is only available in the browser');
  }

  const resolvedRedirect = redirectUri ?? `${window.location.origin}/auth/callback`;
  const verifier = randomString(64);
  const challenge = await generateCodeChallenge(verifier);
  const state = randomString(32);

  savePKCEContext({ verifier, state, redirectUri: resolvedRedirect });

  const authorizeUrl = new URL(cfg.authorize);
  authorizeUrl.searchParams.set('client_id', cfg.clientId);
  authorizeUrl.searchParams.set('response_type', 'code');
  authorizeUrl.searchParams.set('scope', 'openid email profile');
  authorizeUrl.searchParams.set('redirect_uri', resolvedRedirect);
  authorizeUrl.searchParams.set('code_challenge', challenge);
  authorizeUrl.searchParams.set('code_challenge_method', 'S256');
  authorizeUrl.searchParams.set('state', state);

  window.location.href = authorizeUrl.toString();
};

export const exchangeCodeForTokens = async (code: string, redirectUri: string, verifier: string) => {
  const response = await fetch('/api/v1/auth/authentik/token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      code,
      redirect_uri: redirectUri,
      code_verifier: verifier,
    }),
  });

  const text = await response.text();
  let payload: any = {};
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = { error: text };
    }
  }

  if (!response.ok) {
    throw new Error(payload.error || 'Failed to exchange authorization code');
  }

  if (!payload.id_token && payload.token) {
    payload.id_token = payload.token;
  }

  return payload as {
    access_token?: string;
    id_token: string;
    refresh_token?: string;
    expires_in?: number;
    token_type?: string;
  };
};

export const buildLogoutUrl = async (postLogoutRedirectUri: string, idTokenHint?: string): Promise<string | null> => {
  const cfg = await getAuthentikConfig();
  if (!cfg.enabled || !cfg.logout) {
    return null;
  }

  const url = new URL(cfg.logout);
  url.searchParams.set('post_logout_redirect_uri', postLogoutRedirectUri);
  if (idTokenHint) {
    url.searchParams.set('id_token_hint', idTokenHint);
  }
  return url.toString();
};

