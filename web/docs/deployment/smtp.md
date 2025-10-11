# 📧 SMTP
DeploySolo uses `net/smtp` to handle email verification and password reset requests via email using SMTP.

First, acquire a SMTP provider. Here [Proton Mail](https://proton.me/support/smtp-submission) is used.

Here is an example of what you may see after viewing credentials.

![0](/public/images/doc/smtp.png)

## Update PocketBase
Update the PocketBase Admin UI with these credentials to enable email.
