package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type ExportHandler struct {
	questRepo         *repository.QuestRepository
	itemRepo          *repository.ItemRepository
	skillNodeRepo     *repository.SkillNodeRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
	enemyTypeRepo     *repository.EnemyTypeRepository
	alertRepo         *repository.AlertRepository
	githubDataService *services.GitHubDataService
}

func NewExportHandler(
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	enemyTypeRepo *repository.EnemyTypeRepository,
	alertRepo *repository.AlertRepository,
	githubDataService *services.GitHubDataService,
) *ExportHandler {
	return &ExportHandler{
		questRepo:         questRepo,
		itemRepo:          itemRepo,
		skillNodeRepo:     skillNodeRepo,
		hideoutModuleRepo: hideoutModuleRepo,
		enemyTypeRepo:     enemyTypeRepo,
		alertRepo:         alertRepo,
	}
}

func (h *ExportHandler) ExportBots(c *gin.Context) {
	if h.githubDataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub data service unavailable"})
		return
	}
	data, err := h.githubDataService.GetBots(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bots data"})
		return
	}

	csvData := h.genericDataToCSV(data)
	h.sendCSV(c, csvData, "bots")
}

func (h *ExportHandler) ExportMaps(c *gin.Context) {
	if h.githubDataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub data service unavailable"})
		return
	}
	data, err := h.githubDataService.GetMaps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch maps data"})
		return
	}

	csvData := h.genericDataToCSV(data)
	h.sendCSV(c, csvData, "maps")
}

func (h *ExportHandler) ExportTraders(c *gin.Context) {
	if h.githubDataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub data service unavailable"})
		return
	}
	data, err := h.githubDataService.GetTraders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch traders data"})
		return
	}

	csvData := h.genericDataToCSV(data)
	h.sendCSV(c, csvData, "traders")
}

func (h *ExportHandler) ExportProjects(c *gin.Context) {
	if h.githubDataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub data service unavailable"})
		return
	}
	data, err := h.githubDataService.GetProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects data"})
		return
	}

	csvData := h.genericDataToCSV(data)
	h.sendCSV(c, csvData, "projects")
}

// ExportQuests exports all quests as CSV
func (h *ExportHandler) ExportQuests(c *gin.Context) {
	quests, _, err := h.questRepo.FindAll(0, 10000) // Get all quests
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quests"})
		return
	}

	csvData := h.questsToCSV(quests)
	h.sendCSV(c, csvData, "quests")
}

// ExportItems exports all items as CSV
func (h *ExportHandler) ExportItems(c *gin.Context) {
	items, _, err := h.itemRepo.FindAll(0, 10000) // Get all items
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
		return
	}

	csvData := h.itemsToCSV(items)
	h.sendCSV(c, csvData, "items")
}

// ExportSkillNodes exports all skill nodes as CSV
func (h *ExportHandler) ExportSkillNodes(c *gin.Context) {
	skillNodes, _, err := h.skillNodeRepo.FindAll(0, 10000) // Get all skill nodes
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill nodes"})
		return
	}

	csvData := h.skillNodesToCSV(skillNodes)
	h.sendCSV(c, csvData, "skill-nodes")
}

// ExportHideoutModules exports all hideout modules as CSV
func (h *ExportHandler) ExportHideoutModules(c *gin.Context) {
	hideoutModules, _, err := h.hideoutModuleRepo.FindAll(0, 10000) // Get all hideout modules
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout modules"})
		return
	}

	csvData := h.hideoutModulesToCSV(hideoutModules)
	h.sendCSV(c, csvData, "hideout-modules")
}

// ExportEnemyTypes exports all enemy types as CSV
func (h *ExportHandler) ExportEnemyTypes(c *gin.Context) {
	enemyTypes, _, err := h.enemyTypeRepo.FindAll(0, 10000) // Get all enemy types
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enemy types"})
		return
	}

	csvData := h.enemyTypesToCSV(enemyTypes)
	h.sendCSV(c, csvData, "enemy-types")
}

// ExportAlerts exports all alerts as CSV
func (h *ExportHandler) ExportAlerts(c *gin.Context) {
	alerts, _, err := h.alertRepo.FindAll(0, 10000) // Get all alerts
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	csvData := h.alertsToCSV(alerts)
	h.sendCSV(c, csvData, "alerts")
}

