# 🚀 Deploy
## Build
First, make sure to build your app and create a folder for logs.
```sh
go build -o output
mkdir logs
```

## Systemd
To make sure your app starts on reboot, create a systemd service on your server:
```sh
sudo vi /etc/systemd/system/app.service
```

Then insert the following block:
```
[Unit]
Description = your app

[Service]
Type           = simple
User           = root
Group          = root
LimitNOFILE    = 4096
Restart        = always
RestartSec     = 5s
StandardOutput = append:/home/admin/yourapp/logs/std.log
StandardError  = append:/home/admin/yourapp/logs/std.log
ExecStart      = /home/admin/yourapp/output serve 127.0.0.1:8090

[Install]
WantedBy = multi-user.target
```

Then run the following commands to make the service take effect

```sh
sudo systemctl daemon-reload
sudo systemctl enable app.service
sudo systemctl start app.service
sudo systemctl status app.service
```

Now your application will start in the background, and restart on reboot.

## Caddy
Make sure Caddy is installed on your server

```sh
sudo apt install caddy
```

Then add a redirection from your domain to your internal app

```
www.yourdomain.com {
    redir https://yourdomain.com{uri}
}
yourdomain.com {
    request_body {
        max_size 10MB
    }
    reverse_proxy 127.0.0.1:8090 {
        transport http {
            read_timeout 360s
        }
    }
}
```

Restart caddy
```
sudo systemctl restart caddy.service
sudo systemctl status caddy.service
```

Congrats. Your web application is now securely accessible on the internet at your domain name.
