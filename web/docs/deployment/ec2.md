# 🖥️ EC2

This is a walk through for setting up an EC2 server with the correct security permissions.

---

After creating an [AWS](https://aws.amazon.com/) account, go to the EC2 dashboard and click "Launch Instance".
![0](/public/images/doc/ec2/ec2-0.png)

Select a distribution of Linux as the OS. Here Debian is chosen.
![1](/public/images/doc/ec2/ec2-1.png)

Specify an instance type and create a key pair for SSH log in. Resource requirements vary by cost and amount of users. Make sure to keep the downloaded `your_key.pem` file.

But a nano instance or greater is recommended for SQLite mode.

And a medium instance or greater is recommended for Docker Compose mode.
![2](/public/images/doc/ec2/ec2-2.png)

For security settings, ensure SSH and HTTPS traffic is enabled.
![3](/public/images/doc/ec2/ec2-3.png)

Return to the global EC2 dashboard and select your newly created instance. Make a note of the public IPv4 address.
![4](/public/images/doc/ec2/ec2-4.png)

Now enter your terminal and locate your key. Likely in `~/Downloads/your_key.pem`. Type:
```
chmod 600 ~/Downloads/your_key.pem
ssh -i "~/Downloads/your_key.pem" admin@{public IPv4}
```

Once you are in the server, finally type
```
curl ifconfig.me
```

Make a note of the returned IP address. This is what we will point our DNS records to in the next step.
![5](/public/images/doc/ec2/ec2-5.png)

---
## Dependencies

DeploySolo requires golang. Here are the commands to install them, in addition to git.

```sh
# Update package lists and upgrade installed packages
sudo apt update && sudo apt upgrade -y

# Install Git and Go
sudo apt install git golang -y
```

Reboot the server for permissions to take effect.

## Clone Forked Repo
Now that you've set up a server, you need to get the project source code onto the server.

```sh
git clone github.com/your/repo
```

Congrats! You now have DeploySolo on an EC2 instance.
