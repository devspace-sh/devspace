import {ApiHostname} from "./rest";
import { withAuthHeaders } from "./auth";

export default function authFetch(url: string, init?: RequestInit): Promise<Response> {
    return fetch(`${window.location.protocol}//${ApiHostname()}${url}`, {
        ...init,
        credentials: 'include',
        headers: withAuthHeaders(init && init.headers),
    })
}
