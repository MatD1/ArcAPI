package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	defer sqlDB.Close()

	fmt.Println("🌱 Starting database seeding...")

	if len(os.Args) > 1 && os.Args[1] == "--reset" {
		fmt.Println("🔄 Resetting database (deleting all data)...")
		db.DB.Exec("DELETE FROM user_quest_progress")
		db.DB.Exec("DELETE FROM user_skill_node_progress")
		db.DB.Exec("DELETE FROM user_hideout_module_progress")
		db.DB.Exec("DELETE FROM user_blueprint_progress")
		db.DB.Exec("DELETE FROM users")
		db.DB.Exec("DELETE FROM quests")
		db.DB.Exec("DELETE FROM items")
		db.DB.Exec("DELETE FROM skill_nodes")
		db.DB.Exec("DELETE FROM hideout_modules")
		fmt.Println("✓ Database cleared")
	}

	if err := seedUsers(db.DB); err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}

	if err := seedItems(db.DB); err != nil {
		log.Fatalf("Failed to seed items: %v", err)
	}

	if err := seedQuests(db.DB); err != nil {
		log.Fatalf("Failed to seed quests: %v", err)
	}

	if err := seedSkillNodes(db.DB); err != nil {
		log.Fatalf("Failed to seed skill nodes: %v", err)
	}

	if err := seedHideoutModules(db.DB); err != nil {
		log.Fatalf("Failed to seed hideout modules: %v", err)
	}

	fmt.Println("\n✅ Database seeding completed successfully!")
	fmt.Println("\n📊 Next steps:")
	fmt.Println("   1. Run 'make run' to start the development server")
	fmt.Println("   2. Login with email: dev@arcraiders.com (password: auto-generated during OAuth)")
	fmt.Println("   3. Access the dashboard at http://localhost:8080/dashboard")
}

func seedUsers(db *gorm.DB) error {
	fmt.Println("\n📝 Seeding users...")

	var existingUser models.User
	result := db.Where("email = ?", "dev@arcraiders.com").First(&existingUser)
	if result.Error == nil {
		fmt.Println("  ✓ Test user already exists (dev@arcraiders.com)")
		return nil
	}

	users := []models.User{
		{
			Email:         "dev@arcraiders.com",
			Username:      "devuser",
			Role:          models.RoleAdmin,
			CanAccessData: true,
			CreatedViaApp: false,
		},
	}

	for _, user := range users {
		if result := db.Create(&user); result.Error != nil {
			return fmt.Errorf("failed to create user: %w", result.Error)
		}
		fmt.Printf("  ✓ Created user: %s (%s)\n", user.Email, user.Username)
	}

	return nil
}

func seedItems(db *gorm.DB) error {
	fmt.Println("\n📦 Seeding items...")

	items := []models.Item{
		{
			ExternalID:  "item_test_01",
			Name:        "Test Ammunition",
			Description: "Sample ammunition item for development",
			Type:        "Ammunition",
			Data: models.JSONB{
				"rarity":    "common",
				"stackable": true,
				"max_stack": 999,
			},
			SyncedAt: time.Now(),
		},
		{
			ExternalID:  "item_test_02",
			Name:        "Test Component",
			Description: "Sample component item for development",
			Type:        "Component",
			Data: models.JSONB{
				"rarity":    "uncommon",
				"stackable": true,
				"max_stack": 100,
			},
			SyncedAt: time.Now(),
		},
		{
			ExternalID:  "item_test_03",
			Name:        "Test Attachment",
			Description: "Sample attachment item for development",
			Type:        "Attachment",
			Data: models.JSONB{
				"rarity":    "rare",
				"stackable": false,
				"max_stack": 1,
			},
			SyncedAt: time.Now(),
		},
	}

	for _, item := range items {
		var existingItem models.Item
		result := db.Where("external_id = ?", item.ExternalID).First(&existingItem)
		if result.Error == nil {
			continue
		}

		if result := db.Create(&item); result.Error != nil {
			return fmt.Errorf("failed to create item %s: %w", item.ExternalID, result.Error)
		}
		fmt.Printf("  ✓ Created item: %s\n", item.Name)
	}

	return nil
}

