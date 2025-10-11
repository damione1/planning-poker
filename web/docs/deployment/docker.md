# 🚢 Docker Deployment
Finally, it is time to deploy your application to production. Assuming the previous steps were followed correctly, deploying is as simple as:

```sh
docker compose up -d
```

You can view the output of the containers with:

```sh
docker-compose logs -f
```

And after changes are made to the source, they can be applied with:

```sh
docker-compose down
docker-compose up --build -d
```

Caddy will automatically pull the domain name from the .env file and manage HTTPS for you. After a few minutes

## Congratulations!
You now have an HTTPS secured Go web app with plenty of resources to reference.

If you have built something with DeploySolo, I would love to hear it! Feel free to email me directly at matt@masoftware.net
