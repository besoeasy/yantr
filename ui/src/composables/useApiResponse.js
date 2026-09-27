export async function readJsonResponse(response) {
  return response.json().catch(() => ({}));
}

export function getApiErrorMessage(payload, fallbackMessage = "Request failed") {
  if (typeof payload?.message === "string" && payload.message.trim()) {
    return payload.message.trim();
  }

  if (typeof payload?.error === "string" && payload.error.trim()) {
    return payload.error.trim();
  }

  return fallbackMessage;
}

export async function expectApiSuccess(response, fallbackMessage = "Request failed") {
  const data = await readJsonResponse(response);
  if (!response.ok || !data?.success) {
    throw new Error(getApiErrorMessage(data, fallbackMessage));
  }

  return data;
}