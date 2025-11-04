package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type ItemHandler struct {
	repo              *repository.ItemRepository
	questRepo         *repository.QuestRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
}

func NewItemHandler(repo *repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

func NewItemHandlerWithRepos(
	repo *repository.ItemRepository,
	questRepo *repository.QuestRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
) *ItemHandler {
	return &ItemHandler{
		repo:              repo,
		questRepo:         questRepo,
		hideoutModuleRepo: hideoutModuleRepo,
	}
}

func (h *ItemHandler) List(c *gin.Context) {
	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit
	items, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *ItemHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	item, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) Create(c *gin.Context) {
	var item models.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if item.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var item models.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.ID = uint(id)
	err = h.repo.Update(&item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// RequiredItemUsage represents where and how an item is used
type RequiredItemUsage struct {
	SourceType string `json:"source_type"` // "quest" or "hideout_module"
	SourceID   uint   `json:"source_id"`
	SourceName string `json:"source_name"`
	Quantity   int    `json:"quantity"`
	Level      *int   `json:"level,omitempty"` // For hideout modules, which level requires this
}

// RequiredItemResponse represents an item with its requirements
type RequiredItemResponse struct {
	Item     *models.Item        `json:"item"`
	TotalQty int                 `json:"total_quantity"`
	Usages   []RequiredItemUsage `json:"usages"`
}

// RequiredItems returns all items required for quests and hideout modules
func (h *ItemHandler) RequiredItems(c *gin.Context) {
	if h.questRepo == nil || h.hideoutModuleRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Required repositories not initialized"})
		return
	}

	// Map to store item requirements: external_id -> RequiredItemResponse
	itemMap := make(map[string]*RequiredItemResponse)

	// Get all items once for name matching (used in text objective parsing)
	allItems, _, err := h.repo.FindAll(0, 10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
		return
	}

	// Create a map for quick item lookup by name (case-insensitive)
	itemNameMap := make(map[string]string) // lowercase name -> external_id
	for _, item := range allItems {
		itemNameMap[strings.ToLower(item.Name)] = item.ExternalID
		// Also add partial matches for common variations
		itemNameLower := strings.ToLower(item.Name)
		// Add without spaces, with underscores, etc.
		itemNameMap[strings.ReplaceAll(itemNameLower, " ", "")] = item.ExternalID
		itemNameMap[strings.ReplaceAll(itemNameLower, " ", "_")] = item.ExternalID
	}

	// Get all quests
	quests, _, err := h.questRepo.FindAll(0, 10000) // Get all quests
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quests"})
		return
	}

	// Process quests for item requirements
	for _, quest := range quests {
		// Check quest data for required items
		// Items might be in objectives, data.requirementItemIds, or data.requiredItems
		if quest.Data != nil || quest.Objectives != nil {
			h.extractItemsFromQuest(quest, itemMap, itemNameMap, allItems)
		}
	}

	// Get all hideout modules
	hideoutModules, _, err := h.hideoutModuleRepo.FindAll(0, 10000) // Get all modules
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout modules"})
		return
	}

	// Process hideout modules for item requirements
	for _, module := range hideoutModules {
		if module.Levels != nil {
			h.extractItemsFromHideoutModule(module, itemMap)
		}
	}

	// Convert map to slice
	result := make([]RequiredItemResponse, 0, len(itemMap))
	for _, reqItem := range itemMap {
		result = append(result, *reqItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  result,
		"total": len(result),
	})
}

