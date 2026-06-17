package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

type client struct {
	base  string
	token string
	http  *http.Client
}

type responseMap map[string]any

func main() {
	base := strings.TrimRight(os.Getenv("SMOKE_API_BASE"), "/")
	if base == "" {
		base = "http://127.0.0.1:8080/api/v1"
	}
	c := &client{base: base, http: &http.Client{Timeout: 20 * time.Second}}
	stamp := time.Now().UTC().Format("20060102150405")
	email := fmt.Sprintf("smoke-%s@meta-org.local", stamp)
	password := "SmokePass123!"

	user := c.post("/auth/register", responseMap{
		"name":     "Smoke User " + stamp,
		"email":    email,
		"password": password,
	})
	login := c.post("/auth/login", responseMap{"email": email, "password": password})
	c.token = stringField(login, "token")
	userID := stringField(user, "id")
	if userID == "" {
		userID = stringField(login, "user_id")
	}
	must(userID != "", "missing user id")

	org := c.post("/organizations", responseMap{
		"name":        "Smoke Organization " + stamp,
		"description": "PDCA smoke organization",
	})
	orgID := stringField(org, "id")
	dept := c.post("/organizations/"+orgID+"/departments", responseMap{
		"name":        "Delivery",
		"code":        "DEL",
		"description": "Delivery department",
	})
	deptID := stringField(dept, "id")
	template := c.post("/workflows/templates", responseMap{
		"name":            "Smoke PDCA Workflow " + stamp,
		"description":     "Plan, execute, review",
		"organization_id": orgID,
		"department_id":   deptID,
		"stages": []responseMap{
			{"type": "plan", "name": "Plan", "assignee_type": "internal", "required_permission_level": "L1", "risk_level": "low"},
			{"type": "execute", "name": "Do", "assignee_type": "either", "required_permission_level": "L2", "risk_level": "medium"},
			{"type": "review", "name": "Accept", "assignee_type": "internal", "required_permission_level": "L2", "risk_level": "medium"},
		},
	})
	templateID := stringField(template, "id")

	requirement := c.post("/requirements", responseMap{
		"title":           "Smoke PDCA Requirement " + stamp,
		"description":     "Verify requirement, project, workflow, delivery, cost, feedback, and PDCA evidence.",
		"priority":        "medium",
		"risk_level":      "medium",
		"required_level":  "L2",
		"budget_amount":   5000,
		"budget_currency": "CNY",
		"organization_id": orgID,
		"department_id":   deptID,
		"created_by_id":   userID,
		"created_by_type": "internal_human",
	})
	requirementID := stringField(requirement, "id")
	c.upload("/requirements/"+requirementID+"/documents", "smoke-requirement.txt", "text/plain", []byte("smoke requirement evidence"))
	requirement = c.post("/requirements/"+requirementID+"/analyze", responseMap{"notes": "smoke analysis"})
	must(stringField(requirement, "status") == "analyzed", "requirement was not analyzed")
	requirement = c.post("/requirements/"+requirementID+"/approve", responseMap{})
	must(stringField(requirement, "status") == "approved", "requirement was not approved")
	project := c.post("/requirements/"+requirementID+"/convert-to-project", responseMap{})
	projectID := stringField(project, "id")

	c.post("/projects/"+projectID+"/members", responseMap{
		"member_actor_id":    userID,
		"member_actor_type":  "internal_human",
		"role":               "owner",
		"title":              "Smoke owner",
		"allocation_percent": 100,
		"cost_rate":          800,
		"permission_level":   "L2",
		"capabilities":       []string{"planning", "delivery", "review"},
		"actor_id":           userID,
		"actor_type":         "internal_human",
	})
	projectWorkflow := c.post("/projects/"+projectID+"/workflows", responseMap{
		"workflow_template_id": templateID,
		"purpose":              "delivery",
		"actor_id":             userID,
		"actor_type":           "internal_human",
	})
	workflowID := stringField(projectWorkflow, "workflow_id")
	workflow := c.get("/workflows/instances/" + workflowID)
	for _, rawTask := range listField(workflow, "tasks") {
		task, ok := rawTask.(map[string]any)
		must(ok, "invalid workflow task")
		c.patch("/tasks/"+stringField(task, "id")+"/status", responseMap{
			"output": responseMap{"smoke": true, "stage": numberField(task, "stage")},
		})
	}

	project = c.post("/projects/"+projectID+"/status", responseMap{"status": "active", "actor_id": userID, "actor_type": "internal_human"})
	must(stringField(project, "status") == "active", "project was not activated")
	deliverable := c.post("/projects/"+projectID+"/deliverables", responseMap{
		"name":             "Smoke Deliverable",
		"deliverable_type": "document",
		"version":          "1.0",
		"status":           "draft",
		"actor_id":         userID,
		"actor_type":       "internal_human",
	})
	deliverableID := stringField(deliverable, "id")
	c.post("/deliverables/"+deliverableID+"/submit", responseMap{"actor_id": userID, "actor_type": "internal_human"})
	c.post("/deliverables/"+deliverableID+"/accept", responseMap{"actor_id": userID, "actor_type": "internal_human"})
	c.post("/projects/"+projectID+"/cost-refresh", responseMap{"actor_id": userID, "actor_type": "internal_human"})
	c.post("/projects/"+projectID+"/cost-entries", responseMap{
		"source_type":      "manual",
		"entry_actor_id":   userID,
		"entry_actor_type": "internal_human",
		"amount":           120,
		"currency":         "CNY",
		"description":      "smoke manual cost",
		"actor_id":         userID,
		"actor_type":       "internal_human",
	})
	project = c.post("/projects/"+projectID+"/status", responseMap{"status": "delivering", "actor_id": userID, "actor_type": "internal_human"})
	must(stringField(project, "status") == "delivering", "project was not delivering")
	project = c.post("/projects/"+projectID+"/status", responseMap{"status": "completed", "actor_id": userID, "actor_type": "internal_human"})
	must(stringField(project, "status") == "completed", "project was not completed")
	c.post("/projects/"+projectID+"/evaluations", responseMap{
		"evaluated_actor_id":   userID,
		"evaluated_actor_type": "internal_human",
		"quality_score":        0.9,
		"delivery_score":       0.85,
		"cost_score":           0.8,
		"collaboration_score":  0.9,
		"conclusion":           "smoke evaluation passed",
		"actor_id":             userID,
		"actor_type":           "internal_human",
	})
	closeResult := c.post("/projects/"+projectID+"/close-feedback", responseMap{
		"outcome_score": 0.88,
		"conclusion":    "smoke feedback closed",
		"actor_id":      userID,
		"actor_type":    "internal_human",
	})
	closedProject, _ := closeResult["project"].(map[string]any)
	must(stringField(closedProject, "status") == "closed", "project was not closed")

	overview := c.get("/projects/" + projectID + "/overview")
	lifecycle, _ := overview["lifecycle"].(map[string]any)
	cycleID := stringField(lifecycle, "pdca_cycle_id")
	must(cycleID != "", "missing pdca cycle id")
	events := c.get("/pdca-events?cycle_id=" + cycleID)
	must(len(asList(events["items"])) > 0, "missing pdca events")
	fmt.Printf("smoke ok: requirement=%s project=%s pdca_cycle=%s\n", requirementID, projectID, cycleID)
}

