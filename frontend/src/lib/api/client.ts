import { browser } from '$app/environment';
import { PUBLIC_API_BASE_URL } from '$env/static/public';

const configuredBase = PUBLIC_API_BASE_URL?.replace(/\/$/, '');
const base = configuredBase
	? `${configuredBase.includes('://') ? configuredBase : `https://${configuredBase}`}${configuredBase.endsWith('/api') ? '' : '/api'}`
	: (browser && window.location.hostname.endsWith('onrender.com')
		? 'https://regenfeed.onrender.com/api'
		: browser ? `http://${window.location.hostname}:3001/api` : 'http://localhost:3001/api');
const REQUEST_TIMEOUT_MS = 15_000;

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
	const abortFromCaller = () => controller.abort();
	init.signal?.addEventListener('abort', abortFromCaller, { once: true });

	try {
		const isForm = typeof FormData !== 'undefined' && init.body instanceof FormData;
		const headers = isForm
			? { ...(init.headers || {}) }
			: { 'Content-Type': 'application/json', ...(init.headers || {}) };
		const response = await fetch(`${base}${path}`, {
			...init,
			headers,
			credentials: 'include',
			signal: controller.signal
		});

		if (response.status === 401 && typeof window !== 'undefined' && !path.startsWith('/auth/')) {
			window.dispatchEvent(new CustomEvent('regenfeed:unauthorized'));
		}

		if (!response.ok) {
			let message = `Request failed (${response.status})`;
			try {
				const body = await response.json();
				message = typeof body?.error === 'string'
					? body.error
					: body?.error?.message || body?.message || message;
			} catch {
				// Keep the HTTP status message when no JSON error body is available.
			}
			throw new Error(message);
		}

		return response.status === 204 ? undefined as T : response.json();
	} catch (error) {
		if (error instanceof DOMException && error.name === 'AbortError') {
			throw new Error('The server took too long to respond. Please try again.');
		}
		if (error instanceof TypeError && error.message.toLowerCase().includes('fetch')) {
			throw new Error(`Unable to reach the API at ${base}. Check that the backend is running.`);
		}
		throw error;
	} finally {
		clearTimeout(timeout);
		init.signal?.removeEventListener('abort', abortFromCaller);
	}
}
