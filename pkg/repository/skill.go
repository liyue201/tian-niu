package repository

import (
	"errors"

	"github.com/tianniu-ai/tianniu/pkg/model"
	"gorm.io/gorm"
)

// GetAllSkills returns all skills
func (r *SQLStore) GetAllSkills() ([]*model.Skill, error) {
	var skills []model.Skill
	err := r.db.Find(&skills).Error
	if err != nil {
		return nil, err
	}
	result := make([]*model.Skill, 0, len(skills))
	for i := range skills {
		result = append(result, &skills[i])
	}
	return result, nil
}

// GetSkillByID returns a skill by ID
func (r *SQLStore) GetSkillByID(id string) (*model.Skill, error) {
	var s model.Skill
	err := r.db.Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("skill not found")
		}
		return nil, err
	}
	return &s, nil
}

// GetSkillByName returns a system skill by name
func (r *SQLStore) GetSkillByName(name string) (*model.Skill, error) {
	var s model.Skill
	err := r.db.Where("name = ? AND user_id = ?", name, "").First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("skill not found")
		}
		return nil, err
	}
	return &s, nil
}

// GetSkillsByUserID returns skills by user ID
func (r *SQLStore) GetSkillsByUserID(userID string) ([]*model.Skill, error) {
	var skills []model.Skill
	err := r.db.Where("user_id = ?", userID).Find(&skills).Error
	if err != nil {
		return nil, err
	}
	result := make([]*model.Skill, 0, len(skills))
	for i := range skills {
		result = append(result, &skills[i])
	}
	return result, nil
}

// GetSystemSkills returns all system skills
func (r *SQLStore) GetSystemSkills() ([]*model.Skill, error) {
	var skills []model.Skill
	err := r.db.Where("type = ?", "system").Find(&skills).Error
	if err != nil {
		return nil, err
	}
	result := make([]*model.Skill, 0, len(skills))
	for i := range skills {
		result = append(result, &skills[i])
	}
	return result, nil
}

// GetUserSkills returns user-specific skills
func (r *SQLStore) GetUserSkills(userID string) ([]*model.Skill, error) {
	var skills []model.Skill
	err := r.db.Where("type = ? AND user_id = ?", "user", userID).Find(&skills).Error
	if err != nil {
		return nil, err
	}
	result := make([]*model.Skill, 0, len(skills))
	for i := range skills {
		result = append(result, &skills[i])
	}
	return result, nil
}

// GetSkillForUser returns a skill for a specific user
func (r *SQLStore) GetSkillForUser(userID, skillName string) (*model.Skill, error) {
	var s model.Skill
	err := r.db.Where("name = ? AND (user_id = ? OR user_id = ?)", skillName, userID, "").First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("skill not found")
		}
		return nil, err
	}
	return &s, nil
}

// SaveSkill saves a skill to the database
func (r *SQLStore) SaveSkill(s *model.Skill) error {
	return r.db.Save(s).Error
}

// DeleteSkill deletes a skill by ID
func (r *SQLStore) DeleteSkill(id string) error {
	result := r.db.Where("id = ?", id).Delete(&model.Skill{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("skill not found")
	}
	return nil
}

// UpdateSkillStatus updates a skill's status
func (r *SQLStore) UpdateSkillStatus(id string, status string) error {
	result := r.db.Model(&model.Skill{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("skill not found")
	}
	return nil
}
