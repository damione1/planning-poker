package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/mannders00/deploysolo/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

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

type Task struct {
	ID   int    `json:"id"`
	Task string `json:"task"`
	Done bool   `json:"done"`
}

func renderTasks(e *core.RequestEvent) error {
	// Fetch user's tasks from database, which is stored as the following JSON array.
	// [
	//   {
	//     "id": 1,
	//     "task": "task one",
	//     "done": false
	//   },
	//   {
	//   {
	//     "id": 2,
	//     "task": "task two",
	//     "done": true
	//   }
	// ]
	tasks := e.Auth.Get("tasks").(types.JSONRaw)

	// Unmarshal JSON string into []Task
	var taskList []Task
	if len(tasks) != 0 {
		err := json.Unmarshal(tasks, &taskList)
		if err != nil {
			panic(err)
		}
	} else {
		taskList = make([]Task, 0)
	}

	return utils.RenderView("tasks.html", taskList)(e)
}

func createTask(e *core.RequestEvent) error {
	// // Get user task list, which is a JSON array
	taskJson := e.Auth.Get("tasks").(types.JSONRaw)
	var taskArr []Task
	if taskJson == nil {
		taskArr = make([]Task, 0)
	} else {
		json.Unmarshal(taskJson, &taskArr)
	}

	// Find max task ID, to make a unique one
	maxID := 0
	for _, v := range taskArr {
		if maxID < v.ID {
			maxID = v.ID
		}
	}

	// Create task instance from user POST input
	taskInput := e.Request.FormValue("newtask")
	taskCreate := Task{
		Task: taskInput,
		Done: false,
		ID:   maxID + 1,
	}

	// // Prepend new task struct instance to previous task list
	updatedTasksArr := append([]Task{taskCreate}, taskArr...)
	updatedTasksBytes, err := json.Marshal(updatedTasksArr)
	if err != nil {
		return err
	}

	// Save to PocketBase DB
	e.Auth.Set("tasks", types.JSONRaw(updatedTasksBytes))
	if err := e.App.Save(e.Auth); err != nil {
		return err
	}

	tmpl := e.App.Store().Get("tmpl").(*template.Template)
	return tmpl.ExecuteTemplate(e.Response, "task", taskCreate)
}

func editTask(e *core.RequestEvent) error {
	// Get user input
	id := e.Request.PathValue("id")
	idx, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	// Get user task list, which is a JSON array
	taskJson := e.Auth.Get("tasks").(types.JSONRaw)
	var taskArr []Task
	if taskJson == nil {
		taskArr = make([]Task, 0)
	} else {
		json.Unmarshal(taskJson, &taskArr)
	}

	// Find current task from array
	var task Task
	for _, v := range taskArr {
		if v.ID == idx {
			task = v
		}
	}

	tmpl := e.App.Store().Get("tmpl").(*template.Template)
	return tmpl.ExecuteTemplate(e.Response, "edittask", task)
}

func saveTask(e *core.RequestEvent) error {
	// Get user input
	id := e.Request.PathValue("id")
	idx, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	taskInput := e.Request.FormValue("savetask")

	// Get user task list, which is a JSON array
	taskJson := e.Auth.Get("tasks").(types.JSONRaw)
	var taskArr []Task
	if taskJson == nil {
		taskArr = make([]Task, 0)
	} else {
		json.Unmarshal(taskJson, &taskArr)
	}

	// Find current task from array, replace values
	var savedTask Task
	for k, v := range taskArr {
		if v.ID == idx {
			savedTask = Task{
				Task: taskInput,
				Done: v.Done,
				ID:   v.ID,
			}
			taskArr[k] = savedTask
		}
	}

	// Convert task back to JSON string
	tasksConvBytes, err := json.Marshal(taskArr)
	if err != nil {
		return err
	}

	// Save task
	e.Auth.Set("tasks", types.JSONRaw(tasksConvBytes))
	if err := e.App.Save(e.Auth); err != nil {
		return err
	}

	tmpl := e.App.Store().Get("tmpl").(*template.Template)
	return tmpl.ExecuteTemplate(e.Response, "task", savedTask)
}

func deleteTask(e *core.RequestEvent) error {
	// Get user input
	id := e.Request.PathValue("id")
	idx, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	// Get user task list, which is a JSON array
	taskJson := e.Auth.Get("tasks").(types.JSONRaw)
	var taskArr []Task
	if taskJson == nil {
		taskArr = make([]Task, 0)
	} else {
		json.Unmarshal(taskJson, &taskArr)
	}

	// Find task to delete
	for k, v := range taskArr {
		if v.ID == idx {
			taskArr = append(taskArr[:k], taskArr[k+1:]...)
		}
	}

	// Convert task back to JSON string
	tasksConvBytes, err := json.Marshal(taskArr)
	if err != nil {
		return err
	}

	// Save task
	e.Auth.Set("tasks", types.JSONRaw(tasksConvBytes))
	if err := e.App.Save(e.Auth); err != nil {
		return err
	}

	return e.String(http.StatusOK, "")
}
