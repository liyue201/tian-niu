package repository

import "github.com/liyue201/tian-niu/pkg/model"

func (r *Repository) GetConversationMessages(conversationID string, limit int) ([]*model.ChatMessage, error) {
	var list []*model.ChatMessage
	query := r.db.Order("created_at desc")
	if conversationID != "" {
		query = query.Where("conversation_id = ?", conversationID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
