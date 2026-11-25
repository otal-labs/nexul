const PULL_REQUEST_URL = /https?:\/\/github\.com\/([\w.-]+\/[\w.-]+)\/pull\/(\d+)/;

// The first pull request the reply links, for the chip under the prose.
export const pullRequestLink = (text: string): { url: string; label: string } | null => {
  const match = PULL_REQUEST_URL.exec(text);
  if (!match) return null;
  return { url: match[0], label: `${match[1]}#${match[2]}` };
};
