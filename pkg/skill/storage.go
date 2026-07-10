package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tianniu-ai/tianniu/pkg/storage"
)

const (
	skillPrefix = "skill:"
)

type StorageSkillStore struct {
	storage    storage.Storage
	hasDelete  bool
	hasList    bool
	deleteFunc func(ctx context.Context, key string) error
	listFunc   func(ctx context.Context, prefix string) ([]string, error)
}

func NewStorageSkillStore(s storage.Storage) *StorageSkillStore {
	store := &StorageSkillStore{storage: s}

	if adv, ok := s.(interface {
		Delete(ctx context.Context, key string) error
		List(ctx context.Context, prefix string) ([]string, error)
	}); ok {
		store.hasDelete = true
		store.hasList = true
		store.deleteFunc = adv.Delete
		store.listFunc = adv.List
	}

	return store
}

func (s *StorageSkillStore) GetAll() ([]*Skill, error) {
	if !s.hasList {
		return nil, fmt.Errorf("GetAll requires AdvancedStorage")
	}

	keys, err := s.listFunc(context.Background(), skillPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var skills []*Skill
	for _, key := range keys {
		data, err := s.storage.Load(context.Background(), key)
		if err != nil {
			return nil, fmt.Errorf("failed to load skill: %w", err)
		}

		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err != nil {
			return nil, fmt.Errorf("failed to unmarshal skill: %w", err)
		}

		skills = append(skills, &skill)
	}

	return skills, nil
}

func (s *StorageSkillStore) GetByID(id string) (*Skill, error) {
	systemKey := skillPrefix + "system_" + id
	data, err := s.storage.Load(context.Background(), systemKey)
	if err == nil && data != "" {
		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil {
			return &skill, nil
		}
	}

	keys, err := s.getSkillKeysByID(id)
	if err == nil {
		for _, key := range keys {
			data, err := s.storage.Load(context.Background(), key)
			if err != nil || data == "" {
				continue
			}

			var skill Skill
			if err := json.Unmarshal([]byte(data), &skill); err == nil && skill.ID == id {
				return &skill, nil
			}
		}
	}

	return nil, fmt.Errorf("skill not found: %s", id)
}

func (s *StorageSkillStore) GetByName(name string) (*Skill, error) {
	systemKey := skillPrefix + "system_" + name
	data, err := s.storage.Load(context.Background(), systemKey)
	if err == nil && data != "" {
		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil {
			return &skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

func (s *StorageSkillStore) GetByUserID(userID string) ([]*Skill, error) {
	if !s.hasList {
		return nil, fmt.Errorf("GetByUserID requires AdvancedStorage")
	}

	prefix := skillPrefix + "user_" + userID + "_"
	keys, err := s.listFunc(context.Background(), prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var skills []*Skill
	for _, key := range keys {
		data, err := s.storage.Load(context.Background(), key)
		if err != nil || data == "" {
			continue
		}

		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil {
			skills = append(skills, &skill)
		}
	}

	return skills, nil
}

func (s *StorageSkillStore) GetSystemSkills() ([]*Skill, error) {
	if !s.hasList {
		return nil, fmt.Errorf("GetSystemSkills requires AdvancedStorage")
	}

	prefix := skillPrefix + "system_"
	keys, err := s.listFunc(context.Background(), prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var skills []*Skill
	for _, key := range keys {
		data, err := s.storage.Load(context.Background(), key)
		if err != nil || data == "" {
			continue
		}

		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil {
			skills = append(skills, &skill)
		}
	}

	return skills, nil
}

func (s *StorageSkillStore) GetUserSkills(userID string) ([]*Skill, error) {
	return s.GetByUserID(userID)
}

func (s *StorageSkillStore) GetSkillForUser(userID, skillName string) (*Skill, error) {
	systemKey := skillPrefix + "system_" + skillName
	data, err := s.storage.Load(context.Background(), systemKey)
	if err == nil && data != "" {
		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil {
			return &skill, nil
		}
	}

	userKey := skillPrefix + "user_" + userID + "_" + skillName
	data, err = s.storage.Load(context.Background(), userKey)
	if err != nil || data == "" {
		return nil, fmt.Errorf("skill '%s' not found for user '%s'", skillName, userID)
	}

	var skill Skill
	if err := json.Unmarshal([]byte(data), &skill); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skill: %w", err)
	}

	return &skill, nil
}

func (s *StorageSkillStore) Save(skill *Skill) error {
	key := buildSkillKey(skill.Type, skill.UserID, skill.Name)

	data, err := json.Marshal(skill)
	if err != nil {
		return fmt.Errorf("failed to marshal skill: %w", err)
	}

	if err := s.storage.Store(context.Background(), key, string(data)); err != nil {
		return fmt.Errorf("failed to store skill: %w", err)
	}

	return nil
}

func (s *StorageSkillStore) Delete(id string) error {
	if !s.hasDelete {
		return fmt.Errorf("Delete requires AdvancedStorage")
	}

	keys, err := s.getSkillKeysByID(id)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return fmt.Errorf("skill not found: %s", id)
	}

	for _, key := range keys {
		if err := s.deleteFunc(context.Background(), key); err != nil {
			return fmt.Errorf("failed to delete skill: %w", err)
		}
	}

	return nil
}

func (s *StorageSkillStore) UpdateStatus(id string, status SkillStatus) error {
	skill, err := s.GetByID(id)
	if err != nil {
		return err
	}

	skill.Status = status
	return s.Save(skill)
}

func (s *StorageSkillStore) getSkillKeysByID(id string) ([]string, error) {
	if !s.hasList {
		return nil, fmt.Errorf("requires AdvancedStorage")
	}

	keys, err := s.listFunc(context.Background(), skillPrefix)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, key := range keys {
		data, err := s.storage.Load(context.Background(), key)
		if err != nil || data == "" {
			continue
		}

		var skill Skill
		if err := json.Unmarshal([]byte(data), &skill); err == nil && skill.ID == id {
			result = append(result, key)
		}
	}

	return result, nil
}

func buildSkillKey(skillType SkillType, userID, skillName string) string {
	if skillType == SkillTypeSystem {
		return skillPrefix + "system_" + skillName
	}
	return skillPrefix + "user_" + userID + "_" + skillName
}

func parseSkillKey(key string) (SkillType, string, string) {
	prefix := skillPrefix
	if !strings.HasPrefix(key, prefix) {
		return "", "", ""
	}

	remaining := strings.TrimPrefix(key, prefix)
	parts := strings.SplitN(remaining, "_", 2)
	if len(parts) < 2 {
		return "", "", ""
	}

	if parts[0] == "system" {
		return SkillTypeSystem, "", parts[1]
	} else if parts[0] == "user" {
		userParts := strings.SplitN(parts[1], "_", 2)
		if len(userParts) == 2 {
			return SkillTypeUser, userParts[0], userParts[1]
		}
	}

	return "", "", ""
}
