# 👈 Hypermedia
A large portion of the web dev industry has moved to front end frameworks for finer grained control over the browser, creating reactive apps. But also introduce lots of complexity and security considerations.

DeploySolo circumvents this by integrating [htmx](https://htmx.org/). htmx takes a different approach. Instead of shipping a JS Client to the browser, "htmx completes HTML as a hypertext". By extending html and selectively replacing HTTP responses, we can achieve finer grained control over the browser.

DeploySolo provides several sample implementations which comprehensively outline how to develop an htmx application, which includes requesting and modifying data in a database.

For an immediate example, notice the navigation of this side doesn't force a full page reload. This is due to htmx using hx-boost for navigation.

For another example, try out the documentation search feature of this site. The live search results is simply a list of results formatted as HTML sent over the wire and injected into a Bootstrap content div.
