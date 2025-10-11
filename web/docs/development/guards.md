# 📵 Guards
Guards allow or deny access to particular HTTP routes based on some condition.

## RequirePayment
`RequirePayment` redirects user to the login page if not authenticated, and to the purchase page if not paid.

It is useful for creating pages that are only accessible to users if they have paid for your product.

```go
func RegisterTasks(se *core.ServeEvent) error {
	g := se.Router.Group("/app")
	g.Bind(utils.RequirePayment())

	g.GET("/tasks", renderTasks)
	g.POST("/tasks", createTask)
	g.GET("/tasks/{id}", editTask)
	g.PUT("/tasks/{id}", saveTask)
	g.DELETE("/tasks/{id}", deleteTask)

	return nil
}
```

## RequireAuth
`RequireAuth` redirects user to the login page if not authenticated.

```go
se.Router.GET("/checkout", checkoutHandler).Bind(apis.RequireAuth())
```
