export function buildChatGptExplainUrl(appid) {
  const composeUrl = `https://raw.githubusercontent.com/besoeasy/yantr/refs/heads/main/apps/${appid}/compose.yml`;

  const query = `Understand this Yantr Docker stack: ${composeUrl}
(Yantr handles deployment, so skip Docker/installation commands)

Instructions:
- Fetch the compose.yml from the URL (use the raw GitHub URL if needed).
- Understand the project using the compose contents AND the compose labels (especially yantr.* labels).
- If present, treat the yantr.website label as the canonical project/app website.
- Use the internet to fetch/verify up-to-date info (official docs/GitHub) before listing features and alternatives.

Tell me:
1. What does this app do?
2. 5 main features of this app

If you want to know about Yantr : https://github.com/besoeasy/yantr

Make this a well-informed list, keep it short and minimal, and ask if I want to know more.`;

  return `https://chatgpt.com/?q=${encodeURIComponent(query)}`;
}