// Helper function to send CSV response
func (h *ExportHandler) sendCSV(c *gin.Context, csvData [][]string, filename string) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.csv", filename, time.Now().Format("20060102-150405")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	for _, record := range csvData {
		if err := writer.Write(record); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV"})
			return
		}
	}
}

// Helper function to extract English ("en") value from a field, checking both direct field and Data JSONB
func (h *ExportHandler) extractEnglishValue(directValue string, data models.JSONB, dataKey string) string {
	// First, check if direct value exists
	if directValue != "" {
		// If direct value is a JSON string, try to parse it
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(directValue), &parsed); err == nil {
			// It's a JSON object, extract "en"
			if enVal, ok := parsed["en"].(string); ok {
				return enVal
			}
		}
		// If it's not JSON or doesn't have "en", return as-is
		return directValue
	}

	// If direct value is empty, check Data field
	if data != nil {
		if dataValue, ok := data[dataKey]; ok {
			if strVal, ok := dataValue.(string); ok {
				// Try to parse as JSON object
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					if enVal, ok := parsed["en"].(string); ok {
						return enVal
					}
				}
				// If not JSON, return string as-is
				return strVal
			} else if mapVal, ok := dataValue.(map[string]interface{}); ok {
				// Already a map, extract "en"
				if enVal, ok := mapVal["en"].(string); ok {
					return enVal
				}
			}
		}
	}

	return ""
}

func (h *ExportHandler) genericDataToCSV(data interface{}) [][]string {
	headers := []string{"index", "data"}
	rows := [][]string{headers}

	switch list := data.(type) {
	case []interface{}:
		for idx, item := range list {
			rows = append(rows, []string{fmt.Sprintf("%d", idx), h.marshalValueToString(item)})
		}
	case map[string]interface{}:
		rows = append(rows, []string{"0", h.marshalValueToString(list)})
	default:
		rows = append(rows, []string{"0", h.marshalValueToString(list)})
	}

	return rows
}

