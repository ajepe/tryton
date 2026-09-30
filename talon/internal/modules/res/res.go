package res

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("res", &metadata.Module{
		Name:    "res",
		Version: "0.1.0",
		Depends: []string{"ir"},
	})

	metadata.RegisterModel("res.user", &metadata.Model{
		Name:  "res.user",
		Table: "res_user",
		Fields: map[string]*metadata.Field{
			"login":       {Name: "login", Type: metadata.FieldChar, Label: "Login", Required: true},
			"name":        {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"password":      {Name: "password", Type: metadata.FieldChar, Label: "Password"},
			"password_hash": {Name: "password_hash", Type: metadata.FieldChar, Label: "Password Hash"},
			"email":       {Name: "email", Type: metadata.FieldChar, Label: "Email"},
			"active":      {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
			"administrator": {Name: "administrator", Type: metadata.FieldBoolean, Label: "Administrator"},
			"groups":      {Name: "groups", Type: metadata.FieldMany2Many, Label: "Groups", Relation: "res.group"},
			"access":      {Name: "access", Type: metadata.FieldOne2Many, Label: "Access", Relation: "res.user.access"},
			"preferences": {Name: "preferences", Type: metadata.FieldDict, Label: "Preferences"},
		},
	})

	metadata.RegisterModel("res.group", &metadata.Model{
		Name:  "res.group",
		Table: "res_group",
		Fields: map[string]*metadata.Field{
			"name":  {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"users": {Name: "users", Type: metadata.FieldMany2Many, Label: "Users", Relation: "res.user"},
			"model_access": {Name: "model_access", Type: metadata.FieldOne2Many, Label: "Model Access", Relation: "ir.model.access"},
			"field_access": {Name: "field_access", Type: metadata.FieldOne2Many, Label: "Field Access", Relation: "ir.model.field.access"},
		},
	})

	metadata.RegisterModel("res.user.access", &metadata.Model{
		Name:  "res.user.access",
		Table: "res_user_access",
		Fields: map[string]*metadata.Field{
			"user":  {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user", Required: true},
			"group": {Name: "group", Type: metadata.FieldMany2One, Label: "Group", Relation: "res.group", Required: true},
		},
	})

	metadata.RegisterModel("res.user.application", &metadata.Model{
		Name:  "res.user.application",
		Table: "res_user_application",
		Fields: map[string]*metadata.Field{
			"user":        {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"key":         {Name: "key", Type: metadata.FieldChar, Label: "Key"},
			"application": {Name: "application", Type: metadata.FieldChar, Label: "Application"},
			"state":       {Name: "state", Type: metadata.FieldSelection, Label: "State"},
		},
	})

	metadata.RegisterModel("res.user.warning", &metadata.Model{
		Name:  "res.user.warning",
		Table: "res_user_warning",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"user":   {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"always": {Name: "always", Type: metadata.FieldBoolean, Label: "Always"},
		},
	})

	metadata.RegisterModel("res.user.device", &metadata.Model{
		Name:  "res.user.device",
		Table: "res_user_device",
		Fields: map[string]*metadata.Field{
			"user":   {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"device": {Name: "device", Type: metadata.FieldChar, Label: "Device"},
			"token":  {Name: "token", Type: metadata.FieldChar, Label: "Token"},
		},
	})

	metadata.RegisterModel("res.notification", &metadata.Model{
		Name:  "res.notification",
		Table: "res_notification",
		Fields: map[string]*metadata.Field{
			"user":    {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"title":   {Name: "title", Type: metadata.FieldChar, Label: "Title"},
			"body":    {Name: "body", Type: metadata.FieldText, Label: "Body"},
			"priority": {Name: "priority", Type: metadata.FieldSelection, Label: "Priority"},
			"read":    {Name: "read", Type: metadata.FieldBoolean, Label: "Read"},
		},
	})
}
