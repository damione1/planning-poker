package handlers

import (
	"github.com/pocketbase/pocketbase/core"
)

func Home(re *core.RequestEvent) error {
	// Temporary placeholder until templ templates are ready
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Planning Poker</title>
    <style>
        body { font-family: sans-serif; max-width: 500px; margin: 50px auto; padding: 20px; }
        input { width: 100%; padding: 10px; margin: 10px 0; }
        button { padding: 10px 20px; background: #667eea; color: white; border: none; cursor: pointer; }
    </style>
</head>
<body>
    <h1>Planning Poker</h1>
    <p>Create a room to start estimating</p>
    <form method="POST" action="/room">
        <input type="text" name="name" placeholder="Room Name" required />
        <label><input type="radio" name="pointingMethod" value="fibonacci" checked /> Fibonacci</label>
        <label><input type="radio" name="pointingMethod" value="custom" /> Custom</label>
        <button type="submit">Create Room</button>
    </form>
</body>
</html>`
	return re.HTML(200, html)
}
