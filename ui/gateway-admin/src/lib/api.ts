function parseErrorMessage(text: string, status: number) {
  if (!text) {
    return `Request failed: ${status}`;
  }

  try {
    const payload = JSON.parse(text) as {
      error?: { message?: string };
      message?: string;
    };
    return payload.error?.message || payload.message || text;
  } catch {
    return text;
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    headers: { Accept: "application/json" },
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(parseErrorMessage(text, response.status));
  }
  return response.json() as Promise<T>;
}

export async function apiPost<T>(path: string, body: Record<string, unknown> = {}): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(parseErrorMessage(text, response.status));
  }
  return response.json() as Promise<T>;
}
