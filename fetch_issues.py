import urllib.request, json
from datetime import datetime, timezone, timedelta

req = urllib.request.Request("https://api.github.com/repos/devspace-sh/devspace/issues?state=open&sort=created&direction=desc&per_page=40")
req.add_header('User-Agent', 'python-urllib')

try:
    with urllib.request.urlopen(req) as response:
        data = json.loads(response.read().decode())
        
        three_months_ago = datetime.now(timezone.utc) - timedelta(days=90)
        
        print("===== RECENT OPEN ISSUES (NO PRS, UNASSIGNED) =====")
        count = 0
        for issue in data:
            # Skip if PR
            if "pull_request" in issue:
                continue
            
            # Skip if assigned
            if issue.get("assignees"):
                continue
            
            # Check date
            created_at = datetime.strptime(issue['created_at'], "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
            if created_at < three_months_ago:
                continue
            
            print(f"\n[{issue['number']}] {issue['title']} (Created: {issue['created_at'].split('T')[0]})")
            print(f"URL: {issue['html_url']}")
            # Print first 200 chars of body
            body_preview = (issue['body'] or "")[:200].replace('\n', ' ').replace('\r', '')
            print(f"Excerpt: {body_preview}...")
            
            count += 1
            if count >= 10:
                break
                
except Exception as e:
    print(e)