func (c *client) get(path string) responseMap {
	return c.do(http.MethodGet, path, nil, http.StatusOK)
}

func (c *client) post(path string, payload responseMap) responseMap {
	status := http.StatusOK
	if path == "/auth/register" ||
		path == "/organizations" ||
		path == "/workflows/templates" ||
		path == "/requirements" ||
		(strings.HasPrefix(path, "/organizations/") && strings.HasSuffix(path, "/departments")) ||
		strings.HasSuffix(path, "/convert-to-project") ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/members")) ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/workflows")) ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/deliverables")) ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/cost-entries")) ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/cost-refresh")) ||
		(strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/evaluations")) {
		status = http.StatusCreated
	}
	return c.do(http.MethodPost, path, payload, status)
}

func (c *client) patch(path string, payload responseMap) responseMap {
	return c.do(http.MethodPatch, path, payload, http.StatusOK)
}

func (c *client) do(method string, path string, payload responseMap, expected int) responseMap {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			panic(err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		panic(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expected {
		panic(fmt.Sprintf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, expected, string(data)))
	}
	var result responseMap
	if len(data) == 0 {
		return responseMap{}
	}
	if err := json.Unmarshal(data, &result); err != nil {
		var items []any
		if listErr := json.Unmarshal(data, &items); listErr == nil {
			return responseMap{"items": items}
		}
		panic(fmt.Sprintf("%s %s: decode response: %v: %s", method, path, err, string(data)))
	}
	return result
}

func (c *client) upload(path string, fileName string, contentType string, content []byte) responseMap {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		panic(err)
	}
	if _, err := part.Write(content); err != nil {
		panic(err)
	}
	_ = writer.WriteField("metadata", `{"source":"smoke"}`)
	if err := writer.Close(); err != nil {
		panic(err)
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, &body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if contentType != "" {
		req.Header.Set("X-Smoke-Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		panic(fmt.Sprintf("upload %s: got %d: %s", path, resp.StatusCode, string(data)))
	}
	var result responseMap
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	return result
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func numberField(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	value, _ := m[key].(float64)
	return value
}

func listField(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	return asList(m[key])
}

func asList(value any) []any {
	list, _ := value.([]any)
	return list
}

func must(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
