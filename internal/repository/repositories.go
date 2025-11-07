package repository

import (
	"github.com/mat/arcapi/internal/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByGithubID(githubID string) (*models.User, error) {
	var user models.User
	err := r.db.Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByDiscordID(discordID string) (*models.User, error) {
	var user models.User
	err := r.db.Where("discord_id = ?", discordID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

func (r *UserRepository) FindAll(offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&users).Error
	return users, count, err
}

type APIKeyRepository struct {
	db *DB
}

func NewAPIKeyRepository(db *DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(key *models.APIKey) error {
	return r.db.Create(key).Error
}

func (r *APIKeyRepository) FindByHash(hash string) (*models.APIKey, error) {
	var key models.APIKey
	err := r.db.Preload("User").Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) FindByID(id uint) (*models.APIKey, error) {
	var key models.APIKey
	err := r.db.Preload("User").First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) FindByUserID(userID uint) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.Where("user_id = ?", userID).Find(&keys).Error
	return keys, err
}

func (r *APIKeyRepository) Revoke(id uint) error {
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("revoked_at", gorm.Expr("NOW()")).Error
}

func (r *APIKeyRepository) UpdateLastUsed(id uint) error {
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("last_used_at", gorm.Expr("NOW()")).Error
}

func (r *APIKeyRepository) FindAllActive() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.Preload("User").Where("revoked_at IS NULL").Find(&keys).Error
	return keys, err
}

func (r *APIKeyRepository) FindAll() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.Preload("User").Order("id ASC").Find(&keys).Error
	return keys, err
}

type JWTTokenRepository struct {
	db *DB
}

func NewJWTTokenRepository(db *DB) *JWTTokenRepository {
	return &JWTTokenRepository{db: db}
}

func (r *JWTTokenRepository) Create(token *models.JWTToken) error {
	return r.db.Create(token).Error
}

func (r *JWTTokenRepository) FindByHash(hash string) (*models.JWTToken, error) {
	var token models.JWTToken
	err := r.db.Preload("User").Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *JWTTokenRepository) FindActiveByUserID(userID uint) ([]models.JWTToken, error) {
	var tokens []models.JWTToken
	err := r.db.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, gorm.Expr("NOW()")).Find(&tokens).Error
	return tokens, err
}

func (r *JWTTokenRepository) Revoke(id uint) error {
	return r.db.Model(&models.JWTToken{}).Where("id = ?", id).Update("revoked_at", gorm.Expr("NOW()")).Error
}

func (r *JWTTokenRepository) RevokeByHash(hash string) error {
	return r.db.Model(&models.JWTToken{}).Where("token_hash = ?", hash).Update("revoked_at", gorm.Expr("NOW()")).Error
}

type QuestRepository struct {
	db *DB
}

func NewQuestRepository(db *DB) *QuestRepository {
	return &QuestRepository{db: db}
}

func (r *QuestRepository) Create(quest *models.Quest) error {
	return r.db.Create(quest).Error
}