func seedQuests(db *gorm.DB) error {
	fmt.Println("\n🎯 Seeding quests...")

	quests := []models.Quest{
		{
			ExternalID:  "quest_test_01",
			Name:        "Test Quest - Beginner",
			Description: "A sample beginner quest for development and testing",
			Trader:      "Prapor",
			XP:          1000,
			Objectives: models.JSONB{
				"items": []string{"item_test_01"},
				"count": 10,
			},
			RewardItemIds: models.JSONB{
				"items": []map[string]interface{}{
					{"item_id": "item_test_02", "quantity": 5},
				},
			},
			Data: models.JSONB{
				"level": 1,
				"type":  "pickup",
			},
			SyncedAt: time.Now(),
		},
		{
			ExternalID:  "quest_test_02",
			Name:        "Test Quest - Intermediate",
			Description: "A sample intermediate quest for testing progression",
			Trader:      "Mechanic",
			XP:          2500,
			Objectives: models.JSONB{
				"items": []string{"item_test_02"},
				"count": 5,
			},
			RewardItemIds: models.JSONB{
				"items": []map[string]interface{}{
					{"item_id": "item_test_03", "quantity": 2},
				},
			},
			Data: models.JSONB{
				"level": 10,
				"type":  "elimination",
			},
			SyncedAt: time.Now(),
		},
	}

	for _, quest := range quests {
		var existingQuest models.Quest
		result := db.Where("external_id = ?", quest.ExternalID).First(&existingQuest)
		if result.Error == nil {
			continue
		}

		if result := db.Create(&quest); result.Error != nil {
			return fmt.Errorf("failed to create quest %s: %w", quest.ExternalID, result.Error)
		}
		fmt.Printf("  ✓ Created quest: %s\n", quest.Name)
	}

	return nil
}

func seedSkillNodes(db *gorm.DB) error {
	fmt.Println("\n🎓 Seeding skill nodes...")

	skillNodes := []models.SkillNode{
		{
			ExternalID:  "skill_test_01",
			Name:        "Test Skill - Accuracy",
			Description: "Improves weapon accuracy during development testing",
			Category:    "Offensive",
			MaxPoints:   5,
			IconName:    "accuracy",
			Data: models.JSONB{
				"level_requirement": 1,
				"xp_cost":           1000,
			},
			SyncedAt: time.Now(),
		},
		{
			ExternalID:  "skill_test_02",
			Name:        "Test Skill - Recoil Control",
			Description: "Reduces weapon recoil during development testing",
			Category:    "Offensive",
			MaxPoints:   5,
			IconName:    "recoil",
			Data: models.JSONB{
				"level_requirement": 5,
				"xp_cost":           2000,
			},
			SyncedAt: time.Now(),
		},
	}

	for _, node := range skillNodes {
		var existingNode models.SkillNode
		result := db.Where("external_id = ?", node.ExternalID).First(&existingNode)
		if result.Error == nil {
			continue
		}

		if result := db.Create(&node); result.Error != nil {
			return fmt.Errorf("failed to create skill node %s: %w", node.ExternalID, result.Error)
		}
		fmt.Printf("  ✓ Created skill node: %s\n", node.Name)
	}

	return nil
}

func seedHideoutModules(db *gorm.DB) error {
	fmt.Println("\n🏠 Seeding hideout modules...")

	hideoutModules := []models.HideoutModule{
		{
			ExternalID:  "hideout_test_01",
			Name:        "Test Module - Workbench",
			Description: "Workbench for crafting and testing",
			MaxLevel:    1,
			Levels: models.JSONB{
				"levels": []map[string]interface{}{
					{
						"level": 1,
						"cost":  50000,
						"time":  3600,
					},
				},
			},
			Data: models.JSONB{
				"type": "Craft",
			},
			SyncedAt: time.Now(),
		},
	}

	for _, module := range hideoutModules {
		var existingModule models.HideoutModule
		result := db.Where("external_id = ?", module.ExternalID).First(&existingModule)
		if result.Error == nil {
			continue
		}

		if result := db.Create(&module); result.Error != nil {
			return fmt.Errorf("failed to create hideout module %s: %w", module.ExternalID, result.Error)
		}
		fmt.Printf("  ✓ Created hideout module: %s\n", module.Name)
	}

	return nil
}
