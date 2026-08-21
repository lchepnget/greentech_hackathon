import {writable} from 'svelte/store'; import type {User} from '$lib/types'; export const session=writable<User|null>(null); export const walletBalance=writable(0);
