import urllib.request, json
req = urllib.request.Request("https://api.github.com/repos/devspace-sh/devspace/issues?state=open&sort=created&direction=desc&per_page=40")
req.add_header('User-Agent', 'python-urllib')
try:
    with urllib.request.urlopen(req) as response:
        data = json.loads(response.read().decode())
        for issue in data:
            if issue['number'] in [3179, 3174, 3106]:
                print(f"--- ISSUE #{issue['number']} ---")
                print(issue['body'][:1000])
except Exception as e:
    print(e)