func (r *QuestRepository) FindByID(id uint) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.First(&quest, id).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) FindByExternalID(externalID string) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.Where("external_id = ?", externalID).First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) FindAll(offset, limit int) ([]models.Quest, int64, error) {
	var quests []models.Quest
	var count int64
	err := r.db.Model(&models.Quest{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&quests).Error
	return quests, count, err
}

func (r *QuestRepository) Update(quest *models.Quest) error {
	return r.db.Save(quest).Error
}

func (r *QuestRepository) Delete(id uint) error {
	return r.db.Delete(&models.Quest{}, id).Error
}

func (r *QuestRepository) UpsertByExternalID(quest *models.Quest) error {
	var existing models.Quest
	err := r.db.Where("external_id = ?", quest.ExternalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(quest).Error
	}
	if err != nil {
		return err
	}
	quest.ID = existing.ID
	return r.db.Save(quest).Error
}

// MissionRepository is deprecated, use QuestRepository instead
type MissionRepository = QuestRepository

func NewMissionRepository(db *DB) *MissionRepository {
	return NewQuestRepository(db)
}

type ItemRepository struct {
	db *DB
}

func NewItemRepository(db *DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(item *models.Item) error {
	return r.db.Create(item).Error
}

func (r *ItemRepository) FindByID(id uint) (*models.Item, error) {
	var item models.Item
	err := r.db.First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepository) FindByExternalID(externalID string) (*models.Item, error) {
	var item models.Item
	err := r.db.Where("external_id = ?", externalID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepository) FindAll(offset, limit int) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	err := r.db.Model(&models.Item{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

func (r *ItemRepository) Update(item *models.Item) error {
	return r.db.Save(item).Error
}

func (r *ItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.Item{}, id).Error
}

func (r *ItemRepository) UpsertByExternalID(item *models.Item) error {
	var existing models.Item
	err := r.db.Where("external_id = ?", item.ExternalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(item).Error
	}
	if err != nil {
		return err
	}
	item.ID = existing.ID
	return r.db.Save(item).Error
}

type SkillNodeRepository struct {
	db *DB
}

func NewSkillNodeRepository(db *DB) *SkillNodeRepository {
	return &SkillNodeRepository{db: db}
}

func (r *SkillNodeRepository) Create(skillNode *models.SkillNode) error {
	return r.db.Create(skillNode).Error
}

func (r *SkillNodeRepository) FindByID(id uint) (*models.SkillNode, error) {
	var skillNode models.SkillNode
	err := r.db.First(&skillNode, id).Error
	if err != nil {
		return nil, err
	}
	return &skillNode, nil
}

func (r *SkillNodeRepository) FindByExternalID(externalID string) (*models.SkillNode, error) {
	var skillNode models.SkillNode
	err := r.db.Where("external_id = ?", externalID).First(&skillNode).Error
	if err != nil {
		return nil, err
	}
	return &skillNode, nil
}

func (r *SkillNodeRepository) FindAll(offset, limit int) ([]models.SkillNode, int64, error) {
	var skillNodes []models.SkillNode
	var count int64
	err := r.db.Model(&models.SkillNode{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&skillNodes).Error
	return skillNodes, count, err
}

func (r *SkillNodeRepository) Update(skillNode *models.SkillNode) error {
	return r.db.Save(skillNode).Error
}

func (r *SkillNodeRepository) Delete(id uint) error {
	return r.db.Delete(&models.SkillNode{}, id).Error
}

func (r *SkillNodeRepository) UpsertByExternalID(skillNode *models.SkillNode) error {
	var existing models.SkillNode
	err := r.db.Where("external_id = ?", skillNode.ExternalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(skillNode).Error
	}
	if err != nil {
		return err
	}
	skillNode.ID = existing.ID
	return r.db.Save(skillNode).Error
}

type HideoutModuleRepository struct {
	db *DB
}

func NewHideoutModuleRepository(db *DB) *HideoutModuleRepository {
	return &HideoutModuleRepository{db: db}
}

func (r *HideoutModuleRepository) Create(hideoutModule *models.HideoutModule) error {
	return r.db.Create(hideoutModule).Error
}

func (r *HideoutModuleRepository) FindByID(id uint) (*models.HideoutModule, error) {
	var hideoutModule models.HideoutModule
	err := r.db.First(&hideoutModule, id).Error
	if err != nil {
		return nil, err
	}
	return &hideoutModule, nil
}

func (r *HideoutModuleRepository) FindByExternalID(externalID string) (*models.HideoutModule, error) {
	var hideoutModule models.HideoutModule
	err := r.db.Where("external_id = ?", externalID).First(&hideoutModule).Error
	if err != nil {
		return nil, err
	}
	return &hideoutModule, nil
}

func (r *HideoutModuleRepository) FindAll(offset, limit int) ([]models.HideoutModule, int64, error) {
	var hideoutModules []models.HideoutModule

	// Use DISTINCT ON to get unique records by external_id, keeping the one with lowest ID
	// PostgreSQL syntax: SELECT DISTINCT ON (external_id) * FROM ... ORDER BY external_id, id ASC
	// We use Raw() to execute the query, then scan into the model
	err := r.db.Raw(`
		SELECT DISTINCT ON (external_id) 
			id, external_id, name, description, max_level, levels, data, synced_at, created_at, updated_at
		FROM hideout_modules
		ORDER BY external_id, id ASC
		OFFSET ? LIMIT ?
	`, offset, limit).Scan(&hideoutModules).Error
	if err != nil {
		return nil, 0, err
	}

	// Count unique external_ids
	var count int64
	err = r.db.Raw(`SELECT COUNT(DISTINCT external_id) FROM hideout_modules`).Scan(&count).Error
	if err != nil {
		return nil, 0, err
	}

	return hideoutModules, count, nil
}

func (r *HideoutModuleRepository) Update(hideoutModule *models.HideoutModule) error {
	return r.db.Save(hideoutModule).Error
}

func (r *HideoutModuleRepository) Delete(id uint) error {
	return r.db.Delete(&models.HideoutModule{}, id).Error
}

func (r *HideoutModuleRepository) UpsertByExternalID(hideoutModule *models.HideoutModule) error {
	var existing models.HideoutModule
	err := r.db.Where("external_id = ?", hideoutModule.ExternalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(hideoutModule).Error
	}
	if err != nil {
		return err
	}
	hideoutModule.ID = existing.ID
	return r.db.Save(hideoutModule).Error
}

type EnemyTypeRepository struct {
	db *DB
}

func NewEnemyTypeRepository(db *DB) *EnemyTypeRepository {
	return &EnemyTypeRepository{db: db}
}

func (r *EnemyTypeRepository) Create(enemyType *models.EnemyType) error {
	return r.db.Create(enemyType).Error
}

func (r *EnemyTypeRepository) FindByID(id uint) (*models.EnemyType, error) {
	var enemyType models.EnemyType
	err := r.db.First(&enemyType, id).Error
	if err != nil {
		return nil, err
	}
	return &enemyType, nil
}

func (r *EnemyTypeRepository) FindByExternalID(externalID string) (*models.EnemyType, error) {
	var enemyType models.EnemyType
	err := r.db.Where("external_id = ?", externalID).First(&enemyType).Error
	if err != nil {
		return nil, err
	}
	return &enemyType, nil
}

func (r *EnemyTypeRepository) FindAll(offset, limit int) ([]models.EnemyType, int64, error) {
	var enemyTypes []models.EnemyType
	var count int64
	err := r.db.Model(&models.EnemyType{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&enemyTypes).Error
	return enemyTypes, count, err
}

func (r *EnemyTypeRepository) Update(enemyType *models.EnemyType) error {
	return r.db.Save(enemyType).Error
}

func (r *EnemyTypeRepository) Delete(id uint) error {
	return r.db.Delete(&models.EnemyType{}, id).Error
}

type AuditLogRepository struct {
	db *DB
}

func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditLogRepository) FindByFilters(apiKeyID, jwtTokenID, userID *uint, endpoint, method *string, startTime, endTime *string, offset, limit int) ([]models.AuditLog, int64, error) {
	query := r.db.Model(&models.AuditLog{})

	if apiKeyID != nil {
		query = query.Where("api_key_id = ?", *apiKeyID)
	}
	if jwtTokenID != nil {
		query = query.Where("jwt_token_id = ?", *jwtTokenID)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if endpoint != nil {
		query = query.Where("endpoint = ?", *endpoint)
	}
	if method != nil {
		query = query.Where("method = ?", *method)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	var logs []models.AuditLog
	err = query.Preload("APIKey").Preload("JWTToken").Preload("User").
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, count, err
}

// UserQuestProgressRepository handles user quest progress
type UserQuestProgressRepository struct {
	db *DB
}

func NewUserQuestProgressRepository(db *DB) *UserQuestProgressRepository {
	return &UserQuestProgressRepository{db: db}
}

func (r *UserQuestProgressRepository) Upsert(userID, questID uint, completed bool) (*models.UserQuestProgress, error) {
	var progress models.UserQuestProgress
	err := r.db.Where("user_id = ? AND quest_id = ?", userID, questID).First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		progress = models.UserQuestProgress{
			UserID:    userID,
			QuestID:   questID,
			Completed: completed,
		}
		err = r.db.Create(&progress).Error
		return &progress, err
	} else if err != nil {
		return nil, err
	}

	// Update existing
	progress.Completed = completed
	err = r.db.Save(&progress).Error
	return &progress, err
}

func (r *UserQuestProgressRepository) FindByUserID(userID uint) ([]models.UserQuestProgress, error) {
	var progress []models.UserQuestProgress
	err := r.db.Preload("Quest").Where("user_id = ?", userID).Order("id ASC").Find(&progress).Error
	return progress, err
}

func (r *UserQuestProgressRepository) FindByUserAndQuest(userID, questID uint) (*models.UserQuestProgress, error) {
	var progress models.UserQuestProgress
	err := r.db.Preload("Quest").Where("user_id = ? AND quest_id = ?", userID, questID).First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *UserQuestProgressRepository) Delete(userID, questID uint) error {
	return r.db.Where("user_id = ? AND quest_id = ?", userID, questID).Delete(&models.UserQuestProgress{}).Error
}

// UserHideoutModuleProgressRepository handles user hideout module progress
type UserHideoutModuleProgressRepository struct {
	db *DB
}

func NewUserHideoutModuleProgressRepository(db *DB) *UserHideoutModuleProgressRepository {
	return &UserHideoutModuleProgressRepository{db: db}
}

func (r *UserHideoutModuleProgressRepository) Upsert(userID, hideoutModuleID uint, unlocked bool, level int) (*models.UserHideoutModuleProgress, error) {
	var progress models.UserHideoutModuleProgress
	err := r.db.Where("user_id = ? AND hideout_module_id = ?", userID, hideoutModuleID).First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		progress = models.UserHideoutModuleProgress{
			UserID:          userID,
			HideoutModuleID: hideoutModuleID,
			Unlocked:        unlocked,
			Level:           level,
		}
		err = r.db.Create(&progress).Error
		return &progress, err
	} else if err != nil {
		return nil, err
	}

	// Update existing
	progress.Unlocked = unlocked
	progress.Level = level
	err = r.db.Save(&progress).Error
	return &progress, err
}

func (r *UserHideoutModuleProgressRepository) FindByUserID(userID uint) ([]models.UserHideoutModuleProgress, error) {
	var progress []models.UserHideoutModuleProgress
	err := r.db.Preload("HideoutModule").Where("user_id = ?", userID).Order("id ASC").Find(&progress).Error
	return progress, err
}

func (r *UserHideoutModuleProgressRepository) FindByUserAndModule(userID, hideoutModuleID uint) (*models.UserHideoutModuleProgress, error) {
	var progress models.UserHideoutModuleProgress
	err := r.db.Preload("HideoutModule").Where("user_id = ? AND hideout_module_id = ?", userID, hideoutModuleID).First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *UserHideoutModuleProgressRepository) Delete(userID, hideoutModuleID uint) error {
	return r.db.Where("user_id = ? AND hideout_module_id = ?", userID, hideoutModuleID).Delete(&models.UserHideoutModuleProgress{}).Error
}

// UserSkillNodeProgressRepository handles user skill node progress
type UserSkillNodeProgressRepository struct {
	db *DB
}

func NewUserSkillNodeProgressRepository(db *DB) *UserSkillNodeProgressRepository {
	return &UserSkillNodeProgressRepository{db: db}
}

func (r *UserSkillNodeProgressRepository) Upsert(userID, skillNodeID uint, unlocked bool, level int) (*models.UserSkillNodeProgress, error) {
	var progress models.UserSkillNodeProgress
	err := r.db.Where("user_id = ? AND skill_node_id = ?", userID, skillNodeID).First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		progress = models.UserSkillNodeProgress{
			UserID:      userID,
			SkillNodeID: skillNodeID,
			Unlocked:    unlocked,
			Level:       level,
		}
		err = r.db.Create(&progress).Error
		return &progress, err
	} else if err != nil {
		return nil, err
	}

	// Update existing
	progress.Unlocked = unlocked
	progress.Level = level
	err = r.db.Save(&progress).Error
	return &progress, err
}

func (r *UserSkillNodeProgressRepository) FindByUserID(userID uint) ([]models.UserSkillNodeProgress, error) {
	var progress []models.UserSkillNodeProgress
	err := r.db.Preload("SkillNode").Where("user_id = ?", userID).Order("id ASC").Find(&progress).Error
	return progress, err
}

func (r *UserSkillNodeProgressRepository) FindByUserAndSkillNode(userID, skillNodeID uint) (*models.UserSkillNodeProgress, error) {
	var progress models.UserSkillNodeProgress
	err := r.db.Preload("SkillNode").Where("user_id = ? AND skill_node_id = ?", userID, skillNodeID).First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *UserSkillNodeProgressRepository) Delete(userID, skillNodeID uint) error {
	return r.db.Where("user_id = ? AND skill_node_id = ?", userID, skillNodeID).Delete(&models.UserSkillNodeProgress{}).Error
}
