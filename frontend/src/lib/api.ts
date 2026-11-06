/**
 * 统一请求封装
 * - 自动拼接 VITE_API_BASE
 * - 自动附带 JWT（token 存在 localStorage: nodal_token）
 * - 统一解包后端的 { code, message, data } 信封结构
 */

export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api';
const TOKEN_KEY = 'nodal_token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string | null): void {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

function buildHeaders(extra?: HeadersInit): Headers {
  const headers = new Headers(extra);
  if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  const token = getToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: buildHeaders(init.headers),
  });

  if (!res.ok) {
    let message = res.statusText;
    let code = res.status;
    try {
      const body = (await res.json()) as Partial<ApiEnvelope<unknown>>;
      message = body.message ?? message;
      code = body.code ?? code;
    } catch {
      // 非 JSON 响应，保留 HTTP 状态文案
    }
    throw new ApiError(message, res.status, code);
  }

  const body = (await res.json()) as ApiEnvelope<T>;
  return body.data;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, payload?: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(payload ?? {}) }),
  put: <T>(path: string, payload?: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(payload ?? {}) }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};

/** 健康检查（不带信封结构，直接返回后端原始 JSON） */
export async function fetchHealth(): Promise<{ status: string; version: string }> {
  const res = await fetch(`${API_BASE}/healthz`);
  if (!res.ok) throw new ApiError('健康检查失败', res.status, res.status);
  return (await res.json()) as { status: string; version: string };
}
