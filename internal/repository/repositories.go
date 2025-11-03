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

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
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
	err := r.db.Preload("User").Find(&keys).Error
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

type MissionRepository struct {
	db *DB
}

func NewMissionRepository(db *DB) *MissionRepository {
	return &MissionRepository{db: db}
}

func (r *MissionRepository) Create(mission *models.Mission) error {
	return r.db.Create(mission).Error
}

func (r *MissionRepository) FindByID(id uint) (*models.Mission, error) {
	var mission models.Mission
	err := r.db.First(&mission, id).Error
	if err != nil {
		return nil, err
	}
	return &mission, nil
}

func (r *MissionRepository) FindByExternalID(externalID string) (*models.Mission, error) {
	var mission models.Mission
	err := r.db.Where("external_id = ?", externalID).First(&mission).Error
	if err != nil {
		return nil, err
	}
	return &mission, nil
}

func (r *MissionRepository) FindAll(offset, limit int) ([]models.Mission, int64, error) {
	var missions []models.Mission
	var count int64
	err := r.db.Model(&models.Mission{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&missions).Error
	return missions, count, err
}

func (r *MissionRepository) Update(mission *models.Mission) error {
	return r.db.Save(mission).Error
}

func (r *MissionRepository) Delete(id uint) error {
	return r.db.Delete(&models.Mission{}, id).Error
}

func (r *MissionRepository) UpsertByExternalID(mission *models.Mission) error {
	var existing models.Mission
	err := r.db.Where("external_id = ?", mission.ExternalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(mission).Error
	}
	if err != nil {
		return err
	}
	mission.ID = existing.ID
	return r.db.Save(mission).Error
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
	var count int64
	err := r.db.Model(&models.HideoutModule{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.Order("id ASC").Offset(offset).Limit(limit).Find(&hideoutModules).Error
	return hideoutModules, count, err
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