func (h *ExportHandler) marshalValueToString(value interface{}) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// Convert quests to CSV format
func (h *ExportHandler) questsToCSV(quests []models.Quest) [][]string {
	headers := []string{
		"system_id", "external_id", "name", "description", "trader", "xp",
		"objectives", "reward_item_ids", "data",
	}

	rows := [][]string{headers}

	for _, quest := range quests {
		objectives := h.jsonToStringArray(quest.Objectives)
		rewardItemIds := h.jsonToStringArray(quest.RewardItemIds)
		data := h.jsonToStringArray(quest.Data)

		// Extract name, preferring "en" from JSON objects
		name := h.extractEnglishValue(quest.Name, quest.Data, "name")

		// Extract description, preferring "en" from JSON objects
		description := h.extractEnglishValue(quest.Description, quest.Data, "description")

		row := []string{
			strconv.Itoa(int(quest.ID)),
			quest.ExternalID,
			name,
			description,
			quest.Trader,
			strconv.Itoa(quest.XP),
			objectives,
			rewardItemIds,
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Convert items to CSV format
func (h *ExportHandler) itemsToCSV(items []models.Item) [][]string {
	headers := []string{
		"system_id", "external_id", "name", "description", "type",
		"image_url", "image_filename", "data",
	}

	rows := [][]string{headers}

	for _, item := range items {
		data := h.jsonToStringArray(item.Data)

		// Extract name and description, preferring "en" from JSON objects
		name := h.extractEnglishValue(item.Name, item.Data, "name")
		description := h.extractEnglishValue(item.Description, item.Data, "description")

		row := []string{
			strconv.Itoa(int(item.ID)),
			item.ExternalID,
			name,
			description,
			item.Type,
			item.ImageURL,
			item.ImageFilename,
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Convert skill nodes to CSV format
func (h *ExportHandler) skillNodesToCSV(skillNodes []models.SkillNode) [][]string {
	headers := []string{
		"system_id", "external_id", "name", "description", "impacted_skill", "category",
		"max_points", "icon_name", "is_major", "position", "known_value",
		"prerequisite_node_ids", "data",
	}

	rows := [][]string{headers}

	for _, node := range skillNodes {
		position := h.jsonToStringArray(node.Position)
		knownValue := h.jsonToStringArray(node.KnownValue)
		prerequisiteNodeIds := h.jsonToStringArray(node.PrerequisiteNodeIds)
		data := h.jsonToStringArray(node.Data)

		// Extract name and description, preferring "en" from JSON objects
		name := h.extractEnglishValue(node.Name, node.Data, "name")
		description := h.extractEnglishValue(node.Description, node.Data, "description")

		row := []string{
			strconv.Itoa(int(node.ID)),
			node.ExternalID,
			name,
			description,
			node.ImpactedSkill,
			node.Category,
			strconv.Itoa(node.MaxPoints),
			node.IconName,
			strconv.FormatBool(node.IsMajor),
			position,
			knownValue,
			prerequisiteNodeIds,
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Convert hideout modules to CSV format
func (h *ExportHandler) hideoutModulesToCSV(modules []models.HideoutModule) [][]string {
	headers := []string{
		"system_id", "external_id", "name", "description", "max_level",
		"levels", "data",
	}

	rows := [][]string{headers}

	for _, module := range modules {
		levels := h.jsonToStringArray(module.Levels)
		data := h.jsonToStringArray(module.Data)

		// Extract name and description, preferring "en" from JSON objects
		name := h.extractEnglishValue(module.Name, module.Data, "name")
		description := h.extractEnglishValue(module.Description, module.Data, "description")

		row := []string{
			strconv.Itoa(int(module.ID)),
			module.ExternalID,
			name,
			description,
			strconv.Itoa(module.MaxLevel),
			levels,
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Convert enemy types to CSV format
func (h *ExportHandler) enemyTypesToCSV(enemyTypes []models.EnemyType) [][]string {
	headers := []string{
		"system_id", "external_id", "name", "description", "type",
		"image_url", "image_filename", "weakpoints", "data",
	}

	rows := [][]string{headers}

	for _, enemyType := range enemyTypes {
		weakpoints := h.jsonToStringArray(enemyType.Weakpoints)
		data := h.jsonToStringArray(enemyType.Data)

		// Extract name and description, preferring "en" from JSON objects
		name := h.extractEnglishValue(enemyType.Name, enemyType.Data, "name")
		description := h.extractEnglishValue(enemyType.Description, enemyType.Data, "description")

		row := []string{
			strconv.Itoa(int(enemyType.ID)),
			enemyType.ExternalID,
			name,
			description,
			enemyType.Type,
			enemyType.ImageURL,
			enemyType.ImageFilename,
			weakpoints,
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Convert alerts to CSV format
func (h *ExportHandler) alertsToCSV(alerts []models.Alert) [][]string {
	headers := []string{
		"system_id", "name", "description", "severity", "is_active",
		"data",
	}

	rows := [][]string{headers}

	for _, alert := range alerts {
		data := h.jsonToStringArray(alert.Data)

		row := []string{
			strconv.Itoa(int(alert.ID)),
			alert.Name,
			alert.Description,
			alert.Severity,
			strconv.FormatBool(alert.IsActive),
			data,
		}
		rows = append(rows, row)
	}

	return rows
}

// Helper to convert JSONB to string array (for Appwrite compatibility)
func (h *ExportHandler) jsonToStringArray(jsonb models.JSONB) string {
	if jsonb == nil {
		return ""
	}

	// JSONB is map[string]interface{}, so we need to check the underlying value
	// First, marshal to see what we have
	data, err := json.Marshal(jsonb)
	if err != nil {
		return ""
	}

	// Try to unmarshal as array to check if it's an array
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		// It's an array, convert each element to string
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			itemData, err := json.Marshal(item)
			if err != nil {
				continue
			}
			result = append(result, string(itemData))
		}
		resultData, err := json.Marshal(result)
		if err != nil {
			return ""
		}
		return string(resultData)
	}

	// It's an object or other type, convert to single-element array
	result := []string{string(data)}
	resultData, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	return string(resultData)
}

// Helper to convert JSONB to string (deprecated - use jsonToStringArray for Appwrite)
func (h *ExportHandler) jsonToString(jsonb models.JSONB) string {
	return h.jsonToStringArray(jsonb)
}
