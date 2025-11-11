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
	"github.com/mat/arcapi/internal/services"
)

type ItemHandler struct {
	repo              *repository.ItemRepository
	questRepo         *repository.QuestRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
	dataCacheService  *services.DataCacheService
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

func NewItemHandlerWithCache(
	repo *repository.ItemRepository,
	questRepo *repository.QuestRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	dataCacheService *services.DataCacheService,
) *ItemHandler {
	return &ItemHandler{
		repo:              repo,
		questRepo:         questRepo,
		hideoutModuleRepo: hideoutModuleRepo,
		dataCacheService:  dataCacheService,
	}
}

func (h *ItemHandler) List(c *gin.Context) {
	// Check if unpaginated request
	if c.Query("all") == "true" {
		h.ListAll(c)
		return
	}

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
	var items []models.Item
	var count int64
	var err error

	// Use cache service if available
	if h.dataCacheService != nil {
		items, count, err = h.dataCacheService.GetItems(offset, limit)
	} else {
		// Fallback to direct database query
		items, count, err = h.repo.FindAll(offset, limit)
	}

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

func (h *ItemHandler) ListAll(c *gin.Context) {
	var items []models.Item
	var count int64
	var err error

	// Use cache service if available - get all items
	if h.dataCacheService != nil {
		items, count, err = h.dataCacheService.GetItems(0, 999999)
	} else {
		// Fallback to direct database query
		items, count, err = h.repo.FindAll(0, 999999)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": count,
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

	// Invalidate cache on create
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateItemsCache()
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

	// Invalidate cache on update
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateItemsCache()
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

	// Invalidate cache on delete
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateItemsCache()
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

	// Helper function to extract multilingual name from item
	extractItemName := func(item models.Item) string {
		name := item.Name
		if name == "" && item.Data != nil {
			dataMap := map[string]interface{}(item.Data)
			if nameObj, ok := dataMap["name"].(map[string]interface{}); ok {
				// Try English first
				if enName, ok := nameObj["en"].(string); ok && enName != "" {
					name = enName
				} else {
					// Try any available language
					for _, val := range nameObj {
						if nameStr, ok := val.(string); ok && nameStr != "" {
							name = nameStr
							break
						}
					}
				}
			}
		}
		return name
	}

	// Create a map for quick item lookup by name (case-insensitive)
	// Uses multilingual names when available
	itemNameMap := make(map[string]string) // lowercase name -> external_id
	for _, item := range allItems {
		itemName := extractItemName(item)
		if itemName == "" {
			itemName = item.ExternalID // Fallback to external_id
		}
		itemNameLower := strings.ToLower(itemName)
		itemNameMap[itemNameLower] = item.ExternalID
		// Also add partial matches for common variations
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

	// Helper function to extract multilingual name (prefers English, falls back to first available)
	extractMultilingualName := func(data map[string]interface{}, defaultName string) string {
		if data == nil {
			return defaultName
		}
		if nameObj, ok := data["name"].(map[string]interface{}); ok {
			// Try English first
			if enName, ok := nameObj["en"].(string); ok && enName != "" {
				return enName
			}
			// Try any available language
			for _, val := range nameObj {
				if nameStr, ok := val.(string); ok && nameStr != "" {
					return nameStr
				}
			}
		}
		return defaultName
	}

	// Helper function to extract multilingual text from item data
	extractMultilingualText := func(data map[string]interface{}, field string, defaultText string) string {
		if data == nil {
			return defaultText
		}
		if fieldObj, ok := data[field].(map[string]interface{}); ok {
			// Try English first
			if enText, ok := fieldObj["en"].(string); ok && enText != "" {
				return enText
			}
			// Try any available language
			for _, val := range fieldObj {
				if textStr, ok := val.(string); ok && textStr != "" {
					return textStr
				}
			}
		}
		return defaultText
	}

	// Update source names in the item map with multilingual names
	for _, reqItem := range itemMap {
		// Update item name and description if multilingual data exists
		if reqItem.Item != nil && reqItem.Item.Data != nil {
			dataMap := map[string]interface{}(reqItem.Item.Data)
			if reqItem.Item.Name == "" {
				if name := extractMultilingualText(dataMap, "name", ""); name != "" {
					reqItem.Item.Name = name
				}
			}
			if reqItem.Item.Description == "" {
				if desc := extractMultilingualText(dataMap, "description", ""); desc != "" {
					reqItem.Item.Description = desc
				}
			}
		}

		// Update usage source names
		for i := range reqItem.Usages {
			usage := &reqItem.Usages[i]
			if usage.SourceType == "quest" {
				// Find the quest to get its data
				for _, quest := range quests {
					if quest.ID == usage.SourceID {
						if quest.Data != nil {
							if name := extractMultilingualName(map[string]interface{}(quest.Data), usage.SourceName); name != "" {
								usage.SourceName = name
							}
						}
						break
					}
				}
			} else if usage.SourceType == "hideout_module" {
				// Find the hideout module to get its data
				for _, module := range hideoutModules {
					if module.ID == usage.SourceID {
						if module.Data != nil {
							if name := extractMultilingualName(map[string]interface{}(module.Data), usage.SourceName); name != "" {
								usage.SourceName = name
							}
						}
						break
					}
				}
			}
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

// BlueprintItem represents a blueprint item with relevant information
type BlueprintItem struct {
	ID            uint                   `json:"id"`
	ExternalID    string                 `json:"external_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Type          string                 `json:"type,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	ImageFilename string                 `json:"image_filename,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	SyncedAt      string                 `json:"synced_at"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// GetBlueprints returns all blueprint items
// Blueprints are identified by:
// 1. Type field containing "Blueprint" (case-insensitive)
// 2. Name containing "Blueprint" (case-insensitive)
// 3. Data field containing blueprint-related keys
func (h *ItemHandler) GetBlueprints(c *gin.Context) {
	// Get all items
	allItems, _, err := h.repo.FindAll(0, 100000) // Get all items
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
		return
	}

	var blueprints []BlueprintItem

	for _, item := range allItems {
		isBlueprint := false

		// Check 1: Type field
		if strings.Contains(strings.ToLower(item.Type), "blueprint") {
			isBlueprint = true
		}

		// Check 2: Name contains "Blueprint"
		if !isBlueprint && strings.Contains(strings.ToLower(item.Name), "blueprint") {
			isBlueprint = true
		}

		// Check 3: Data field contains blueprint indicators
		if !isBlueprint && item.Data != nil {
			dataMap := map[string]interface{}(item.Data)

			// Check for common blueprint-related fields
			blueprintFields := []string{
				"blueprint", "isBlueprint", "is_blueprint", "blueprintType",
				"blueprint_type", "craftable", "consumable", "recipe",
			}

			for _, field := range blueprintFields {
				if val, exists := dataMap[field]; exists {
					// If field exists and is truthy, it's likely a blueprint
					if boolVal, ok := val.(bool); ok && boolVal {
						isBlueprint = true
						break
					} else if val != nil && val != "" {
						isBlueprint = true
						break
					}
				}
			}

			// Also check if type in data is blueprint
			if typeVal, ok := dataMap["type"].(string); ok {
				if strings.Contains(strings.ToLower(typeVal), "blueprint") {
					isBlueprint = true
				}
			}
		}

		// Check 4: External ID pattern (some games use IDs like "bp_*" or "*_blueprint")
		if !isBlueprint {
			lowerID := strings.ToLower(item.ExternalID)
			if strings.Contains(lowerID, "blueprint") ||
				strings.HasPrefix(lowerID, "bp_") ||
				strings.HasSuffix(lowerID, "_bp") {
				isBlueprint = true
			}
		}

		if isBlueprint {
			// Extract multilingual name and description
			displayName := item.Name
			displayDescription := item.Description

			if item.Data != nil {
				dataMap := map[string]interface{}(item.Data)

				// Extract multilingual name
				if displayName == "" {
					if nameObj, ok := dataMap["name"].(map[string]interface{}); ok {
						// Try English first
						if enName, ok := nameObj["en"].(string); ok && enName != "" {
							displayName = enName
						} else {
							// Try any available language
							for _, val := range nameObj {
								if nameStr, ok := val.(string); ok && nameStr != "" {
									displayName = nameStr
									break
								}
							}
						}
					}
				}

				// Extract multilingual description
				if displayDescription == "" {
					if descObj, ok := dataMap["description"].(map[string]interface{}); ok {
						// Try English first
						if enDesc, ok := descObj["en"].(string); ok && enDesc != "" {
							displayDescription = enDesc
						} else {
							// Try any available language
							for _, val := range descObj {
								if descStr, ok := val.(string); ok && descStr != "" {
									displayDescription = descStr
									break
								}
							}
						}
					}
				}
			}

			// Fallback to external_id if no name found
			if displayName == "" {
				displayName = item.ExternalID
			}

			blueprint := BlueprintItem{
				ID:            item.ID,
				ExternalID:    item.ExternalID,
				Name:          displayName,
				Description:   displayDescription,
				Type:          item.Type,
				ImageURL:      item.ImageURL,
				ImageFilename: item.ImageFilename,
				SyncedAt:      item.SyncedAt.Format("2006-01-02T15:04:05Z07:00"),
				CreatedAt:     item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:     item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}

			// Include full data if present
			if item.Data != nil {
				blueprint.Data = map[string]interface{}(item.Data)
			}

			blueprints = append(blueprints, blueprint)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  blueprints,
		"total": len(blueprints),
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

		// Try data.objectives - objectives might be objects with requirementItemIds or text strings (or multilingual objects)
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

				// Check if objective is a multilingual object (has language codes as keys)
				if objMap, ok := obj.(map[string]interface{}); ok {
					// Check if it's a multilingual text object (has language codes like "en", "de", etc.)
					languageCodes := []string{"en", "de", "es", "fr", "it", "ja", "kr", "no", "pl", "pt", "ru", "tr", "uk", "zh-CN", "zh-TW", "da", "hr", "sr"}
					isMultilingual := false
					for key := range objMap {
						for _, lang := range languageCodes {
							if key == lang {
								isMultilingual = true
								break
							}
						}
						if isMultilingual {
							break
						}
					}

					if isMultilingual {
						// Extract English text first, fallback to any language
						var objectiveText string
						if enText, ok := objMap["en"].(string); ok && enText != "" {
							objectiveText = enText
						} else {
							// Try any available language
							for _, lang := range languageCodes {
								if text, ok := objMap[lang].(string); ok && text != "" {
									objectiveText = text
									break
								}
							}
						}

						if objectiveText != "" {
							if itemID, qty := h.parseTextObjective(objectiveText, itemNameMap, allItems); itemID != "" && qty > 0 {
								key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
								if !processedItems[key] {
									h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
									processedItems[key] = true
								}
							}
						}
						continue
					}
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

				// Check if objective is a multilingual object (has language codes as keys)
				if objMap, ok := obj.(map[string]interface{}); ok {
					// Check if it's a multilingual text object (has language codes like "en", "de", etc.)
					languageCodes := []string{"en", "de", "es", "fr", "it", "ja", "kr", "no", "pl", "pt", "ru", "tr", "uk", "zh-CN", "zh-TW", "da", "hr", "sr"}
					isMultilingual := false
					for key := range objMap {
						for _, lang := range languageCodes {
							if key == lang {
								isMultilingual = true
								break
							}
						}
						if isMultilingual {
							break
						}
					}

					if isMultilingual {
						// Extract English text first, fallback to any language
						var objectiveText string
						if enText, ok := objMap["en"].(string); ok && enText != "" {
							objectiveText = enText
						} else {
							// Try any available language
							for _, lang := range languageCodes {
								if text, ok := objMap[lang].(string); ok && text != "" {
									objectiveText = text
									break
								}
							}
						}

						if objectiveText != "" {
							if itemID, qty := h.parseTextObjective(objectiveText, itemNameMap, allItems); itemID != "" && qty > 0 {
								key := fmt.Sprintf("quest:%d:%s", quest.ID, itemID)
								if !processedItems[key] {
									h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
									processedItems[key] = true
								}
							}
						}
						continue
					}
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
			// Helper to extract multilingual item name
			getItemDisplayName := func(item models.Item) string {
				name := item.Name
				if name == "" && item.Data != nil {
					dataMap := map[string]interface{}(item.Data)
					if nameObj, ok := dataMap["name"].(map[string]interface{}); ok {
						// Try English first
						if enName, ok := nameObj["en"].(string); ok && enName != "" {
							name = enName
						} else {
							// Try any available language
							for _, val := range nameObj {
								if nameStr, ok := val.(string); ok && nameStr != "" {
									name = nameStr
									break
								}
							}
						}
					}
				}
				return name
			}

			for _, item := range allItems {
				itemDisplayName := getItemDisplayName(item)
				itemNameLowerDB := strings.ToLower(itemDisplayName)
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
