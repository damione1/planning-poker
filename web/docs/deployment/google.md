# 🔐 Google OAuth
DeploySolo supports authentication with OAuth. Although other providers could be added, "Log In with Google" is already set up.

After creating a Google account, an OAuth client needs to be created at the [Google Cloud Console](https://console.cloud.google.com/apis/credentials)

---

First create an OAuth 2.0 Client

![0](/public/images/doc/google/google-0.png)


Then add the following as an Authorized JavaScript Origin
```
https://yourdomain.com
```

And the following as an Authorized Redirect URL
```
https://yourdomain.com/oauth/google/callback
```

![1](/public/images/doc/google/google-1.png)

After creating the OAauth 2.0 Client, make note of the Client ID and Client Secret
![2](/public/images/doc/google/google-2.png)

## Update PocketBase
Update the PocketBase Admin UI with these credentials to enable log in with Google.