// extractItemsFromQuest extracts required items from a quest's data
func (h *ItemHandler) extractItemsFromQuest(quest models.Quest, itemMap map[string]*RequiredItemResponse, itemNameMap map[string]string, allItems []models.Item) {
	// Track processed items to avoid duplicates
	processedItems := make(map[string]bool)

	// Helper function to process requirement items
	processReqItems := func(reqItems []interface{}, questID uint, questName string) {
		for _, reqItem := range reqItems {
			if itemReq, ok := reqItem.(map[string]interface{}); ok {
				itemID, qty := h.parseItemRequirement(itemReq)
				if itemID != "" && qty > 0 {
					// Create unique key to avoid duplicates
					key := fmt.Sprintf("quest:%d:%s", questID, itemID)
					if !processedItems[key] {
						h.addItemRequirement(itemMap, itemID, "quest", questID, questName, qty, nil)
						processedItems[key] = true
					}
				}
			}
		}
	}

	// Check quest.Data for requirementItemIds
	if quest.Data != nil {
		// Try various field names for requirement items at quest level
		fieldNames := []string{"requirementItemIds", "requiredItems", "requirements", "required_item_ids", "requirement_items"}
		for _, fieldName := range fieldNames {
			if reqItems, ok := quest.Data[fieldName].([]interface{}); ok {
				processReqItems(reqItems, quest.ID, quest.Name)
			}
		}

		// Try data.objectives - objectives might be objects with requirementItemIds or text strings
		if objectives, ok := quest.Data["objectives"].([]interface{}); ok {
			for _, obj := range objectives {
				// Check if objective is a string (text objective like "Get 3 ARC Alloy for Shani")
				if objStr, ok := obj.(string); ok {
					if itemID, qty := h.parseTextObjective(objStr, itemNameMap, allItems); itemID != "" && qty > 0 {
						key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
						if !processedItems[key] {
							h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
							processedItems[key] = true
						}
					}
					continue
				}

				// Check if objective is an object/map
				if objMap, ok := obj.(map[string]interface{}); ok {
					// Check various field names in objectives
					for _, fieldName := range []string{"requirementItemIds", "requiredItems", "requirements", "required_item_ids", "requirement_items"} {
						if reqItems, ok := objMap[fieldName].([]interface{}); ok {
							processReqItems(reqItems, quest.ID, quest.Name)
						}
					}
					// Also check if the objective itself is a requirement object
					if itemID, qty := h.parseItemRequirement(objMap); itemID != "" && qty > 0 {
						key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
						if !processedItems[key] {
							h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
							processedItems[key] = true
						}
					}
					// Check if objective has a text field that might contain item requirements
					if textField, ok := objMap["text"].(string); ok {
						if itemID, qty := h.parseTextObjective(textField, itemNameMap, allItems); itemID != "" && qty > 0 {
							key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
							if !processedItems[key] {
								h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
								processedItems[key] = true
							}
						}
					}
					if descField, ok := objMap["description"].(string); ok {
						if itemID, qty := h.parseTextObjective(descField, itemNameMap, allItems); itemID != "" && qty > 0 {
							key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
							if !processedItems[key] {
								h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
								processedItems[key] = true
							}
						}
					}
				}
			}
		}
	}

	// Also check the Objectives field directly (stored separately in the model)
	if quest.Objectives != nil {
		// Objectives is stored as {"objectives": [...]}
		if objectivesData, ok := quest.Objectives["objectives"].([]interface{}); ok {
			for _, obj := range objectivesData {
				// Check if objective is a string (text objective like "Get 3 ARC Alloy for Shani")
				if objStr, ok := obj.(string); ok {
					if itemID, qty := h.parseTextObjective(objStr, itemNameMap, allItems); itemID != "" && qty > 0 {
						key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
						if !processedItems[key] {
							h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
							processedItems[key] = true
						}
					}
					continue
				}

				if objMap, ok := obj.(map[string]interface{}); ok {
					// Check various field names in objectives
					for _, fieldName := range []string{"requirementItemIds", "requiredItems", "requirements", "required_item_ids", "requirement_items"} {
						if reqItems, ok := objMap[fieldName].([]interface{}); ok {
							processReqItems(reqItems, quest.ID, quest.Name)
						}
					}
					// Also check if the objective itself is a requirement object
					if itemID, qty := h.parseItemRequirement(objMap); itemID != "" && qty > 0 {
						key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
						if !processedItems[key] {
							h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
							processedItems[key] = true
						}
					}
					// Check if objective has a text field that might contain item requirements
					if textField, ok := objMap["text"].(string); ok {
						if itemID, qty := h.parseTextObjective(textField, itemNameMap, allItems); itemID != "" && qty > 0 {
							key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
							if !processedItems[key] {
								h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
								processedItems[key] = true
							}
						}
					}
					if descField, ok := objMap["description"].(string); ok {
						if itemID, qty := h.parseTextObjective(descField, itemNameMap, allItems); itemID != "" && qty > 0 {
							key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
							if !processedItems[key] {
								h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
								processedItems[key] = true
							}
						}
					}
				}
			}
		}
	}
}

