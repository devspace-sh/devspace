import {ApiHostname} from "./rest";

export default function authFetch(url: string): Promise<Response> {
    return fetch(`http://${ApiHostname()}${url}`, {
        headers: new Headers({
            "Authorization": (window as any).token,
        })
    })
}