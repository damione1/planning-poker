# ⚡Live Reload
[air-verse/air](https://github.com/air-verse/air) makes it simple to have changes in templates or source code automatically trigger a rebuild. This is extremely helpful for development, as changes appear in the browser automatically without an extra step to rebuild.

(with go 1.22 or higher)
```sh
go install github.com/air-verse/air@latest
```

## Running
Simply type `air` at the command line, with additional parameters.

```sh
matt@matt-mbp:~/Development/Projects/deploysolo
$ air -- serve --http 192.168.1.111:8090

  __    _   ___
 / /\  | | | |_)
/_/--\ |_| |_| \_ v1.52.2, built with Go go1.22.2

watching .
... watching source files ...
building...
running...
Listening on :8090
```

Now, when changes are detected in templates or Go source files, Air will rebuild the code, instantly applying in the browser after a refresh.