// extractItemsFromHideoutModule extracts required items from hideout module levels
func (h *ItemHandler) extractItemsFromHideoutModule(module models.HideoutModule, itemMap map[string]*RequiredItemResponse) {
	// Track processed items per level to avoid duplicates
	processedItems := make(map[string]bool)

	// Helper function to process requirement items for a level
	processLevelReqItems := func(reqItems []interface{}, moduleID uint, moduleName string, levelNum int) {
		for _, reqItem := range reqItems {
			if itemReq, ok := reqItem.(map[string]interface{}); ok {
				itemID, qty := h.parseItemRequirement(itemReq)
				if itemID != "" && qty > 0 {
					// Create unique key to avoid duplicates (module:level:item)
					key := fmt.Sprintf("hideout_module:%d:level:%d:%s", moduleID, levelNum, itemID)
					if !processedItems[key] {
						h.addItemRequirement(itemMap, itemID, "hideout_module", moduleID, moduleName, qty, &levelNum)
						processedItems[key] = true
					}
				}
			}
		}
	}

	// Prefer Levels field if it exists, otherwise fall back to Data field
	var levelsData []interface{}
	var foundLevels bool

	if module.Levels != nil {
		// Levels is stored as {"levels": [...]}
		if data, ok := module.Levels["levels"].([]interface{}); ok {
			levelsData = data
			foundLevels = true
		}
	}

	// If Levels field doesn't have data, try Data field
	if !foundLevels && module.Data != nil {
		if data, ok := module.Data["levels"].([]interface{}); ok {
			levelsData = data
			foundLevels = true
		}
	}

	if !foundLevels {
		return
	}

	// Process each level
	for levelIdx, levelData := range levelsData {
		if level, ok := levelData.(map[string]interface{}); ok {
			levelNum := levelIdx + 1 // Levels are 1-indexed

			// Check requirementItemIds in this level
			if reqItems, ok := level["requirementItemIds"].([]interface{}); ok {
				processLevelReqItems(reqItems, module.ID, module.Name, levelNum)
			}
		}
	}
}

// parseItemRequirement extracts item ID and quantity from a requirement object
func (h *ItemHandler) parseItemRequirement(itemReq map[string]interface{}) (string, int) {
	var itemID string
	var qty int

	// Try various field names for item ID (camelCase, snake_case, and other variations)
	itemIDFields := []string{"itemId", "item_id", "id", "item", "itemName", "item_name", "itemID"}
	for _, field := range itemIDFields {
		if id, ok := itemReq[field].(string); ok && id != "" {
			itemID = id
			break
		}
	}

	// Try quantity (various formats and field names)
	quantityFields := []string{"quantity", "qty", "amount", "count"}
	for _, field := range quantityFields {
		if quantity, ok := itemReq[field].(float64); ok {
			qty = int(quantity)
			break
		} else if quantity, ok := itemReq[field].(int); ok {
			qty = quantity
			break
		}
	}

	return itemID, qty
}

