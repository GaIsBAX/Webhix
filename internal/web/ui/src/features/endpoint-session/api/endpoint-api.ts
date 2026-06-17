import type { ApiResponse, WebhookRequest } from '../../../entities/request/model/types';

export interface Endpoint {
  id: number;
  token: string;
  name?: string;
  url: string;
  createdAt: string;
}

export async function fetchEndpoints(): Promise<Endpoint[]> {
  const response = await fetch('/api/endpoints');
  const json = (await response.json()) as ApiResponse<Endpoint[]>;
  if (!json.success) return [];
  return json.body ?? [];
}

export async function createEndpoint(): Promise<string> {
  const response = await fetch('/api/endpoints', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: '' }),
  });
  const json = (await response.json()) as ApiResponse<{ token: string }>;
  if (!json.success) throw new Error(json.error?.message || 'Failed');
  if (!json.body?.token) throw new Error('Endpoint response did not include a token');
  return json.body.token;
}

export async function fetchRequests(token: string): Promise<WebhookRequest[]> {
  const response = await fetch(`/api/endpoints/${token}/requests`);
  const json = (await response.json()) as ApiResponse<WebhookRequest[]>;
  if (!json.success) return [];
  return json.body || [];
}

export interface HookResponse {
  statusCode: number;
  headers: Record<string, string>;
  body: string;
}

export async function fetchHookResponse(token: string): Promise<HookResponse> {
  const response = await fetch(`/api/endpoints/${token}/response`);
  const json = (await response.json()) as ApiResponse<HookResponse>;
  if (!json.success || !json.body) return { statusCode: 200, headers: {}, body: '' };
  return json.body;
}

export async function saveHookResponse(token: string, data: HookResponse): Promise<void> {
  const response = await fetch(`/api/endpoints/${token}/response`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  const json = (await response.json()) as ApiResponse<unknown>;
  if (!json.success) throw new Error(json.error?.message || 'Failed to save');
}

export interface Notification {
  telegramBotToken: string;
  telegramChatId: string;
  proxyUrl: string;
}

export async function fetchNotification(token: string): Promise<Notification> {
  const response = await fetch(`/api/endpoints/${token}/notifications`);
  const json = (await response.json()) as ApiResponse<Notification>;
  if (!json.success || !json.body)
    return { telegramBotToken: '', telegramChatId: '', proxyUrl: '' };
  return json.body;
}

export async function saveNotification(token: string, data: Notification): Promise<void> {
  const response = await fetch(`/api/endpoints/${token}/notifications`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  const json = (await response.json()) as ApiResponse<unknown>;
  if (!json.success) throw new Error(json.error?.message || 'Failed to save');
}

export function connectEvents(
  token: string,
  handlers: {
    onOpen?: () => void;
    onRequest?: (request: WebhookRequest) => void;
    onError?: () => void;
  },
): EventSource {
  const source = new EventSource(`/api/endpoints/${token}/events`);

  source.onopen = () => handlers.onOpen?.();
  source.onmessage = (event) => {
    try {
      handlers.onRequest?.(JSON.parse(event.data) as WebhookRequest);
    } catch {
      // Ignore malformed SSE payloads; the stream can continue.
    }
  };
  source.onerror = () => handlers.onError?.();

  return source;
}
