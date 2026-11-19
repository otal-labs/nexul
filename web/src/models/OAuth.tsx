export const getOAuthConsentKey = "getOAuthConsent";

// What the consent page shows before the person decides.
export interface OAuthConsent {
  client_name: string;
  redirect_uri: string;
}

export interface OAuthDecision {
  redirect_to: string;
}

const unfollowable = new Set(["javascript:", "data:", "vbscript:", "file:", "blob:", "about:"]);

// The server only answers with a registered redirect; this keeps a script URL from ever reaching location.assign.
export const isFollowableRedirect = (to: string): boolean => {
  try {
    return !unfollowable.has(new URL(to).protocol);
  } catch {
    return false;
  }
};

// Where approving sends the person back to: a site's host, or an app's scheme with it, since any app may claim a host.
export const redirectTarget = (uri: string): string => {
  try {
    const url = new URL(uri);
    if (url.protocol === "https:" || url.protocol === "http:") return url.host;
    return `${url.protocol}//${url.host}`;
  } catch {
    return uri;
  }
};
