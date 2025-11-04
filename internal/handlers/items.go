package handlers

import (
	"fmt"
	"net/http"
	"strconv"

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
		if quest.Data != nil {
			h.extractItemsFromQuest(quest, itemMap)
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
func (h *ItemHandler) extractItemsFromQuest(quest models.Quest, itemMap map[string]*RequiredItemResponse) {
	// Check various possible locations for required items in quest data
	if quest.Data == nil {
		return
	}

	// Try data.requirementItemIds
	if reqItems, ok := quest.Data["requirementItemIds"].([]interface{}); ok {
		for _, reqItem := range reqItems {
			if itemReq, ok := reqItem.(map[string]interface{}); ok {
				itemID, qty := h.parseItemRequirement(itemReq)
				if itemID != "" && qty > 0 {
					h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
				}
			}
		}
	}

	// Try data.requiredItems (alternative field name)
	if reqItems, ok := quest.Data["requiredItems"].([]interface{}); ok {
		for _, reqItem := range reqItems {
			if itemReq, ok := reqItem.(map[string]interface{}); ok {
				itemID, qty := h.parseItemRequirement(itemReq)
				if itemID != "" && qty > 0 {
					h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
				}
			}
		}
	}

	// Try data.objectives (objectives might contain item requirements)
	if objectives, ok := quest.Data["objectives"].([]interface{}); ok {
		for _, obj := range objectives {
			if objMap, ok := obj.(map[string]interface{}); ok {
				// Check if objective has item requirements
				if reqItems, ok := objMap["requirementItemIds"].([]interface{}); ok {
					for _, reqItem := range reqItems {
						if itemReq, ok := reqItem.(map[string]interface{}); ok {
							itemID, qty := h.parseItemRequirement(itemReq)
							if itemID != "" && qty > 0 {
								h.addItemRequirement(itemMap, itemID, "quest", quest.ID, quest.Name, qty, nil)
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
	if module.Levels == nil {
		return
	}

	// Levels is stored as {"levels": [...]}
	levelsData, ok := module.Levels["levels"].([]interface{})
	if !ok {
		return
	}

	for levelIdx, levelData := range levelsData {
		if level, ok := levelData.(map[string]interface{}); ok {
			levelNum := levelIdx + 1 // Levels are 1-indexed

			// Check requirementItemIds
			if reqItems, ok := level["requirementItemIds"].([]interface{}); ok {
				for _, reqItem := range reqItems {
					if itemReq, ok := reqItem.(map[string]interface{}); ok {
						itemID, qty := h.parseItemRequirement(itemReq)
						if itemID != "" && qty > 0 {
							h.addItemRequirement(itemMap, itemID, "hideout_module", module.ID, module.Name, qty, &levelNum)
						}
					}
				}
			}

			// Also check the full data field if it exists
			if module.Data != nil {
				if levels, ok := module.Data["levels"].([]interface{}); ok && levelIdx < len(levels) {
					if levelData, ok := levels[levelIdx].(map[string]interface{}); ok {
						if reqItems, ok := levelData["requirementItemIds"].([]interface{}); ok {
							for _, reqItem := range reqItems {
								if itemReq, ok := reqItem.(map[string]interface{}); ok {
									itemID, qty := h.parseItemRequirement(itemReq)
									if itemID != "" && qty > 0 {
										h.addItemRequirement(itemMap, itemID, "hideout_module", module.ID, module.Name, qty, &levelNum)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// parseItemRequirement extracts item ID and quantity from a requirement object
func (h *ItemHandler) parseItemRequirement(itemReq map[string]interface{}) (string, int) {
	var itemID string
	var qty int

	// Try itemId (camelCase)
	if id, ok := itemReq["itemId"].(string); ok {
		itemID = id
	} else if id, ok := itemReq["item_id"].(string); ok {
		itemID = id
	} else if id, ok := itemReq["id"].(string); ok {
		itemID = id
	}

	// Try quantity (various formats)
	if quantity, ok := itemReq["quantity"].(float64); ok {
		qty = int(quantity)
	} else if quantity, ok := itemReq["quantity"].(int); ok {
		qty = quantity
	} else if quantity, ok := itemReq["qty"].(float64); ok {
		qty = int(quantity)
	} else if quantity, ok := itemReq["qty"].(int); ok {
		qty = quantity
	}

	return itemID, qty
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

	// Add usage
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
