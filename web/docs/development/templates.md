# 🪟 Templates
Templates allow the rendering of views and the reuse of HTML fragments with injected data.

They can be compared to Components in a Front End Framework.

DeploySolo implements a custom template management scheme which is important to understand.

Consider the `templates` directory.
```sh
tree -L 1 web/templates/
web/templates/
├── app
├── auth
├── components
└── pages
```

## Loading
Templates are loaded into a single instance of `var tmpl *template.Template`, which is accessible in the pocketbase store.

## Views
Views define content inside pages. See `views/info/index.html`

```html
<!DOCTYPE html>
<html lang="en" class="dark">

<head>
  <title>Home | Project</title>
  {{ template "head" . }}
</head>

<body hx-boost="true">

    {{ template "navbar" . }}

    <div class="container">
        <!-- Page Content -->
    </div>

    {{ template "searchmodal" . }}
    {{ template "footer" . }}

</body>

{{ end }}
```

Pages are complete html files, which allow the injection of reused HTML fragments defined in `*.tmpl` files, and passing in runtime Go data with the `html/template` `ExecuteTemplate` function.

## Partials
Consider a fragment from `views/app/tasks.html`:
```html
<div id="tasks">
    {{ range $task := .Context.Data }}
    {{ template "task" $task }}
    {{ end }}
</div>
```

Inside, `partials/components/tasks.tmpl`, we define the template "task" with:
```html
{{ define "task" }}
<div id="target-{{ .ID }}">
  <div class="input-group py-2">
  	<input type="search" id="search-dropdown" disabled class="form-control" placeholder="{{ .Task }}" required />
  	<button hx-get="/app/tasks/{{ .ID }}" hx-target="#target-{{ .ID }}" type="submit" class="btn btn-primary">
  		Edit
  	</button>
  </div>
</div>
{{ end }}
```

Notice how all *.tmpl files are accessible from every view. The server provides an array of `Tasks`, and using `html/template` syntax, we can create a list of tasks with the predefined template. 
