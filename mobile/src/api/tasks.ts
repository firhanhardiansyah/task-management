import { Platform } from 'react-native';

export type TaskStatus = 'todo' | 'in_progress' | 'done';

export type Task = {
  id: number;
  title: string;
  description: string;
  status: TaskStatus;
  assignee: string;
  created_at: string;
  updated_at: string;
};

export type TaskInput = Pick<Task, 'title' | 'description' | 'status' | 'assignee'>;

export type TaskListResponse = {
  data: Task[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

type TaskFilters = { keyword: string; status: '' | TaskStatus; page: number; limit: number };

const localHost = Platform.OS === 'android' ? '10.0.2.2' : 'localhost';
export const API_URL = process.env.EXPO_PUBLIC_API_URL ?? `http://${localHost}:8080`;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as
      | { error?: { message?: string } }
      | null;
    throw new Error(body?.error?.message ?? `Request failed (${response.status})`);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export function listTasks(filters: TaskFilters, signal?: AbortSignal) {
  const params = new URLSearchParams({
    keyword: filters.keyword,
    status: filters.status,
    page: String(filters.page),
    limit: String(filters.limit),
    sort: 'created_at_desc',
  });
  return request<TaskListResponse>(`/api/tasks?${params.toString()}`, { signal });
}

export function createTask(input: TaskInput) {
  return request<Task>('/api/tasks', { method: 'POST', body: JSON.stringify(input) });
}

export function updateTask(id: number, input: TaskInput) {
  return request<Task>(`/api/tasks/${id}`, { method: 'PUT', body: JSON.stringify(input) });
}

export function deleteTask(id: number) {
  return request<void>(`/api/tasks/${id}`, { method: 'DELETE' });
}
