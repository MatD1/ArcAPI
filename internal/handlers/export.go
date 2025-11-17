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
)

type ExportHandler struct {
	questRepo           *repository.QuestRepository
	itemRepo            *repository.ItemRepository
	skillNodeRepo       *repository.SkillNodeRepository
	hideoutModuleRepo   *repository.HideoutModuleRepository
	enemyTypeRepo       *repository.EnemyTypeRepository
	alertRepo           *repository.AlertRepository
}

func NewExportHandler(
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	enemyTypeRepo *repository.EnemyTypeRepository,
	alertRepo *repository.AlertRepository,
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

// Convert quests to CSV format
func (h *ExportHandler) questsToCSV(quests []models.Quest) [][]string {
	headers := []string{
		"id", "external_id", "name", "description", "trader", "xp",
		"objectives", "reward_item_ids", "data",
	}
	
	rows := [][]string{headers}
	
	for _, quest := range quests {
		objectives := h.jsonToStringArray(quest.Objectives)
		rewardItemIds := h.jsonToStringArray(quest.RewardItemIds)
		data := h.jsonToStringArray(quest.Data)
		
		// Extract name and description, checking Data field if direct fields are empty
		name := quest.Name
		if name == "" && quest.Data != nil {
			if dataName, ok := quest.Data["name"]; ok {
				if nameStr, ok := dataName.(string); ok {
					name = nameStr
				} else {
					// If it's a JSON object, convert to JSON string
					if nameBytes, err := json.Marshal(dataName); err == nil {
						name = string(nameBytes)
					}
				}
			}
		}
		
		description := quest.Description
		if description == "" && quest.Data != nil {
			if dataDesc, ok := quest.Data["description"]; ok {
				if descStr, ok := dataDesc.(string); ok {
					description = descStr
				} else {
					// If it's a JSON object, convert to JSON string
					if descBytes, err := json.Marshal(dataDesc); err == nil {
						description = string(descBytes)
					}
				}
			}
		}
		
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
		"id", "external_id", "name", "description", "type",
		"image_url", "image_filename", "data",
	}
	
	rows := [][]string{headers}
	
	for _, item := range items {
		data := h.jsonToStringArray(item.Data)
		
		row := []string{
			strconv.Itoa(int(item.ID)),
			item.ExternalID,
			item.Name,
			item.Description,
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
		"id", "external_id", "name", "description", "impacted_skill", "category",
		"max_points", "icon_name", "is_major", "position", "known_value",
		"prerequisite_node_ids", "data",
	}
	
	rows := [][]string{headers}
	
	for _, node := range skillNodes {
		position := h.jsonToStringArray(node.Position)
		knownValue := h.jsonToStringArray(node.KnownValue)
		prerequisiteNodeIds := h.jsonToStringArray(node.PrerequisiteNodeIds)
		data := h.jsonToStringArray(node.Data)
		
		row := []string{
			strconv.Itoa(int(node.ID)),
			node.ExternalID,
			node.Name,
			node.Description,
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
		"id", "external_id", "name", "description", "max_level",
		"levels", "data",
	}
	
	rows := [][]string{headers}
	
	for _, module := range modules {
		levels := h.jsonToStringArray(module.Levels)
		data := h.jsonToStringArray(module.Data)
		
		row := []string{
			strconv.Itoa(int(module.ID)),
			module.ExternalID,
			module.Name,
			module.Description,
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
		"id", "external_id", "name", "description", "type",
		"image_url", "image_filename", "weakpoints", "data",
	}
	
	rows := [][]string{headers}
	
	for _, enemyType := range enemyTypes {
		weakpoints := h.jsonToStringArray(enemyType.Weakpoints)
		data := h.jsonToStringArray(enemyType.Data)
		
		row := []string{
			strconv.Itoa(int(enemyType.ID)),
			enemyType.ExternalID,
			enemyType.Name,
			enemyType.Description,
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
		"id", "name", "description", "severity", "is_active",
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

