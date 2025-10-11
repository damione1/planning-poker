# 🏷️ DNS
Purchase a domain at your preferred DNS provider. Here [gandi.net](https://www.gandi.net) is used.

After purchasing a domain, set the `A` value to the public IP address retrieved in the earlier step.

This gives your website a user friendly name to connect with, and allows the server to retrieve an HTTPS certificate for security.

Note that DNS records can take several hours to propagate.

![0](/public/images/doc/dns.png)

## Update .env
In your project directory, edit the .env file.

```
cd ~/deploysolo/
vi .env
```

And set the following environment variable.

```
DOMAIN_NAME=yourdomain.com
```
