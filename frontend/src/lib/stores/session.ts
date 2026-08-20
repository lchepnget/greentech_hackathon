import {writable} from 'svelte/store'; import type {User} from '$lib/api/types'; export const session=writable<User|null>(null); export const walletBalance=writable(0);
