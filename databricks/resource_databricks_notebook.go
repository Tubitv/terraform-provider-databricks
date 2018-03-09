package databricks

import (
	"encoding/base64"
	"fmt"
	"github.com/betabandido/databricks-sdk-go/client"
	"github.com/betabandido/databricks-sdk-go/models"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceDatabricksNotebook() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatabricksNotebookCreate,
		Read:   resourceDatabricksNotebookRead,
		Update: resourceDatabricksNotebookUpdate,
		Delete: resourceDatabricksNotebookDelete,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"language": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceDatabricksNotebookCreate(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*Client).workspace

	path := d.Get("path").(string)
	language := models.WorkspaceLanguage(d.Get("language").(string))
	content := d.Get("content").(string)

	err := apiClient.Import(&models.WorkspaceImportRequest{
		Path:     path,
		Language: &language,
		Content:  content,
	})
	if err != nil {
		return err
	}

	d.SetId(path)

	return nil
}

func resourceDatabricksNotebookRead(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*Client).workspace

	format := models.SOURCE

	resp, err := apiClient.Export(&models.WorkspaceExportRequest{
		Path:   d.Id(),
		Format: &format,
	})
	if err != nil {
		if databricksError, ok := err.(client.Error); ok && databricksError.Code() == "RESOURCE_DOES_NOT_EXIST" {
			d.SetId("")
			return nil
		}
		return err
	}

	content, err := resourceDatabricksNotebookSanitizeContent(resp.Content)
	if err != nil {
		return err
	}

	d.Set("content", *content)

	return nil
}

func resourceDatabricksNotebookSanitizeContent(content string) (*string, error) {
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(decoded), "\n")
	if lines[0] != "# Databricks notebook source" {
		return nil, fmt.Errorf("notebook starts with unexpected text: %s", lines[0])
	}

	sanitized := strings.Join(lines[1:], "\n")

	content = base64.StdEncoding.EncodeToString([]byte(sanitized))

	return &content, nil
}

func resourceDatabricksNotebookUpdate(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*Client).workspace

	if d.HasChange("content") {
		_, newContent := d.GetChange("content")
		language := models.WorkspaceLanguage(d.Get("language").(string))

		err := apiClient.Import(&models.WorkspaceImportRequest{
			Path:      d.Id(),
			Language:  &language,
			Content:   newContent.(string),
			Overwrite: true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceDatabricksNotebookDelete(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*Client).workspace

	err := apiClient.Delete(&models.WorkspaceDeleteRequest{
		Path: d.Id(),
	})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}