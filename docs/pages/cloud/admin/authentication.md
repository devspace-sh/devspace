---
title: Authentication
---

Currently DevSpace Cloud only supports email signup (This will change soon and other forms of authentication, such as LDAP and SAML will be supported). In order to configure who is allowed to signup to your DevSpace Cloud instance, navigate to the Admin -> Authentication tab.  

You are able to create patterns for email addresses that are allowed to signup and you can also create patterns for email addresses that should be not allowed to register. With no patterns, nobody is allowed to register. On signup, DevSpace Cloud will try to validate the email address by sending a confirmation email to the address with the provided SMTP settings. If you haven't specified any SMTP settings, DevSpace Cloud will NOT verify the email address.    

Example patterns could be:
- `*` allows every email to register
- `*@my-domain.com` allows every email with the ending my-domain.com to register
- you can also combine allow and disallow patterns: allow `*@my-domain.com` and disallow `dev*@my-domain.com` allows everybody with the domain ending my-domain.com to register, except the ones that start with dev
