package ir

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("ir", &metadata.Module{
		Name:    "ir",
		Version: "0.1.0",
	})

	metadata.RegisterModel("ir.model", &metadata.Model{
		Name:  "ir.model",
		Table: "ir_model",
		Fields: map[string]*metadata.Field{
			"name":        {Name: "name", Type: metadata.FieldChar, Label: "Model", Required: true},
			"model":       {Name: "model", Type: metadata.FieldChar, Label: "Model Name", Required: true},
			"module":      {Name: "module", Type: metadata.FieldChar, Label: "Module"},
			"global_name": {Name: "global_name", Type: metadata.FieldChar, Label: "Global Name"},
		},
	})

	metadata.RegisterModel("ir.model.field", &metadata.Model{
		Name:  "ir.model.field",
		Table: "ir_model_field",
		Fields: map[string]*metadata.Field{
			"name":        {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"model":       {Name: "model", Type: metadata.FieldMany2One, Label: "Model", Relation: "ir.model", Required: true},
			"field_description": {Name: "field_description", Type: metadata.FieldChar, Label: "Field Description"},
			"ttype":       {Name: "ttype", Type: metadata.FieldChar, Label: "Field Type"},
			"relation":    {Name: "relation", Type: metadata.FieldChar, Label: "Relation"},
			"module":      {Name: "module", Type: metadata.FieldChar, Label: "Module"},
		},
	})

	metadata.RegisterModel("ir.model.access", &metadata.Model{
		Name:  "ir.model.access",
		Table: "ir_model_access",
		Fields: map[string]*metadata.Field{
			"model":       {Name: "model", Type: metadata.FieldMany2One, Label: "Model", Relation: "ir.model", Required: true},
			"group":       {Name: "group", Type: metadata.FieldMany2One, Label: "Group", Relation: "res.group"},
			"perm_read":   {Name: "perm_read", Type: metadata.FieldBoolean, Label: "Read Access"},
			"perm_write":  {Name: "perm_write", Type: metadata.FieldBoolean, Label: "Write Access"},
			"perm_create": {Name: "perm_create", Type: metadata.FieldBoolean, Label: "Create Access"},
			"perm_delete": {Name: "perm_delete", Type: metadata.FieldBoolean, Label: "Delete Access"},
		},
	})

	metadata.RegisterModel("ir.model.button", &metadata.Model{
		Name:  "ir.model.button",
		Table: "ir_model_button",
		Fields: map[string]*metadata.Field{
			"name":      {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"model":     {Name: "model", Type: metadata.FieldMany2One, Label: "Model", Relation: "ir.model", Required: true},
			"module":    {Name: "module", Type: metadata.FieldChar, Label: "Module"},
		},
	})

	metadata.RegisterModel("ir.model.button.click", &metadata.Model{
		Name:  "ir.model.button.click",
		Table: "ir_model_button_click",
		Fields: map[string]*metadata.Field{
			"button":   {Name: "button", Type: metadata.FieldMany2One, Label: "Button", Relation: "ir.model.button"},
			"record":   {Name: "record", Type: metadata.FieldInteger, Label: "Record"},
			"user":     {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
		},
	})

	metadata.RegisterModel("ir.model.field.access", &metadata.Model{
		Name:  "ir.model.field.access",
		Table: "ir_model_field_access",
		Fields: map[string]*metadata.Field{
			"field":       {Name: "field", Type: metadata.FieldMany2One, Label: "Field", Relation: "ir.model.field"},
			"group":       {Name: "group", Type: metadata.FieldMany2One, Label: "Group", Relation: "res.group"},
			"perm_read":   {Name: "perm_read", Type: metadata.FieldBoolean, Label: "Read Access"},
			"perm_write":  {Name: "perm_write", Type: metadata.FieldBoolean, Label: "Write Access"},
			"perm_create": {Name: "perm_create", Type: metadata.FieldBoolean, Label: "Create Access"},
			"perm_delete": {Name: "perm_delete", Type: metadata.FieldBoolean, Label: "Delete Access"},
		},
	})

	metadata.RegisterModel("ir.model.data", &metadata.Model{
		Name:  "ir.model.data",
		Table: "ir_model_data",
		Fields: map[string]*metadata.Field{
			"module":   {Name: "module", Type: metadata.FieldChar, Label: "Module", Required: true},
			"model":    {Name: "model", Type: metadata.FieldChar, Label: "Model", Required: true},
			"fs_id":    {Name: "fs_id", Type: metadata.FieldChar, Label: "FS ID", Required: true},
			"db_id":    {Name: "db_id", Type: metadata.FieldInteger, Label: "DB ID"},
			"noupdate": {Name: "noupdate", Type: metadata.FieldBoolean, Label: "No Update"},
		},
	})

	metadata.RegisterModel("ir.model.log", &metadata.Model{
		Name:  "ir.model.log",
		Table: "ir_model_log",
		Fields: map[string]*metadata.Field{
			"model":    {Name: "model", Type: metadata.FieldMany2One, Label: "Model", Relation: "ir.model"},
			"field":    {Name: "field", Type: metadata.FieldChar, Label: "Field"},
			"user":     {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"create_date": {Name: "create_date", Type: metadata.FieldDateTime, Label: "Create Date"},
			"write_date":  {Name: "write_date", Type: metadata.FieldDateTime, Label: "Write Date"},
		},
	})

	metadata.RegisterModel("ir.ui.view", &metadata.Model{
		Name:  "ir.ui.view",
		Table: "ir_ui_view",
		Fields: map[string]*metadata.Field{
			"name":     {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"model":    {Name: "model", Type: metadata.FieldChar, Label: "Model", Required: true},
			"type":     {Name: "type", Type: metadata.FieldSelection, Label: "View Type"},
			"arch":     {Name: "arch", Type: metadata.FieldText, Label: "Architecture"},
			"priority": {Name: "priority", Type: metadata.FieldInteger, Label: "Priority"},
			"module":   {Name: "module", Type: metadata.FieldChar, Label: "Module"},
			"inherit":  {Name: "inherit", Type: metadata.FieldMany2One, Label: "Inherited View", Relation: "ir.ui.view"},
		},
	})

	metadata.RegisterModel("ir.ui.view_tree_width", &metadata.Model{
		Name:  "ir.ui.view_tree_width",
		Table: "ir_ui_view_tree_width",
		Fields: map[string]*metadata.Field{
			"model":  {Name: "model", Type: metadata.FieldChar, Label: "Model"},
			"field":  {Name: "field", Type: metadata.FieldChar, Label: "Field"},
			"user":   {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"width":  {Name: "width", Type: metadata.FieldInteger, Label: "Width"},
		},
	})

	metadata.RegisterModel("ir.ui.view_tree_state", &metadata.Model{
		Name:  "ir.ui.view_tree_state",
		Table: "ir_ui_view_tree_state",
		Fields: map[string]*metadata.Field{
			"model":  {Name: "model", Type: metadata.FieldChar, Label: "Model"},
			"field":  {Name: "field", Type: metadata.FieldChar, Label: "Field"},
			"user":   {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"expand": {Name: "expand", Type: metadata.FieldText, Label: "Expand"},
		},
	})

	metadata.RegisterModel("ir.ui.view_tree_optional", &metadata.Model{
		Name:  "ir.ui.view_tree_optional",
		Table: "ir_ui_view_tree_optional",
		Fields: map[string]*metadata.Field{
			"model":    {Name: "model", Type: metadata.FieldChar, Label: "Model"},
			"field":    {Name: "field", Type: metadata.FieldChar, Label: "Field"},
			"user":     {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"optional": {Name: "optional", Type: metadata.FieldBoolean, Label: "Optional"},
		},
	})

	metadata.RegisterModel("ir.ui.menu", &metadata.Model{
		Name:  "ir.ui.menu",
		Table: "ir_ui_menu",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"parent":     {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "ir.ui.menu"},
			"action":     {Name: "action", Type: metadata.FieldReference, Label: "Action"},
			"action_keywords": {Name: "action_keywords", Type: metadata.FieldOne2Many, Label: "Keywords", Relation: "ir.ui.menu.favorite"},
			"favorite":   {Name: "favorite", Type: metadata.FieldBoolean, Label: "Favorite"},
			"module":     {Name: "module", Type: metadata.FieldChar, Label: "Module"},
		},
	})

	metadata.RegisterModel("ir.ui.menu.favorite", &metadata.Model{
		Name:  "ir.ui.menu.favorite",
		Table: "ir_ui_menu_favorite",
		Fields: map[string]*metadata.Field{
			"menu":  {Name: "menu", Type: metadata.FieldMany2One, Label: "Menu", Relation: "ir.ui.menu"},
			"user":  {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"sequence": {Name: "sequence", Type: metadata.FieldInteger, Label: "Sequence"},
		},
	})

	metadata.RegisterModel("ir.ui.icon", &metadata.Model{
		Name:  "ir.ui.icon",
		Table: "ir_ui_icon",
		Fields: map[string]*metadata.Field{
			"name":  {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"module": {Name: "module", Type: metadata.FieldChar, Label: "Module"},
			"icon":  {Name: "icon", Type: metadata.FieldBinary, Label: "Icon"},
		},
	})

	metadata.RegisterModel("ir.action", &metadata.Model{
		Name:  "ir.action",
		Table: "ir_action",
		Fields: map[string]*metadata.Field{
			"name":    {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"type":    {Name: "type", Type: metadata.FieldSelection, Label: "Action Type", Required: true},
			"usage":   {Name: "usage", Type: metadata.FieldChar, Label: "Usage"},
			"module":  {Name: "module", Type: metadata.FieldChar, Label: "Module"},
		},
	})

	metadata.RegisterModel("ir.action.act_window", &metadata.Model{
		Name:  "ir.action.act_window",
		Table: "ir_action_act_window",
		Fields: map[string]*metadata.Field{
			"action":      {Name: "action", Type: metadata.FieldMany2One, Label: "Action", Relation: "ir.action", Required: true},
			"res_model":   {Name: "res_model", Type: metadata.FieldChar, Label: "Model"},
			"view_mode":   {Name: "view_mode", Type: metadata.FieldChar, Label: "View Mode"},
			"domain":      {Name: "domain", Type: metadata.FieldChar, Label: "Domain"},
			"context":     {Name: "context", Type: metadata.FieldChar, Label: "Context"},
			"order":       {Name: "order", Type: metadata.FieldChar, Label: "Order"},
			"limit":       {Name: "limit", Type: metadata.FieldInteger, Label: "Limit"},
		},
	})

	metadata.RegisterModel("ir.action.wizard", &metadata.Model{
		Name:  "ir.action.wizard",
		Table: "ir_action_wizard",
		Fields: map[string]*metadata.Field{
			"action":    {Name: "action", Type: metadata.FieldMany2One, Label: "Action", Relation: "ir.action", Required: true},
			"wiz_name":  {Name: "wiz_name", Type: metadata.FieldChar, Label: "Wizard"},
			"window":    {Name: "window", Type: metadata.FieldBoolean, Label: "Window"},
		},
	})

	metadata.RegisterModel("ir.action.report", &metadata.Model{
		Name:  "ir.action.report",
		Table: "ir_action_report",
		Fields: map[string]*metadata.Field{
			"action":       {Name: "action", Type: metadata.FieldMany2One, Label: "Action", Relation: "ir.action", Required: true},
			"report_name":  {Name: "report_name", Type: metadata.FieldChar, Label: "Report"},
			"model":        {Name: "model", Type: metadata.FieldChar, Label: "Model"},
			"direct_print": {Name: "direct_print", Type: metadata.FieldBoolean, Label: "Direct Print"},
		},
	})

	metadata.RegisterModel("ir.action.url", &metadata.Model{
		Name:  "ir.action.url",
		Table: "ir_action_url",
		Fields: map[string]*metadata.Field{
			"action": {Name: "action", Type: metadata.FieldMany2One, Label: "Action", Relation: "ir.action", Required: true},
			"url":    {Name: "url", Type: metadata.FieldChar, Label: "URL"},
		},
	})

	metadata.RegisterModel("ir.action.keyword", &metadata.Model{
		Name:  "ir.action.keyword",
		Table: "ir_action_keyword",
		Fields: map[string]*metadata.Field{
			"action":  {Name: "action", Type: metadata.FieldMany2One, Label: "Action", Relation: "ir.action"},
			"keyword": {Name: "keyword", Type: metadata.FieldChar, Label: "Keyword"},
			"model":   {Name: "model", Type: metadata.FieldChar, Label: "Model"},
		},
	})

	metadata.RegisterModel("ir.sequence", &metadata.Model{
		Name:  "ir.sequence",
		Table: "ir_sequence",
		Fields: map[string]*metadata.Field{
			"name":         {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":         {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"module":       {Name: "module", Type: metadata.FieldChar, Label: "Module"},
			"prefix":       {Name: "prefix", Type: metadata.FieldChar, Label: "Prefix"},
			"suffix":       {Name: "suffix", Type: metadata.FieldChar, Label: "Suffix"},
			"number_next":  {Name: "number_next", Type: metadata.FieldInteger, Label: "Next Number"},
			"number_increment": {Name: "number_increment", Type: metadata.FieldInteger, Label: "Increment"},
			"padding":     {Name: "padding", Type: metadata.FieldInteger, Label: "Padding"},
		},
	})

	metadata.RegisterModel("ir.sequence.type", &metadata.Model{
		Name:  "ir.sequence.type",
		Table: "ir_sequence_type",
		Fields: map[string]*metadata.Field{
			"name": {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
		},
	})

	metadata.RegisterModel("ir.translation", &metadata.Model{
		Name:  "ir.translation",
		Table: "ir_translation",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"lang":       {Name: "lang", Type: metadata.FieldChar, Label: "Language"},
			"type":       {Name: "type", Type: metadata.FieldSelection, Label: "Type"},
			"src":        {Name: "src", Type: metadata.FieldText, Label: "Source"},
			"src_plural": {Name: "src_plural", Type: metadata.FieldText, Label: "Source Plural"},
			"value":      {Name: "value", Type: metadata.FieldText, Label: "Value"},
			"value_1":    {Name: "value_1", Type: metadata.FieldText, Label: "Value 1"},
			"value_2":    {Name: "value_2", Type: metadata.FieldText, Label: "Value 2"},
			"value_3":    {Name: "value_3", Type: metadata.FieldText, Label: "Value 3"},
			"module":     {Name: "module", Type: metadata.FieldChar, Label: "Module"},
			"res_id":     {Name: "res_id", Type: metadata.FieldInteger, Label: "Resource ID"},
			"fuzzy":      {Name: "fuzzy", Type: metadata.FieldBoolean, Label: "Fuzzy"},
		},
	})

	metadata.RegisterModel("ir.lang", &metadata.Model{
		Name:  "ir.lang",
		Table: "ir_lang",
		Fields: map[string]*metadata.Field{
			"code":      {Name: "code", Type: metadata.FieldChar, Label: "Code", Required: true},
			"name":      {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"direction": {Name: "direction", Type: metadata.FieldChar, Label: "Direction"},
			"date_format": {Name: "date_format", Type: metadata.FieldChar, Label: "Date Format"},
			"decimal_point": {Name: "decimal_point", Type: metadata.FieldChar, Label: "Decimal Point"},
			"thousands_sep": {Name: "thousands_sep", Type: metadata.FieldChar, Label: "Thousands Separator"},
		},
	})

	metadata.RegisterModel("ir.rule.group", &metadata.Model{
		Name:  "ir.rule.group",
		Table: "ir_rule_group",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"model":  {Name: "model", Type: metadata.FieldChar, Label: "Model", Required: true},
			"global_p": {Name: "global_p", Type: metadata.FieldBoolean, Label: "Global"},
			"default_p": {Name: "default_p", Type: metadata.FieldBoolean, Label: "Default"},
			"groups": {Name: "groups", Type: metadata.FieldMany2Many, Label: "Groups", Relation: "res.group"},
			"rules":  {Name: "rules", Type: metadata.FieldOne2Many, Label: "Tests", Relation: "ir.rule"},
			"perm_read":   {Name: "perm_read", Type: metadata.FieldBoolean, Label: "Read Access"},
			"perm_write":  {Name: "perm_write", Type: metadata.FieldBoolean, Label: "Write Access"},
			"perm_create": {Name: "perm_create", Type: metadata.FieldBoolean, Label: "Create Access"},
			"perm_delete": {Name: "perm_delete", Type: metadata.FieldBoolean, Label: "Delete Access"},
		},
	})

	metadata.RegisterModel("ir.rule", &metadata.Model{
		Name:  "ir.rule",
		Table: "ir_rule",
		Fields: map[string]*metadata.Field{
			"rule_group": {Name: "rule_group", Type: metadata.FieldMany2One, Label: "Rule Group", Relation: "ir.rule.group", Required: true},
			"domain":     {Name: "domain", Type: metadata.FieldText, Label: "Domain", Required: true},
		},
	})

	metadata.RegisterModel("ir.cron", &metadata.Model{
		Name:  "ir.cron",
		Table: "ir_cron",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"user":       {Name: "user", Type: metadata.FieldMany2One, Label: "User", Relation: "res.user"},
			"model":      {Name: "model", Type: metadata.FieldChar, Label: "Model"},
			"function":   {Name: "function", Type: metadata.FieldChar, Label: "Function"},
			"args":       {Name: "args", Type: metadata.FieldText, Label: "Arguments"},
			"interval_number": {Name: "interval_number", Type: metadata.FieldInteger, Label: "Interval Number"},
			"interval_type":   {Name: "interval_type", Type: metadata.FieldChar, Label: "Interval Type"},
			"number_calls":    {Name: "number_calls", Type: metadata.FieldInteger, Label: "Number of Calls"},
			"next_call":       {Name: "next_call", Type: metadata.FieldDateTime, Label: "Next Call"},
			"active":          {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("ir.export", &metadata.Model{
		Name:  "ir.export",
		Table: "ir_export",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"module": {Name: "module", Type: metadata.FieldChar, Label: "Module"},
		},
	})

	metadata.RegisterModel("ir.export.line", &metadata.Model{
		Name:  "ir.export.line",
		Table: "ir_export_line",
		Fields: map[string]*metadata.Field{
			"export": {Name: "export", Type: metadata.FieldMany2One, Label: "Export", Relation: "ir.export"},
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name"},
		},
	})

	metadata.RegisterModel("ir.attachment", &metadata.Model{
		Name:  "ir.attachment",
		Table: "ir_attachment",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"description": {Name: "description", Type: metadata.FieldText, Label: "Description"},
			"resource":   {Name: "resource", Type: metadata.FieldReference, Label: "Resource"},
			"file_id":    {Name: "file_id", Type: metadata.FieldChar, Label: "File"},
			"filename":   {Name: "filename", Type: metadata.FieldChar, Label: "Filename"},
			"mimetype":   {Name: "mimetype", Type: metadata.FieldChar, Label: "MIME Type"},
			"size":       {Name: "size", Type: metadata.FieldInteger, Label: "Size"},
			"create_date": {Name: "create_date", Type: metadata.FieldDateTime, Label: "Create Date"},
			"write_date":  {Name: "write_date", Type: metadata.FieldDateTime, Label: "Write Date"},
		},
	})

	metadata.RegisterModel("ir.email", &metadata.Model{
		Name:  "ir.email",
		Table: "ir_email",
		Fields: map[string]*metadata.Field{
			"from":    {Name: "from", Type: metadata.FieldChar, Label: "From"},
			"to":      {Name: "to", Type: metadata.FieldChar, Label: "To"},
			"subject": {Name: "subject", Type: metadata.FieldChar, Label: "Subject"},
			"body":    {Name: "body", Type: metadata.FieldText, Label: "Body"},
		},
	})

	metadata.RegisterModel("ir.email.template", &metadata.Model{
		Name:  "ir.email.template",
		Table: "ir_email_template",
		Fields: map[string]*metadata.Field{
			"name":    {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"subject": {Name: "subject", Type: metadata.FieldChar, Label: "Subject"},
			"body":    {Name: "body", Type: metadata.FieldText, Label: "Body"},
		},
	})

	metadata.RegisterModel("ir.note", &metadata.Model{
		Name:  "ir.note",
		Table: "ir_note",
		Fields: map[string]*metadata.Field{
			"resource": {Name: "resource", Type: metadata.FieldReference, Label: "Resource"},
			"message":  {Name: "message", Type: metadata.FieldText, Label: "Message"},
			"unread":   {Name: "unread", Type: metadata.FieldBoolean, Label: "Unread"},
		},
	})

	metadata.RegisterModel("ir.chat.channel", &metadata.Model{
		Name:  "ir.chat.channel",
		Table: "ir_chat_channel",
		Fields: map[string]*metadata.Field{
			"resource": {Name: "resource", Type: metadata.FieldReference, Label: "Resource"},
			"name":     {Name: "name", Type: metadata.FieldChar, Label: "Name"},
		},
	})

	metadata.RegisterModel("ir.module", &metadata.Model{
		Name:  "ir.module",
		Table: "ir_module",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"state":  {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"version": {Name: "version", Type: metadata.FieldChar, Label: "Version"},
		},
	})

	metadata.RegisterModel("ir.module.dependency", &metadata.Model{
		Name:  "ir.module.dependency",
		Table: "ir_module_dependency",
		Fields: map[string]*metadata.Field{
			"module":   {Name: "module", Type: metadata.FieldMany2One, Label: "Module", Relation: "ir.module"},
			"dependency": {Name: "dependency", Type: metadata.FieldChar, Label: "Dependency"},
		},
	})
}