// parseTextObjective extracts item name and quantity from text objectives like "Get 3 ARC Alloy for Shani"
func (h *ItemHandler) parseTextObjective(objectiveText string, itemNameMap map[string]string, allItems []models.Item) (string, int) {
	// Patterns to match:
	// "Get 3 ARC Alloy for Shani"
	// "Collect 5 Steel"
	// "Obtain 10 Materials"
	// "Get X ItemName" or "Collect X ItemName" or "Obtain X ItemName"

	patterns := []*regexp.Regexp{
		// "Get X ItemName" or "Get X ItemName for Y"
		regexp.MustCompile(`(?i)^get\s+(\d+)\s+(.+?)(?:\s+for\s+|\s*$)`),
		// "Collect X ItemName" or "Collect X ItemName for Y"
		regexp.MustCompile(`(?i)^collect\s+(\d+)\s+(.+?)(?:\s+for\s+|\s*$)`),
		// "Obtain X ItemName" or "Obtain X ItemName for Y"
		regexp.MustCompile(`(?i)^obtain\s+(\d+)\s+(.+?)(?:\s+for\s+|\s*$)`),
		// "Gather X ItemName"
		regexp.MustCompile(`(?i)^gather\s+(\d+)\s+(.+?)(?:\s+for\s+|\s*$)`),
		// "Find X ItemName"
		regexp.MustCompile(`(?i)^find\s+(\d+)\s+(.+?)(?:\s+for\s+|\s*$)`),
	}

	objectiveText = strings.TrimSpace(objectiveText)

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(objectiveText)
		if len(matches) >= 3 {
			qty, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			itemName := strings.TrimSpace(matches[2])
			itemNameLower := strings.ToLower(itemName)

			// First try exact match in the name map
			if itemID, found := itemNameMap[itemNameLower]; found {
				return itemID, qty
			}

			// Try without spaces (e.g., "ARC Alloy" -> "arcalloy")
			itemNameNoSpaces := strings.ReplaceAll(itemNameLower, " ", "")
			if itemID, found := itemNameMap[itemNameNoSpaces]; found {
				return itemID, qty
			}

			// Try partial match - search through all items
			for _, item := range allItems {
				itemNameLowerDB := strings.ToLower(item.Name)
				// Exact match
				if itemNameLowerDB == itemNameLower {
					return item.ExternalID, qty
				}
				// Partial match - item name contains extracted name or vice versa
				if strings.Contains(itemNameLowerDB, itemNameLower) ||
					strings.Contains(itemNameLower, itemNameLowerDB) {
					return item.ExternalID, qty
				}
			}

			// If no match found, try searching by external_id containing the item name
			for _, item := range allItems {
				if strings.Contains(strings.ToLower(item.ExternalID), itemNameLower) {
					return item.ExternalID, qty
				}
			}
		}
	}

	return "", 0
}

// addItemRequirement adds or updates an item requirement in the map
func (h *ItemHandler) addItemRequirement(
	itemMap map[string]*RequiredItemResponse,
	itemID string,
	sourceType string,
	sourceID uint,
	sourceName string,
	quantity int,
	level *int,
) {
	if itemID == "" || quantity <= 0 {
		return
	}

	// Get or create the item response
	reqItem, exists := itemMap[itemID]
	if !exists {
		// Look up the item by external_id
		item, err := h.repo.FindByExternalID(itemID)
		if err != nil {
			// Item not found, create a placeholder with just the ID
			item = &models.Item{
				ExternalID: itemID,
				Name:       fmt.Sprintf("Unknown Item (%s)", itemID),
			}
		}

		reqItem = &RequiredItemResponse{
			Item:     item,
			TotalQty: 0,
			Usages:   []RequiredItemUsage{},
		}
		itemMap[itemID] = reqItem
	}

	// Check if we already have a usage for this exact source and level
	var existingUsage *RequiredItemUsage
	for i := range reqItem.Usages {
		usage := &reqItem.Usages[i]
		if usage.SourceType == sourceType &&
			usage.SourceID == sourceID &&
			((level == nil && usage.Level == nil) || (level != nil && usage.Level != nil && *level == *usage.Level)) {
			existingUsage = usage
			break
		}
	}

	if existingUsage != nil {
		// Merge quantities for duplicate usage
		existingUsage.Quantity += quantity
		reqItem.TotalQty += quantity
	} else {
		// Add new usage
		usage := RequiredItemUsage{
			SourceType: sourceType,
			SourceID:   sourceID,
			SourceName: sourceName,
			Quantity:   quantity,
			Level:      level,
		}
		reqItem.Usages = append(reqItem.Usages, usage)
		reqItem.TotalQty += quantity
	}
}
