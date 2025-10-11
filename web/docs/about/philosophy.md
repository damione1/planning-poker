# ☯️ Philosophy
## TLDR
DeploySolo is a SaaS starter kit that can be easily extended to develop and deploy your monetized web application.

Go, htmx, and PocketBase are incredibly powerful. But they can be difficult to bring together in a complete package.

DeploySolo combines these tools in a way that is easy for anyone to get started.

You can build the app you're dreaming of today, but learn these in-demand software skills at your own pace.

## Motivation
Web development is in a strange place. Instead of becoming easier to build web apps over time, there is often more complexity to deliver business logic to end users.

### Front End Frameworks
- **Front End Complexity**: Designed to bring richer experiences to the web with finer grained control, they bring as many "things to learn" as Back End.
- **Feature Creep**: Create new features labeled as upgrades, but also create an endless stream of work for developers to learn and upgrade deprecated code.
- **Dependencies**: Depend heavily on an ecosystem of npm dependencies.
- **End Product**: Serve a definite use case, but often make it more difficult to build the majority of web apps that could be built with [htmx](https://htmx.org/) instead.

### Back End
- **Infra Complexity**: New concepts such as Kubernetes, Terraform, Docker, EKS, Serverless, GraphQL, AWS, Heroku...
- **External Services**: It is common practice to offload tasks onto microservices, such as Auth, Database, or Doc Search. This increases latency from the network hop, increases monthly costs, and revokes full control over one's app.
- **PaaS**: Manage the components introduced by **Infra Complexity** and **External Services** with a Web UI that's simple to use. This makes it possible for developers to deploy apps, but lock them into monthly billing cycles for infrastructure they couldn't replicate on prem if needed.
- **JS/TS**: Since JS/TS must be learned for the front end, developers often reuse their knowledge and write the server with TS/JS.
- **End Result**: The Back End often ends up a TS server connecting the front end with other services using JSON APIs, doing little business logic on the server.

### Full Stack Frameworks
- **Server Utilization**: Full Stack Frameworks include batteries that solve common web app requirements such as Auth or Database in the server itself. This removes the need to offload tasks onto external services.
- **Saving Time / Shipping**: Full Stack Frameworks enable indie devs and time conscious devs to ship a stable product more quickly.
- **Languages**: Often written in scripting languages like Python or Ruby, which have intrinsic performance limitations.
- **Opinionated**: Require a set of framework specific conventions, which reduce decision fatigue, but aren't widely applicable.

### Reminder
The goal is to use the best available software to build and deploy web apps, while writing minimal boilerplate, and maintaining full project autonomy.

### DeploySolo
<b><u>DeploySolo is a Go SaaS Starter Kit with the above taken into account.</u></b>

- **Batteries**: DeploySolo implements the most common use cases for your web app including:
    - Stripe Integration and Paid Guards
    - Database with ORM
    - Email Auth
    - OAuth (Log In with Google)
    - Template Rendering
    - Live Documentation Search
    - SQLite Mode with Live Reload
- **Samples**: Pre-made Go+htmx reference implementations:
    - AI Chatbot
    - CRUD Tasks
    - Invite to GitHub
    - File Hosting
- **Educational Resources**: Documentation from a dev, for devs. I'll walk you from zero to a production web app, ready to be built on.
- **Autonomy**: The ability to take your code and run it on any server. Although a VPS may be more convenient, you can run it on your hardware easily.
- **Simplicity**: Computer programming will always be difficult, but choosing the right tools can make it easier without compromising on performance.
- **Performance**: Simplicity is no longer an excuse for slow code. High performance languages that are easy to use exist.
- **Scalability**: In an ideal world where your app gets millions of users, the architecture should be capable of scaling horizontally without major refactor, but still have a "one command quickstart". 
- **UNIX Philosophy**: DeploySolo is mostly built with the Go stdlib. It is a broadly applicable and in demand skill because it follows the UNIX Philosophy of doing one thing and doing it well. That is, being a simple and productive language that produces high performance and scalable code.
